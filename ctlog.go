package ctlog

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"dario.cat/mergo"
	"github.com/root4loot/goutils/log"
)

type Runner struct {
	Options *Options
	client  *http.Client
	Results chan Result
	Visited map[string]bool
}

type Options struct {
	Concurrency int
	Timeout     int
	Delay       int
	DelayJitter int
	UserAgent   string
	Debug       bool
	HTTPClient  *http.Client
}

type Results struct {
	Results []Result
}

type Result struct {
	Query          string `json:"query"`
	Error          error  `json:"error"`
	IssuerCaID     int    `json:"issuer_ca_id"`
	IssuerName     string `json:"issuer_name"`
	CommonName     string `json:"common_name"`
	NameValue      string `json:"name_value"`
	ID             int64  `json:"id"`
	EntryTimestamp string `json:"entry_timestamp"`
	NotBefore      string `json:"not_before"`
	NotAfter       string `json:"not_after"`
	SerialNumber   string `json:"serial_number"`
}

var seen map[string]bool // map of seen domains

func init() {
	log.Init("ctlog")
}

func DefaultOptions() *Options {
	const timeout = 90 * time.Second

	return &Options{
		Concurrency: 3,
		Timeout:     int(timeout.Seconds()),
		Delay:       2,
		UserAgent:   "ctlog",
		Debug:       false,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				ForceAttemptHTTP2:     true,
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
				ResponseHeaderTimeout: timeout,
			},
			Timeout: timeout,
		},
	}
}

func NewRunner() *Runner {
	return newRunner(nil)
}

func NewRunnerWithOptions(options *Options) *Runner {
	return newRunner(options)
}

func NewRunnerWithDefaultOptions() *Runner {
	return newRunner(DefaultOptions())
}

func newRunner(options *Options) *Runner {
	defaultOptions := DefaultOptions()

	if options != nil {
		err := mergo.Merge(options, defaultOptions)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		options = defaultOptions
	}

	if options.Debug {
		log.SetLevel(log.DebugLevel)
	}

	return &Runner{
		Results: make(chan Result),
		Visited: make(map[string]bool),
		Options: options,
	}
}

func (r *Runner) Run(target string) (results []Result) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
	defer cancel()
	results = r.query(ctx, target)
	return uniqueResults(results)
}

func (r *Runner) RunMultiple(targets []string) (results [][]Result) {
	for _, target := range targets {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
		defer cancel()
		res := r.query(ctx, target)
		results = append(results, res)
	}
	return results
}

func (r *Runner) RunMultipleAsync(targets []string) {
	defer close(r.Results)
	sem := make(chan struct{}, r.Options.Concurrency)
	var wg sync.WaitGroup
	for _, target := range targets {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
		defer cancel()
		if !r.Visited[target] {
			r.Visited[target] = true

			sem <- struct{}{}
			wg.Add(1)
			go func(u string) {
				defer func() { <-sem }()
				defer wg.Done()
				results := r.query(ctx, u)
				for _, res := range results {
					res.Query = u
					r.Results <- res
				}
				time.Sleep(time.Millisecond * 100) // make room for processing results
			}(target)
			time.Sleep(r.getDelay() * time.Millisecond)
		}
	}
	wg.Wait()
}

func (r *Result) GetCommonName() (domain string) {
	domain = strings.Trim(r.CommonName, "*.")
	u, err := url.Parse("http://" + domain)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func (r *Result) GetMatchingIdentity() (domain string) {
	domain = strings.Trim(r.NameValue, "*.")
	u, err := url.Parse("http://" + domain)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func uniqueResults(results []Result) []Result {
	uniqueMap := make(map[Result]bool)
	var uniqueList []Result
	for _, res := range results {
		if !uniqueMap[res] {
			uniqueMap[res] = true
			uniqueList = append(uniqueList, res)
		}
	}
	return uniqueList
}

func (r *Runner) query(ctx context.Context, target string) (results []Result) {
	log.Infof("Querying %s", target)

	endpoint := "https://crt.sh/?q=" + url.QueryEscape(target) + "&output=json"
	seen = make(map[string]bool)

	maxRetries := 5
	retryDelay := 4 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			log.Errorf("%v", err.Error())
			return nil
		}

		if r.Options.UserAgent != "" {
			req.Header.Add("User-Agent", r.Options.UserAgent)
		}

		resp, err := r.Options.HTTPClient.Do(req)
		if err != nil {
			if isTimeoutError(err) {
				log.Errorf("timeout exceeded (%s) - retrying in %v", endpoint, retryDelay)
				time.Sleep(retryDelay)
				continue
			} else {
				log.Warnf("%v - %s", err.Error(), target)
				return nil
			}
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			log.Errorf("too many requests - retrying in %v (consider lowering concurrency)", retryDelay)
			time.Sleep(retryDelay)
			continue
		}

		if resp.StatusCode == http.StatusBadGateway {
			log.Errorf("bad gateway (%s) - retrying in %v", endpoint, retryDelay)
			time.Sleep(retryDelay)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyBytes, &results)

		for i := range results {
			if !seen[results[i].CommonName] {
				seen[results[i].CommonName] = true
				results[i].Query = target
				results = append(results, results[i])
			}
		}

		return results
	}

	log.Errorf("failed to get a successful response after %d attempts", maxRetries)
	return nil
}

func (r *Runner) getDelay() time.Duration {
	if r.Options.DelayJitter != 0 {
		return time.Duration(r.Options.Delay + rand.Intn(r.Options.DelayJitter))
	}
	return time.Duration(r.Options.Delay)
}

func isTimeoutError(err error) bool {
	e, ok := err.(net.Error)
	return ok && e.Timeout()
}

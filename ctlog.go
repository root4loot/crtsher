package ctlog

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/levels"
	log "github.com/root4loot/ctlog/pkg/log"
)

type Runner struct {
	Options *Options        // options for the runner
	client  *http.Client    // http client
	Results chan Result     // channel to receive results
	Visited map[string]bool // map of visited targets
}

// Options contains options for the runner
type Options struct {
	Concurrency int    // number of concurrent requests
	Timeout     int    // timeout in seconds
	Delay       int    // delay in seconds
	DelayJitter int    // delay jitter in seconds
	Verbose     bool   // hide info messages
	UserAgent   string // user agent
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

// DefaultOptions returns default options
func DefaultOptions() *Options {
	return &Options{
		Concurrency: 3,
		Timeout:     90,
		Delay:       2,
		DelayJitter: 1,
		UserAgent:   "ctlog",
		Verbose:     true,
	}
}

// NewRunner returns a new runner
func NewRunner() *Runner {
	options := DefaultOptions()
	return &Runner{
		Results: make(chan Result),
		Visited: make(map[string]bool),
		Options: options,
		client: &http.Client{
			Transport: &http.Transport{
				ForceAttemptHTTP2:     true,
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
				ResponseHeaderTimeout: time.Duration(options.Timeout) * time.Second,
			},
			Timeout: time.Duration(options.Timeout) * time.Second,
		},
	}
}

// Single runs ctlog against a single target and waits for results to be returned
func Single(target string) (results []Result) {
	r := NewRunner()
	r.Options.Concurrency = 1
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
	defer cancel()
	results = r.query(ctx, target, r.client)
	return
}

// Multiple runs ctlog against multiple targets and waits for results to be returned
// Allows for options to be optionally passed
func Multiple(targets []string, options ...Options) (results [][]Result) {
	r := NewRunner()

	if len(options) > 0 {
		r.Options = &options[0]
	}

	// limit concurrency to number of targets
	if r.Options.Concurrency > len(targets) {
		r.Options.Concurrency = len(targets)
	}

	for _, target := range targets {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
		defer cancel()
		res := r.query(ctx, target, r.client)
		results = append(results, res)
	}
	return
}

// MultipleStream runs ctlog against multiple targets and streams results to Results channel
func (r *Runner) MultipleStream(targets []string) {
	defer close(r.Results)

	// limit concurrency to number of targets
	if r.Options.Concurrency > len(targets) {
		r.Options.Concurrency = len(targets)
	}

	if r.Options.Verbose {
		gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
	}

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
				results := r.query(ctx, u, r.client)
				for _, res := range results {
					res.Query = u
					r.Results <- res
				}
				time.Sleep(time.Millisecond * 100) // make room for processing results
			}(target)
			time.Sleep(r.getDelay() * time.Millisecond) // delay between requests
		}
	}
	wg.Wait()
}

func (r *Result) Domain() (domain string) {
	domain = strings.Trim(r.CommonName, "*.")
	u, err := url.Parse("http://" + domain)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func (r *Runner) query(ctx context.Context, target string, client *http.Client) (results []Result) {
	endpoint := "https://crt.sh/?q=" + url.QueryEscape(target) + "&output=json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	seen = make(map[string]bool)

	if err != nil {
		log.Errorf("%v", err.Error())
		return nil
	}

	if r.Options.UserAgent != "" {
		req.Header.Add("User-Agent", r.Options.UserAgent)
	}

	// dump, _ := httputil.DumpRequestOut(req, true)
	// fmt.Println(string(dump))

	resp, err := client.Do(req)

	if err != nil {
		if isTimeoutError(err) {
			if errors.Is(err, context.DeadlineExceeded) {
				log.Errorf("timeout exceeded (%s) - trying again after 4 seconds", endpoint)
				time.Sleep(time.Millisecond * 4000) // wait some
				ctx2, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
				r.query(ctx2, target, client)
				cancel()
				return nil
			}
		} else {
			log.Warningf("%v - %s", err.Error(), target)
			return nil
		}
	} else {
		// try again if too many requests
		if resp.StatusCode == http.StatusTooManyRequests {
			log.Errorf("%s", "too many requests - wait and try again (consider lowering concurrency)")
			time.Sleep(time.Millisecond * 10000) // wait some
			ctx2, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
			r.query(ctx2, target, client)
			cancel()
			return nil
		}

		// try again if bad gateway
		if resp.StatusCode == http.StatusBadGateway {
			log.Errorf("bad gateway (%s) - trying again after 5 seconds", endpoint)
			time.Sleep(time.Millisecond * 5000) // wait some
			ctx2, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
			r.query(ctx2, target, client)
			cancel()
			return nil
		}

		bodyBytes, _ := ioutil.ReadAll(resp.Body)

		_ = json.Unmarshal(bodyBytes, &results)

		for i := range results {
			if !seen[results[i].CommonName] {
				seen[results[i].CommonName] = true
				results[i].Query = target
				results = append(results, results[i])
			}
		}
	}

	return
}

// delay returns total delay from options
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

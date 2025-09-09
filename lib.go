package crtsher

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/root4loot/goutils/log"
	_ "github.com/lib/pq"
)

type Runner struct {
	Options *Options
	Results chan Result
	Visited map[string]bool
}

type Options struct {
	Concurrency     int
	Timeout         int
	Delay           int
	DelayJitter     int
	UserAgent       string
	Debug           bool
	HTTPClient      *http.Client
	DatabaseURL     string
	PreferDatabase  bool
}

type Results struct {
	Results []Result
}

type Result struct {
	Query              string `json:"query"`
	Error              error  `json:"error"`
	IssuerCaID         int    `json:"issuer_ca_id"`
	IssuerName         string `json:"issuer_name"`
	IssuerOrganization string `json:"issuer_organization"`
	CommonName         string `json:"common_name"`
	NameValue          string `json:"name_value"`
	ID                 int64  `json:"id"`
	EntryTimestamp     string `json:"entry_timestamp"`
	NotBefore          string `json:"not_before"`
	NotAfter           string `json:"not_after"`
	SerialNumber       string `json:"serial_number"`
}

var seen map[string]bool

func init() {
	log.Init("crtsher")
}

func DefaultOptions() *Options {
	const timeout = 90 * time.Second

	return &Options{
		Concurrency:    3,
		Timeout:        int(timeout.Seconds()),
		Delay:          2,
		UserAgent:      "crtsher",
		Debug:          false,
		DatabaseURL:    "postgres://guest@crt.sh:5432/certwatch?sslmode=disable",
		PreferDatabase: true,
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

func (r *Result) GetCommonName() (domain string) {
	domain = strings.Trim(r.CommonName, "*.")
	u, _ := url.Parse("http://" + domain)
	return u.Hostname()
}

func (r *Result) GetMatchingIdentity() (domain string) {
	domain = strings.Trim(r.NameValue, "*.")
	u, _ := url.Parse("http://" + domain)
	return u.Hostname()
}

func (r *Runner) Query(target string) (results []Result) {
	log.Infof("Querying %s", target)
	seen = make(map[string]bool)

	if r.Options.PreferDatabase {
		log.Debugf("Attempting database query first")
		results = r.queryDatabase(target)
		if len(results) > 0 {
			log.Debugf("Database query returned %d results", len(results))
			return results
		}
		log.Debugf("Database query failed, falling back to HTTP")
	}

	return r.queryHTTP(target)
}

func (r *Runner) queryDatabase(target string) (results []Result) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", r.Options.DatabaseURL)
	if err != nil {
		log.Debugf("Failed to connect to database: %v", err)
		return nil
	}
	defer db.Close()

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(5 * time.Second)

	if err := db.PingContext(ctx); err != nil {
		log.Debugf("Database ping failed: %v", err)
		return nil
	}

	query := `SELECT cai.name_value
		FROM certificate_and_identities cai
		WHERE cai.name_value ILIKE ('%%' || $1 || '%%')
		ORDER BY cai.certificate_id DESC 
		LIMIT 1000`

	rows, err := db.QueryContext(ctx, query, target)
	if err != nil {
		log.Debugf("Database query failed: %v", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var domain string
		err := rows.Scan(&domain)
		if err != nil {
			log.Debugf("Failed to scan row: %v", err)
			continue
		}

		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}

		if !seen[domain] {
			seen[domain] = true
			result := Result{
				Query:      target,
				CommonName: domain,
				NameValue:  domain,
			}
			results = append(results, result)
		}
	}

	return results
}

func (r *Runner) queryHTTP(target string) (results []Result) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
	defer cancel()

	endpoint := "https://crt.sh/?q=" + url.QueryEscape(target) + "&output=json"

	maxRetries := 5
	retryDelay := time.Duration(r.Options.Timeout) * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		log.Debugf("Attempting request to endpoint: %s (attempt %d)", endpoint, attempt+1)
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
			if errors.Is(err, context.DeadlineExceeded) {
				log.Errorf("timeout exceeded (%s) - retrying in %v", endpoint, retryDelay)
				time.Sleep(retryDelay)
				continue
			} else {
				log.Warnf("%v - %s", err.Error(), target)
				return nil
			}
		}
		defer resp.Body.Close()

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

		time.Sleep(r.getDelay())

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

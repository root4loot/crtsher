package crtsher

import (
	"net/http"
	"testing"
	"time"
)

func TestNewRunner(t *testing.T) {
	runner := NewRunner()
	if runner.Options.Concurrency != 3 {
		t.Errorf("Expected Concurrency to be 3, got %d", runner.Options.Concurrency)
	}
	if runner.Options.Timeout != 90 {
		t.Errorf("Expected Timeout to be 90, got %d", runner.Options.Timeout)
	}
	if !runner.Options.PreferDatabase {
		t.Error("Expected PreferDatabase to be true by default")
	}
}

func TestQuery(t *testing.T) {
	runner := NewRunnerWithOptions(&Options{
		PreferDatabase: false, // Force HTTP mode for consistent testing
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	})

	domains := []string{"example.com", "hackerone.com"}
	success := false

	for _, domain := range domains {
		results := runner.Query(domain)
		if len(results) > 0 {
			success = true
			// Test that new fields are populated
			result := results[0]
			if result.Query != domain {
				t.Errorf("Expected Query to be %s, got %s", domain, result.Query)
			}
			break
		}
	}

	if !success {
		t.Error("Expected results for at least one domain, got none")
	}
}

func TestDatabaseFallback(t *testing.T) {
	runner := NewRunnerWithOptions(&Options{
		PreferDatabase: true,  // Try database first
		DatabaseURL:    "invalid-url", // Force database failure
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	})

	results := runner.Query("example.com")
	if len(results) == 0 {
		t.Error("Expected fallback to HTTP to work, got no results")
	}
}

func TestHTTPMode(t *testing.T) {
	runner := NewRunnerWithOptions(&Options{
		PreferDatabase: false, // HTTP only
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	})

	results := runner.Query("example.com")
	if len(results) == 0 {
		t.Error("Expected HTTP mode to return results")
	}

	// Test certificate information is populated
	if len(results) > 0 {
		result := results[0]
		if result.IssuerName == "" {
			t.Error("Expected IssuerName to be populated")
		}
		if result.SerialNumber == "" {
			t.Error("Expected SerialNumber to be populated")
		}
	}
}

func TestGetCommonName(t *testing.T) {
	result := Result{CommonName: "*.example.com"}
	expected := "example.com"
	if result.GetCommonName() != expected {
		t.Errorf("Expected %s, got %s", expected, result.GetCommonName())
	}
}

func TestGetMatchingIdentity(t *testing.T) {
	result := Result{NameValue: "*.example.com"}
	expected := "example.com"
	if result.GetMatchingIdentity() != expected {
		t.Errorf("Expected %s, got %s", expected, result.GetMatchingIdentity())
	}
}

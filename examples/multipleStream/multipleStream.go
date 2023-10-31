package main

import (
	"fmt"

	"github.com/root4loot/ctlog"
)

func main() {
	targets := []string{"Hackerone Inc", "example.com", "google.com"}

	// initialize runner
	runner := ctlog.NewRunner()

	runner.Options = &ctlog.Options{
		Concurrency: len(targets),
		Timeout:     90,
		Delay:       2,
		DelayJitter: 1,
		UserAgent:   "ctlog",
		Verbose:     true,
	}

	// process results
	go func() {
		for result := range runner.Results {
			if result.Domain() != "" {
				fmt.Println(result.Domain())
			}
		}
	}()

	// run ctlog against targets
	runner.MultipleStream(targets)
}

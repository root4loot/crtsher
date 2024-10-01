package main

import (
	"fmt"

	"github.com/root4loot/ctlog"
)

func main() {
	targets := []string{"Hackerone Inc", "example.com", "google.com"}

	runner := ctlog.NewRunner()

	runner.Options = &ctlog.Options{
		Concurrency: len(targets),
		Timeout:     90,
		Delay:       2,
		UserAgent:   "ctlog",
		Debug:       true,
	}

	go func() {
		for result := range runner.Results {
			if result.GetCommonName() != "" {
				fmt.Println(result.GetCommonName())
			}
		}
	}()

	runner.RunMultipleAsync(targets)
}

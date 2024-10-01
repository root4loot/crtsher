package main

import (
	"fmt"

	"github.com/root4loot/crtsher"
)

func main() {
	targets := []string{"Hackerone Inc", "example.com", "google.com"}

	runner := crtsher.NewRunner()

	runner.Options = &crtsher.Options{
		Concurrency: len(targets),
		Timeout:     90,
		Delay:       2,
		UserAgent:   "crtsher",
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

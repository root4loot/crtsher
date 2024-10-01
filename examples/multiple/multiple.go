package main

import (
	"fmt"

	"github.com/root4loot/crtsher"
)

func main() {
	options := &crtsher.Options{
		Concurrency: 2,
		Timeout:     90,
		Delay:       2,
		DelayJitter: 1,
		UserAgent:   "crtsher",
	}

	runner := crtsher.NewRunnerWithOptions(options)
	results := runner.RunMultiple([]string{"example.com", "Hackerone Inc"})
	for _, result := range results {
		for _, res := range result {
			if res.GetCommonName() != "" {
				fmt.Println(res.GetCommonName())
			}
		}
	}
}

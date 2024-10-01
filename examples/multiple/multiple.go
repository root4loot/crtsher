package main

import (
	"fmt"

	"github.com/root4loot/ctlog"
)

func main() {
	options := &ctlog.Options{
		Concurrency: 2,
		Timeout:     90,
		Delay:       2,
		DelayJitter: 1,
		UserAgent:   "ctlog",
	}

	runner := ctlog.NewRunnerWithOptions(options)
	results := runner.RunMultiple([]string{"example.com", "Hackerone Inc"})
	for _, result := range results {
		for _, res := range result {
			if res.GetCommonName() != "" {
				fmt.Println(res.GetCommonName())
			}
		}
	}
}

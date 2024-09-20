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

	results := ctlog.RunMultiple([]string{"example.com", "Hackerone Inc"}, *options)
	for _, result := range results {
		for _, res := range result {
			if res.Domain() != "" {
				fmt.Println(res.Domain())
			}
		}
	}
}

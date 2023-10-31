package main

import (
	"fmt"

	"github.com/root4loot/ctlog"
)

func main() {
	// With default options
	multiple()

	// With custom options
	multipleWithOptions(ctlog.Options{
		Concurrency: 2,
		Timeout:     90,
		Delay:       2,
		DelayJitter: 1,
		UserAgent:   "ctlog",
	})
}

func multiple() {
	results := ctlog.Multiple([]string{"example.com", "Hackerone Inc"})
	for _, result := range results {
		for _, res := range result {
			if res.Domain() != "" {
				fmt.Println(res.Domain())
			}
		}
	}
}

func multipleWithOptions(options ctlog.Options) {
	results := ctlog.Multiple([]string{"example.com", "Hackerone Inc"}, options)
	for _, result := range results {
		for _, res := range result {
			if res.Domain() != "" {
				fmt.Println(res.Domain())
			}
		}
	}
}

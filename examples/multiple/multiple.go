package main

import (
	"fmt"

	"github.com/root4loot/ctlog"
)

func main() {
	// run ctlog against targets
	results := ctlog.Multiple([]string{"example.com", "Hackerone Inc"})
	for _, result := range results {
		for _, res := range result {
			if res.Domain() != "" {
				fmt.Println(res.Domain())
			}
		}
	}
}

package main

import (
	"fmt"

	"github.com/root4loot/ctlog"
)

func main() {
	// run ctlog against single target
	results := ctlog.Single("example.com")
	for _, result := range results {
		if result.Domain() != "" {
			fmt.Println(result.Domain())
		}
	}
}

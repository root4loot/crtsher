package main

import (
	"fmt"

	"github.com/root4loot/ctlog"
)

func main() {
	targets := []string{"Hackerone Inc", "example.com", "google.com"}

	// initialize runner
	ctlog := ctlog.NewRunner()

	// process results
	go func() {
		for result := range ctlog.Results {
			if result.Domain() != "" {
				fmt.Println(result.Domain())
			}
		}
	}()

	// run ctlog against targets
	ctlog.MultipleStream(targets)
}

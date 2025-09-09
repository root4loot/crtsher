package main

import (
	"fmt"

	"github.com/root4loot/crtsher"
)

func main() {
	options := crtsher.DefaultOptions()
	options.Debug = true

	runner := crtsher.NewRunnerWithOptions(options)
	results := runner.Query("google.com")

	fmt.Printf("Found %d certificates for google.com\n", len(results))

	for i, result := range results {
		if i >= 5 {
			fmt.Printf("... and %d more\n", len(results)-5)
			break
		}
		fmt.Printf("%s (Issuer: %s)\n", result.GetCommonName(), result.IssuerName)
	}
}

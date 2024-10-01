package main

import (
	"fmt"

	"github.com/root4loot/crtsher"
)

func main() {
	runner := crtsher.NewRunner()

	results := runner.Run("example.com")
	for _, result := range results {
		if result.Domain() != "" {
			fmt.Println(result.Domain())
		}
	}
}

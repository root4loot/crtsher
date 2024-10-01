package main

import (
	"fmt"

	"github.com/root4loot/ctlog"
)

func main() {
	runner := ctlog.NewRunner()

	results := runner.Run("example.com")
	for _, result := range results {
		if result.Domain() != "" {
			fmt.Println(result.Domain())
		}
	}
}

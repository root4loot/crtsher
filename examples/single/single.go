package main

import (
	"fmt"

	"github.com/root4loot/ctlog"
)

func main() {
	results := ctlog.Run("example.com")
	for _, result := range results {
		if result.Domain() != "" {
			fmt.Println(result.Domain())
		}
	}
}

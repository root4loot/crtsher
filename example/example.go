package main

import (
	"fmt"

	"github.com/root4loot/crtsher"
)

func main() {
	runner := crtsher.NewRunner()

	results := runner.Query("example.com")
	for _, result := range results {
		fmt.Println(result.GetCommonName())
	}
}

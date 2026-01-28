package main

import "fmt"

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	fmt.Printf("tabs-cli %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
	fmt.Println("Coming soon! Implementation in progress.")
	fmt.Println("\nSee docs/ for complete design specifications.")
}

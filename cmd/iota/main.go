package main

import (
	"fmt"
	"github.com/redhill42/iota/api"
)

func main() {
	fmt.Println("This is the Iota server")
	fmt.Println()
	fmt.Printf("GitCommit: %s\n", api.GitCommit)
	fmt.Printf("Version:   %s\n", api.Version)
	fmt.Printf("BuildTime: %s\n", api.BuildTime)
}

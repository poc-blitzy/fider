package main

import (
	"log"
	"os"

	"github.com/getfider/fider/app/cmd"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// init loads environment variables before any other package initialization
// This ensures .env file is loaded before app/pkg/env tries to read config
func init() {
	// Skip loading .env if GO_ENV is set to "test" (test environment already loaded by godotenv -f .test.env)
	if os.Getenv("GO_ENV") != "test" {
		// Load .env file if it exists (ignore error if file doesn't exist)
		if err := godotenv.Load(); err != nil {
			log.Printf("Warning: Error loading .env file: %v", err)
		}
	}
}

func main() {

	args := os.Args[1:]
	if len(args) > 0 && args[0] == "ping" {
		os.Exit(cmd.RunPing())
	} else if len(args) > 0 && args[0] == "migrate" {
		os.Exit(cmd.RunMigrate())
	} else {
		os.Exit(cmd.RunServer())
	}
}

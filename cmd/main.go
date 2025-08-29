package main

import (
	"flag"
	"log"
	"os"

	"task-herald/internal/app"
)

func main() {
	cfgPath := flag.String("config", "", "Path to config.yaml (overrides env/ defaults)")
	flag.Parse()

	if err := app.Run(*cfgPath); err != nil {
		log.Println("Fatal error:", err)
		os.Exit(1)
	}
}

package main

import (
	"log"
	"os"

	"github.com/frigidplatypus/task-herald/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Println("Fatal error:", err)
		os.Exit(1)
	}
}

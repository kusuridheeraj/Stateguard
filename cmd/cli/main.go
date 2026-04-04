package main

import (
	"log"

	"github.com/kusuridheeraj/stateguard/internal/app"
)

func main() {
	if err := app.RunCLI(); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"log"

	"github.com/kusuridheeraj/stateguard/internal/app"
)

func main() {
	if err := app.RunDaemon(); err != nil {
		log.Fatal(err)
	}
}

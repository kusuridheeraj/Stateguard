package main

import (
	"log"
	"os"

	"github.com/kusuridheeraj/stateguard/internal/app"
)

func main() {
	if err := app.RunCLI(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		log.Fatal(err)
	}
}

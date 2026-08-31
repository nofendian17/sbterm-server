package main

import (
	"log"

	"github.com/nofendian17/sbterm/apps/core/internal/container"
)

func main() {
	if err := container.Run(); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"log"

	"github.com/nofendian17/sbterm-server/internal/container"
)

func main() {
	if err := container.Run(); err != nil {
		log.Fatal(err)
	}
}

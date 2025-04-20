package main

import (
	"fmt"
	"log"

	"github.com/jorgesanchez-e/simple-ddns/internal/config"
)

func main() {
	config, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(config)
}

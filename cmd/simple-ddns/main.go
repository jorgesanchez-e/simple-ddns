package main

import (
	"fmt"

	"github.com/jorgesanchez-e/simple-ddns/internal/config"
	"github.com/jorgesanchez-e/simple-ddns/internal/log"
)

func main() {
	log := log.New(log.Error)
	config, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(config)
}

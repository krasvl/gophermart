package main

import (
	"log"

	"github.com/krasvl/market/internal/server"
)

func main() {
	addrDefault := ""
	databaseDefault := ""
	secretDefault := ""

	server, err := server.GetConfiguredServer(addrDefault, databaseDefault, secretDefault)
	if err != nil {
		log.Fatalf("Server configure error: %v", err)
	}

	server.Start()
}

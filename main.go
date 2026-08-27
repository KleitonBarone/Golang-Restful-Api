package main

import (
	"log"
	"os"
)

const defaultListenAddress = "localhost:8080"

func main() {
	router := setupRouter()
	if err := router.Run(listenAddress()); err != nil {
		log.Fatal(err)
	}
}

func listenAddress() string {
	if address := os.Getenv("LISTEN_ADDRESS"); address != "" {
		return address
	}
	return defaultListenAddress
}

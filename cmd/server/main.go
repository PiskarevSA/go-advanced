package main

import (
	"log"
	"os"

	"github.com/PiskarevSA/go-advanced/internal/app/server"
)

func main() {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	server := server.NewServer()
	log.Println("[server] run")
	err := server.Run()
	if err != nil {
		log.Println(err)
		exitCode = 1
	}
	log.Println("[server] shutdown")
}

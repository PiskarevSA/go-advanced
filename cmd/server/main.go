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
	success := server.Run()
	if !success {
		exitCode = 1
	}
	log.Println("[server] gracefull shutdown")
}

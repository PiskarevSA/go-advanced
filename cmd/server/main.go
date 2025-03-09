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

	config, err := server.ReadConfig()
	if err != nil {
		log.Println(err)
		exitCode = 1
		return
	}

	server := server.NewServer()
	log.Println("[server] run")
	success := server.Run(config)
	if !success {
		exitCode = 1
	}
	log.Println("[server] gracefull shutdown")
}

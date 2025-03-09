package main

import (
	"log"
	"os"

	"github.com/PiskarevSA/go-advanced/internal/app/agent"
)

func main() {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	config, err := agent.ReadConfig()
	if err != nil {
		log.Println(err)
		exitCode = 1
		return
	}

	agent := agent.NewAgent()
	log.Println("[agent] run")
	success := agent.Run(config)
	if !success {
		exitCode = 1
	}
	log.Println("[agent] gracefull shutdown")
}

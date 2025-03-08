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

	agent := agent.NewAgent()
	log.Println("[agent] run")
	success := agent.Run()
	if !success {
		exitCode = 1
	}
	log.Println("[agent] gracefull shutdown")
}

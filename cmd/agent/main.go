package main

import (
	"os"

	"github.com/PiskarevSA/go-advanced/internal/app/agent"
	"github.com/PiskarevSA/go-advanced/internal/logger"
)

func main() {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	logger.Init()
	defer logger.Sync()

	config, err := agent.ReadConfig()
	if err != nil {
		logger.Plain.Error(err.Error())
		exitCode = 1
		return
	}

	agent := agent.NewAgent()
	logger.Plain.Info("[agent] run")
	success := agent.Run(config)
	if !success {
		exitCode = 1
	}
	logger.Plain.Info("[agent] gracefull shutdown")
}

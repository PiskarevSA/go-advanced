package main

import (
	"log/slog"
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
		slog.Error(err.Error())
		exitCode = 1
		return
	}

	agent := agent.NewAgent()
	slog.Info("[main] running agent")
	success := agent.Run(config)
	if !success {
		exitCode = 2
	}
	slog.Info("[main] gracefull shutdown complete")
}

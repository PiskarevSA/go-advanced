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
	slog.Info("[agent] run")
	success := agent.Run(config)
	if !success {
		exitCode = 1
	}
	slog.Info("[agent] gracefull shutdown")
}

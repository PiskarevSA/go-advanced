package main

import (
	"log/slog"
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
		slog.Error(err.Error())
		exitCode = 1
		return
	}

	server := server.NewServer()
	slog.Info("[server] run")
	success := server.Run(config)
	if !success {
		exitCode = 1
	}
	slog.Info("[server] gracefull shutdown")
}

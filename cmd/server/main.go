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

	server := server.NewServer(config)
	slog.Info("[main] running server")
	success := server.Run()
	if !success {
		exitCode = 2
	}
	slog.Info("[main] gracefull shutdown complete")
}

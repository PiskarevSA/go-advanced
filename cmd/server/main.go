package main

import (
	"os"

	"github.com/PiskarevSA/go-advanced/internal/app/server"
	"github.com/PiskarevSA/go-advanced/internal/logger"
)

func main() {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	logger.Init()
	defer logger.Sync()

	config, err := server.ReadConfig()
	if err != nil {
		logger.Plain.Error(err.Error())
		exitCode = 1
		return
	}

	server := server.NewServer()
	logger.Plain.Info("[server] run")
	success := server.Run(config)
	if !success {
		exitCode = 1
	}
	logger.Plain.Info("[server] gracefull shutdown")
}

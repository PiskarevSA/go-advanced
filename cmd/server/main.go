// Сервер для сбора метрик времени выполнения, который собирает отчеты от
// агентов по протоколу HTTP
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/PiskarevSA/go-advanced/internal/app/server"
)

var (
	buildVersion string = `N/A`
	buildDate    string = `N/A`
	buildCommit  string = `N/A`
)

func printVersion() {
	fmt.Println("Build version:", buildVersion)
	fmt.Println("Build date:", buildDate)
	fmt.Println("Build commit:", buildCommit)
}

func main() {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	printVersion()

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

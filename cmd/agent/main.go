// Агент, собирающий метрики и периодически отправляющий отчет на сервер по
// протоколу HTTP
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/PiskarevSA/go-advanced/internal/app/agent"
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

package main

import "github.com/PiskarevSA/go-advanced/internal/app/agent"

func main() {
	agent := agent.NewAgent()
	err := agent.Run()
	if err != nil {
		panic(err)
	}
}

package main

import (
	"fmt"

	"github.com/PiskarevSA/go-advanced/internal/app/agent"
)

func main() {
	agent := agent.NewAgent()
	fmt.Println("[agent] run")
	err := agent.Run()
	if err != nil {
		panic(err)
	}
	fmt.Println("[agent] shutdown")
}

package main

import (
	"fmt"

	"github.com/PiskarevSA/go-advanced/internal/app/agent"
)

func main() {
	agent := agent.NewAgent()
	// TODO PR #5
	// Вместо fmt лучше использовать log. Он потокобезопасный и со
	// свистелками-перделками по типу таймстемпа итд
	//
	// поправить надо во всём коде
	fmt.Println("[agent] run")
	err := agent.Run()
	if err != nil {
		// TODO PR #5
		// Вместо паник и os.Exit'ов всегда стоит использовать log.Fatal, паники
		// и os.Exit экстренно завершают приложения не позволяя выполнится
		// defer'ам => не получится graceful shutdown
		//
		// поправить надо во всём коде
		panic(err)
	}
	fmt.Println("[agent] shutdown")
}

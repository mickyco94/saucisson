package main

import (
	"os"

	"github.com/mickyco94/saucisson/internal/app"
)

func main() {
	if len(os.Args) != 2 {
		panic("Insufficient args")
	}

	err := app.Run(os.Args[1])
	if err != nil {
		panic(err)
	}
}

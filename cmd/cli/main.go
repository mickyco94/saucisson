package main

import (
	"context"
	"os"

	"github.com/mickyco94/saucisson/internal/app"
)

func main() {
	if len(os.Args) != 2 {
		panic("Insufficient args")
	}

	err := app.New(context.Background()).Run(os.Args[1])
	if err != nil {
		panic(err)
	}
}

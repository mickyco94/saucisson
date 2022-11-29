package main

import (
	"context"

	"github.com/mickyco94/saucisson/internal/app"
)

func main() {
	err := app.New(context.Background()).Run()
	if err != nil {
		panic(err)
	}
}

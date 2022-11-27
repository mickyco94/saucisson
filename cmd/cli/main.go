package main

import (
	"context"
	"log"

	"github.com/mickyco94/saucisson/internal/app"
)

func main() {
	err := app.New(context.Background()).Run()
	if err != nil {
		log.Panicf("err: %v", err)
	}
}

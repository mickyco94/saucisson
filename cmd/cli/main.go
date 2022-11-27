package main

import (
	"context"

	"github.com/mickyco94/saucisson/internal/app"
)

func main() {
	app.New(context.Background()).Run()
}

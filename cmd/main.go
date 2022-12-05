package main

import (
	"log"
	"os"
	"path"

	"github.com/mickyco94/saucisson/internal/app"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "saucisson",
		Usage: "Do X when Y",
		Flags: []cli.Flag{cli.BashCompletionFlag, &cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path of the saucisson definition YAML file. Defaults to ~/.saucisson.yml",
		}},
		Description: "Saucisson is a background service that uses provided configuration to run specified procedures when the specified condition(s) are met.",
		Action:      cli.ShowAppHelp,
		Commands: []*cli.Command{
			{
				Name: "run",
				Action: func(ctx *cli.Context) error {
					configPath := ctx.String("config")

					if configPath == "" {
						homedir := os.Getenv("HOME")
						configPath = path.Join(homedir, ".saucisson.yml")
					}

					err := app.Run(configPath)
					if err != nil {
						log.Printf(err.Error())
						return err
					}
					return nil
				},
			},
		},
	}

	app.Run(os.Args)
}

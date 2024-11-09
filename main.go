package main

import (
	"context"
	"log"
	"os"

	"github.com/AaronLieb/goat/commands/logs"
	"github.com/AaronLieb/goat/util"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:                   util.CommandName,
		Usage:                  "An aws cli wrapper written in Go",
		Version:                "v0.05",
		EnableShellCompletion:  true,
		UseShortOptionHandling: true,
		Commands: []*cli.Command{
			logs.Command(),
		},
	}

	if err := cmd.Run(context.TODO(), os.Args); err != nil {
		log.Fatal(err)
	}
}

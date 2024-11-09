package logs

import (
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "CloudWatch logs",
		Commands: []*cli.Command{
			GetSubcommand(),
		},
	}
}

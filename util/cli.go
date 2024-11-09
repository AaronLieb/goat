package util

import (
	"context"

	"github.com/urfave/cli/v3"
)

/*
	* If the user attempts to autocomplete after "--", it will treat the "--" as
	* "end of command options", and thus treat anything following this as an argument
	* rather than a flag, including the hidden "--generate-bash-completion" that is
	* passed when using autocomplete. This feature can't be disabled.
  *
  * This command will not require double dash usage, so this chunk is a crappy
	* way of removing this behavior
*/
func DisableDoubleDash(ctx context.Context, cmd *cli.Command) bool {
	autocomplete := cmd.NArg() > 0 && cmd.Args().Get(cmd.NArg()-1) == "--generate-shell-completion"
	if autocomplete {
		cmd.ShellComplete(ctx, cmd)
		cli.Exit("", 0)
		return true
	}
	return false
}

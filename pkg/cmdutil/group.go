package cmdutil

import (
	"fmt"

	"github.com/spf13/cobra"
)

// GroupCommand marks cmd as a container for subcommands.
//
// Cobra's default for a parent with no Run is to print help and exit 0, so a
// typo'd or unimplemented verb — `clickup task archive` — looks like it
// succeeded. For a CLI this heavily scripted, silent success on an unknown verb
// is the worst available default: it was mistaken for "the archive worked"
// during the 2026-07-27 list cleanup.
//
// With this applied, a bare `clickup task` still prints help and exits 0
// (the discoverable path), but `clickup task bogus` exits non-zero and names
// the offending verb.
func GroupCommand(cmd *cobra.Command) *cobra.Command {
	cmd.Args = cobra.ArbitraryArgs
	cmd.RunE = func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return c.Help()
		}
		return fmt.Errorf("unknown command %q for %q\n\nRun '%s --help' for usage",
			args[0], c.CommandPath(), c.CommandPath())
	}
	return cmd
}

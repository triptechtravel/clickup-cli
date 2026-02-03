package app

import (
	"fmt"

	"github.com/triptechtravel/clickup-cli/internal/iostreams"
	"github.com/triptechtravel/clickup-cli/pkg/cmd/root"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// Run bootstraps and executes the CLI. Returns exit code.
func Run() int {
	ios := iostreams.System()
	f := cmdutil.NewFactory(ios)

	rootCmd := root.NewCmdRoot(f)

	if err := rootCmd.Execute(); err != nil {
		if cmdutil.IsSilentError(err) {
			return 1
		}
		if cmdutil.IsAuthError(err) {
			fmt.Fprintln(ios.ErrOut, err.Error())
			return 4
		}
		fmt.Fprintln(ios.ErrOut, err.Error())
		return 1
	}

	return 0
}

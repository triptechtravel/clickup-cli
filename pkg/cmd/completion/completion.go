package completion

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewCmdCompletion creates the completion command that generates shell completion scripts.
func NewCmdCompletion(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for clickup.

To load completions:

Bash:
  $ source <(clickup completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ clickup completion bash > /etc/bash_completion.d/clickup
  # macOS:
  $ clickup completion bash > $(brew --prefix)/etc/bash_completion.d/clickup

Zsh:
  $ source <(clickup completion zsh)
  # To load completions for each session, execute once:
  $ clickup completion zsh > "${fpath[1]}/_clickup"

Fish:
  $ clickup completion fish | source
  # To load completions for each session, execute once:
  $ clickup completion fish > ~/.config/fish/completions/clickup.fish

PowerShell:
  PS> clickup completion powershell | Out-String | Invoke-Expression
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return rootCmd.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
	return cmd
}

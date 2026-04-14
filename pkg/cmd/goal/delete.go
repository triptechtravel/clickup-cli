package goal

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdGoalDelete returns the goal delete command.
func NewCmdGoalDelete(f *cmdutil.Factory) *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <goal-id>",
		Short: "Delete a goal",
		Long: `Delete a ClickUp goal permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.`,
		Example: `  # Delete a goal (with confirmation)
  clickup goal delete e53a33d0-2eb2-4664-a4b3-5e1b0df0e912

  # Delete without confirmation
  clickup goal delete e53a33d0-2eb2-4664-a4b3-5e1b0df0e912 --yes`,
		Args:    cobra.ExactArgs(1),
		PreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			goalID := args[0]
			ios := f.IOStreams
			cs := ios.ColorScheme()

			client, err := f.ApiClient()
			if err != nil {
				return err
			}

			if !confirm && ios.IsTerminal() {
				// Try to fetch the goal name for the prompt.
				msg := fmt.Sprintf("Delete goal %s?", goalID)
				resp, fetchErr := apiv2.GetGoal(context.Background(), client, goalID)
				if fetchErr == nil {
					msg = fmt.Sprintf("Delete goal %s (%s)?", cs.Bold(resp.Goal.Name), goalID)
				}

				p := prompter.New(ios)
				ok, err := p.Confirm(msg, false)
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(ios.ErrOut, "Cancelled.")
					return nil
				}
			}

			_, err = apiv2.DeleteGoal(context.Background(), client, goalID)
			if err != nil {
				return fmt.Errorf("failed to delete goal: %w", err)
			}

			fmt.Fprintf(ios.Out, "%s Goal deleted (%s)\n", cs.Green("!"), goalID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

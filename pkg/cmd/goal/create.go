package goal

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	clickupv2 "github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdGoalCreate returns the goal create command.
func NewCmdGoalCreate(f *cmdutil.Factory) *cobra.Command {
	var (
		name        string
		dueDate     int
		description string
		color       string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a goal",
		Long:  "Create a new goal in your ClickUp workspace.",
		Example: `  # Create a goal
  clickup goal create --name "Q1 Revenue Target" --description "Hit $1M ARR"

  # Create with a due date (Unix timestamp in ms)
  clickup goal create --name "Ship v2" --due-date 1704067200000`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			client, err := f.ApiClient()
			if err != nil {
				return err
			}

			cfg, err := f.Config()
			if err != nil {
				return err
			}

			teamID := cfg.Workspace
			if teamID == "" {
				return fmt.Errorf("no workspace configured. Run 'clickup auth' first")
			}

			req := &clickupv2.CreateGoalJSONRequest{
				Name:        name,
				DueDate:     dueDate,
				Description: description,
				Color:       color,
			}

			resp, err := apiv2.CreateGoal(context.Background(), client, teamID, req)
			if err != nil {
				return fmt.Errorf("failed to create goal: %w", err)
			}

			cs := f.IOStreams.ColorScheme()
			goalID := ""
			if m, ok := resp.Goal.(map[string]interface{}); ok {
				if id, ok := m["id"].(string); ok {
					goalID = id
				}
			}
			if goalID != "" {
				fmt.Fprintf(f.IOStreams.Out, "%s Goal %s created (id: %s)\n", cs.Green("!"), cs.Bold(name), goalID)
			} else {
				fmt.Fprintf(f.IOStreams.Out, "%s Goal %s created\n", cs.Green("!"), cs.Bold(name))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Goal name (required)")
	cmd.Flags().IntVar(&dueDate, "due-date", 0, "Due date (Unix timestamp in ms)")
	cmd.Flags().StringVar(&description, "description", "", "Goal description")
	cmd.Flags().StringVar(&color, "color", "", "Goal color hex")

	return cmd
}

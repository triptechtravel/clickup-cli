package member

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type memberEntry struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// NewCmdMemberList returns a command to list workspace members.
func NewCmdMemberList(f *cmdutil.Factory) *cobra.Command {
	var jsonFlags cmdutil.JSONFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspace members",
		Long: `List all members in the configured ClickUp workspace.

Displays each member's ID, username, email, and role. Member IDs
are useful for assigning tasks, adding watchers, and tagging users.`,
		Example: `  # List workspace members
  clickup member list

  # JSON output for scripting
  clickup member list --json

  # Get a specific member's ID
  clickup member list --json --jq '.[] | select(.username == "Isaac") | .id'`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMemberList(f, &jsonFlags)
		},
	}

	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}

func roleName(role int) string {
	switch role {
	case 1:
		return "owner"
	case 2:
		return "admin"
	case 3:
		return "member"
	case 4:
		return "guest"
	default:
		return strconv.Itoa(role)
	}
}

func runMemberList(f *cmdutil.Factory, jsonFlags *cmdutil.JSONFlags) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()
	ctx := context.Background()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	cfg, err := f.Config()
	if err != nil {
		return err
	}

	workspaceID := cfg.Workspace
	if workspaceID == "" {
		return fmt.Errorf("no workspace configured. Run 'clickup auth login' first")
	}

	teams, _, err := client.Clickup.Teams.GetTeams(ctx)
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	var entries []memberEntry
	for _, team := range teams {
		if team.ID != workspaceID {
			continue
		}
		for _, m := range team.Members {
			entries = append(entries, memberEntry{
				ID:       m.User.ID,
				Username: m.User.Username,
				Email:    m.User.Email,
				Role:     roleName(m.User.Role),
			})
		}
		break
	}

	if len(entries) == 0 {
		fmt.Fprintln(ios.Out, "No members found for this workspace.")
		return nil
	}

	if jsonFlags.WantsJSON() {
		return jsonFlags.OutputJSON(ios.Out, entries)
	}

	tp := tableprinter.New(ios)
	for _, e := range entries {
		tp.AddField(strconv.Itoa(e.ID))
		tp.AddField(e.Username)
		tp.AddField(e.Email)
		tp.AddField(cs.Gray(e.Role))
		tp.EndRow()
	}
	tp.Render()

	fmt.Fprintf(ios.Out, "\n%s\n", cs.Gray(fmt.Sprintf("%d members", len(entries))))

	return nil
}

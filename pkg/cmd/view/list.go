package view

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdViewList returns the view list command.
func NewCmdViewList(f *cmdutil.Factory) *cobra.Command {
	var (
		spaceID   string
		folderID  string
		listID    string
		teamFlag  string
		jsonFlags cmdutil.JSONFlags
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List views",
		Long: `List views at different levels of the ClickUp hierarchy.

Specify one of --space, --folder, --list, or --team to choose the scope.
Defaults to --team (workspace-level views) if none specified.`,
		Example: `  # List workspace-level views
  clickup view list --team

  # List views in a space
  clickup view list --space 12345

  # List views in a folder
  clickup view list --folder 67890

  # List views in a list
  clickup view list --list abc123`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.ApiClient()
			if err != nil {
				return err
			}

			cfg, err := f.Config()
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Collect raw views as JSON for uniform handling since view types are `any`.
			var rawViews []any
			var scope string

			switch {
			case spaceID != "":
				scope = "space"
				resp, err := apiv2.GetSpaceViews(ctx, client, spaceID)
				if err != nil {
					return fmt.Errorf("failed to fetch space views: %w", err)
				}
				for _, v := range resp.Views {
					rawViews = append(rawViews, v)
				}
			case folderID != "":
				scope = "folder"
				resp, err := apiv2.GetFolderViews(ctx, client, folderID)
				if err != nil {
					return fmt.Errorf("failed to fetch folder views: %w", err)
				}
				for _, v := range resp.Views {
					rawViews = append(rawViews, v)
				}
			case listID != "":
				scope = "list"
				resp, err := apiv2.GetListViews(ctx, client, listID)
				if err != nil {
					return fmt.Errorf("failed to fetch list views: %w", err)
				}
				for _, v := range resp.Views {
					rawViews = append(rawViews, v)
				}
			default:
				scope = "team"
				teamID := cfg.Workspace
				if strings.TrimSpace(teamFlag) != "" {
					teamID = strings.TrimSpace(teamFlag)
				}
				if teamID == "" {
					return fmt.Errorf("no workspace configured. Run 'clickup auth' first")
				}
				resp, err := apiv2.GetTeamViews(ctx, client, teamID)
				if err != nil {
					return fmt.Errorf("failed to fetch team views: %w", err)
				}
				for _, v := range resp.Views {
					rawViews = append(rawViews, v)
				}
			}

			if jsonFlags.WantsJSON() {
				return jsonFlags.OutputJSON(f.IOStreams.Out, rawViews)
			}

			if len(rawViews) == 0 {
				fmt.Fprintf(f.IOStreams.Out, "No views found for %s.\n", scope)
				return nil
			}

			tp := tableprinter.New(f.IOStreams)

			for _, v := range rawViews {
				m := toMap(v)
				tp.AddField(strVal(m, "id"))
				tp.AddField(strVal(m, "name"))
				tp.AddField(strVal(m, "type"))
				tp.EndRow()
			}

			if err := tp.Render(); err != nil {
				return err
			}

			cs := f.IOStreams.ColorScheme()
			fmt.Fprintln(f.IOStreams.Out)
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("---"))
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("Quick actions:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup view get <id>\n", cs.Gray("View:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup view tasks <id>\n", cs.Gray("Tasks:"))

			return nil
		},
	}

	cmd.Flags().StringVar(&spaceID, "space", "", "List views in a space")
	cmd.Flags().StringVar(&folderID, "folder", "", "List views in a folder")
	cmd.Flags().StringVar(&listID, "list", "", "List views in a list")
	cmd.Flags().StringVar(&teamFlag, "team", "", "Workspace ID override (or pass without value for default workspace)")
	cmd.Flags().Lookup("team").NoOptDefVal = " "

	// Default space from config.
	if s := os.Getenv("CLICKUP_SPACE"); s != "" && spaceID == "" {
		spaceID = s
	}

	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}

// toMap converts an any value to a map by round-tripping through JSON.
func toMap(v any) map[string]any {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

// strVal extracts a string value from a map.
func strVal(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

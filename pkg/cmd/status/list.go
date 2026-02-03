package status

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type listOptions struct {
	factory *cmdutil.Factory
	space   string
	json    cmdutil.JSONFlags
}

// listSpaceResponse represents the response from GET /space/{id} for the list command.
type listSpaceResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Statuses []struct {
		ID         string `json:"id"`
		Status     string `json:"status"`
		Color      string `json:"color"`
		Type       string `json:"type"`
		Orderindex int    `json:"orderindex"`
	} `json:"statuses"`
}

// statusEntry represents a single status for JSON output.
type statusEntry struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Color      string `json:"color"`
	Type       string `json:"type"`
	Orderindex int    `json:"orderindex"`
}

// NewCmdList returns the "status list" command.
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available statuses for a space",
		Long: `List all available statuses configured for a ClickUp space.

Uses the --space flag or falls back to the default space from configuration.`,
		Example: `  # List statuses for the default space
  clickup status list

  # List statuses for a specific space
  clickup status list --space 12345678

  # Output as JSON
  clickup status list --json`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.space, "space", "", "Space ID (defaults to configured space)")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func listRun(opts *listOptions) error {
	ios := opts.factory.IOStreams

	// Resolve space ID.
	spaceID := opts.space
	if spaceID == "" {
		cfg, err := opts.factory.Config()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Check for directory-specific space override.
		cwd, _ := os.Getwd()
		spaceID = cfg.SpaceForDir(cwd)

		if spaceID == "" {
			return fmt.Errorf("no space specified. Use --space flag or set a default with 'clickup config set space <id>'")
		}
	}

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Fetch space with statuses.
	spaceURL := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s", spaceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spaceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch space: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch space (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var spaceResp listSpaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&spaceResp); err != nil {
		return fmt.Errorf("failed to parse space response: %w", err)
	}

	if len(spaceResp.Statuses) == 0 {
		fmt.Fprintln(ios.ErrOut, "No statuses found for this space.")
		return nil
	}

	// JSON output.
	if opts.json.WantsJSON() {
		entries := make([]statusEntry, len(spaceResp.Statuses))
		for i, s := range spaceResp.Statuses {
			entries[i] = statusEntry{
				ID:         s.ID,
				Status:     s.Status,
				Color:      s.Color,
				Type:       s.Type,
				Orderindex: s.Orderindex,
			}
		}
		return opts.json.OutputJSON(ios.Out, entries)
	}

	// Table output.
	cs := ios.ColorScheme()
	tp := tableprinter.New(ios)

	for _, s := range spaceResp.Statuses {
		colorFn := cs.StatusColor(s.Status)
		tp.AddField(colorFn(s.Status))
		tp.AddField(s.Color)
		tp.AddField(s.Type)
		tp.EndRow()
	}

	fmt.Fprintf(ios.Out, "Showing %d statuses for space %s\n\n", len(spaceResp.Statuses), cs.Bold(spaceResp.Name))
	if err := tp.Render(); err != nil {
		return err
	}

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup status set <status> <task-id>\n", cs.Gray("Set:"))
	fmt.Fprintf(ios.Out, "  %s  clickup status list --json\n", cs.Gray("JSON:"))

	return nil
}

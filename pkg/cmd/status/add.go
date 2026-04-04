package status

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type addOptions struct {
	factory *cmdutil.Factory
	name    string
	color   string
	space   string
	confirm bool
}

// spaceResponse represents the full response from GET /space/{id}.
type spaceResponse struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Statuses []spaceStatus  `json:"statuses"`
}

// spaceStatus represents a single status in the space response.
type spaceStatus struct {
	ID         string `json:"id,omitempty"`
	Status     string `json:"status"`
	Color      string `json:"color"`
	Type       string `json:"type"`
	Orderindex int    `json:"orderindex"`
}

// NewCmdAdd returns the "status add" command.
func NewCmdAdd(f *cmdutil.Factory) *cobra.Command {
	opts := &addOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new status to a space",
		Long: `Add a new custom status to a ClickUp space.

The new status is inserted before the final "Closed"/"done" status in the
workflow ordering. Since statuses affect all tasks in a space, this command
requires interactive confirmation unless -y is passed.`,
		Example: `  # Add a "done" status to the default space
  clickup status add "done"

  # Add with a specific color
  clickup status add "QA Review" --color "#7C4DFF"

  # Skip confirmation prompt
  clickup status add "done" -y

  # Add to a specific space
  clickup status add "done" --space 12345`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			return addRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.color, "color", "", "Status color hex (e.g. \"#7C4DFF\"); omit to let ClickUp pick")
	cmd.Flags().StringVar(&opts.space, "space", "", "Space ID (defaults to configured space)")
	cmd.Flags().BoolVarP(&opts.confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func addRun(opts *addOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve space ID.
	spaceID := opts.space
	if spaceID == "" {
		cfg, err := opts.factory.Config()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
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

	// Fetch current space with statuses.
	spaceURL := client.URL("space/%s", spaceID)
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

	var space spaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&space); err != nil {
		return fmt.Errorf("failed to parse space response: %w", err)
	}

	// Check if status already exists.
	for _, s := range space.Statuses {
		if s.Status == opts.name {
			return fmt.Errorf("status %q already exists in space %s", opts.name, cs.Bold(space.Name))
		}
	}

	// Find the insertion point: before the last status (typically "Closed"/"done").
	// ClickUp spaces always end with a "closed" type status.
	insertIdx := len(space.Statuses)
	for i := len(space.Statuses) - 1; i >= 0; i-- {
		if space.Statuses[i].Type == "closed" {
			insertIdx = i
			break
		}
	}

	// Show current statuses and where the new one will go.
	fmt.Fprintf(ios.Out, "Space: %s\n\n", cs.Bold(space.Name))
	fmt.Fprintln(ios.Out, "Current statuses:")
	for i, s := range space.Statuses {
		colorFn := cs.StatusColor(s.Status)
		if i == insertIdx {
			fmt.Fprintf(ios.Out, "  %s  %s\n", cs.Green("→"), cs.Green(opts.name+" (new)"))
		}
		fmt.Fprintf(ios.Out, "  %s  %s\n", cs.Gray(fmt.Sprintf("%d.", i+1)), colorFn(s.Status))
	}
	// If insertIdx is at the end (no closed status found), show after all.
	if insertIdx == len(space.Statuses) {
		fmt.Fprintf(ios.Out, "  %s  %s\n", cs.Green("→"), cs.Green(opts.name+" (new)"))
	}
	fmt.Fprintln(ios.Out)

	// Require confirmation.
	if !opts.confirm {
		fmt.Fprintf(ios.Out, "This will add status %s to space %s.\n", cs.Bold(opts.name), cs.Bold(space.Name))
		fmt.Fprintf(ios.Out, "Statuses affect all tasks in the space. Continue? [y/N] ")

		var answer string
		fmt.Fscanln(ios.In, &answer)
		if answer != "y" && answer != "Y" {
			fmt.Fprintln(ios.Out, "Cancelled.")
			return nil
		}
	}

	// Build the updated statuses array.
	// ClickUp PUT /space/{id} expects statuses as an array of {status, color, orderindex}.
	type statusPayload struct {
		Status     string `json:"status"`
		Color      string `json:"color,omitempty"`
		Orderindex int    `json:"orderindex"`
	}

	var updatedStatuses []statusPayload
	orderIdx := 0
	for i, s := range space.Statuses {
		if i == insertIdx {
			newStatus := statusPayload{
				Status:     opts.name,
				Orderindex: orderIdx,
			}
			if opts.color != "" {
				newStatus.Color = opts.color
			}
			updatedStatuses = append(updatedStatuses, newStatus)
			orderIdx++
		}
		updatedStatuses = append(updatedStatuses, statusPayload{
			Status:     s.Status,
			Color:      s.Color,
			Orderindex: orderIdx,
		})
		orderIdx++
	}
	// If insertIdx was at the end.
	if insertIdx == len(space.Statuses) {
		newStatus := statusPayload{
			Status:     opts.name,
			Orderindex: orderIdx,
		}
		if opts.color != "" {
			newStatus.Color = opts.color
		}
		updatedStatuses = append(updatedStatuses, newStatus)
	}

	// PUT /space/{id} with updated statuses.
	payload := map[string]interface{}{
		"statuses": updatedStatuses,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, spaceURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}
	putReq.Header.Set("Content-Type", "application/json")

	putResp, err := client.HTTPClient.Do(putReq)
	if err != nil {
		return fmt.Errorf("failed to update space: %w", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		return fmt.Errorf("failed to update space statuses (HTTP %d): %s", putResp.StatusCode, string(body))
	}

	fmt.Fprintf(ios.Out, "\n%s Added status %s to space %s\n", cs.Green("!"), cs.Bold(opts.name), cs.Bold(space.Name))

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup status list\n", cs.Gray("List:"))
	fmt.Fprintf(ios.Out, "  %s  clickup status set %q <task-id>\n", cs.Gray("Use:"), opts.name)

	return nil
}

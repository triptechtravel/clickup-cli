package list

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type lsEntry struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Folder string `json:"folder"`
}

// NewCmdListLs returns the "list ls" command.
func NewCmdListLs(f *cmdutil.Factory) *cobra.Command {
	var jsonFlags cmdutil.JSONFlags
	var spaceID string

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List all lists in a space",
		Long: `Show all lists in the configured (or specified) space,
grouped by folder. Folderless lists are shown first.`,
		Example: `  # List all lists in the current space
  clickup list ls

  # List all lists in a specific space
  clickup list ls --space 12345`,
		PreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.ApiClient()
			if err != nil {
				return err
			}

			cfg, err := f.Config()
			if err != nil {
				return err
			}

			sid := spaceID
			if sid == "" {
				sid = cfg.Space
			}
			if sid == "" {
				return fmt.Errorf("no space configured. Run 'clickup space select' or use --space")
			}

			ctx := context.Background()
			_ = ctx // reserved for future use

			var entries []lsEntry

			// 1. Folderless lists
			folderlessLists, err := getFolderlessLists(client, sid)
			if err != nil {
				return fmt.Errorf("failed to fetch folderless lists: %w", err)
			}
			for _, l := range folderlessLists {
				entries = append(entries, lsEntry{ID: l.ID, Name: l.Name, Folder: ""})
			}

			// 2. Folders and their lists
			folders, err := getFolders(client, sid)
			if err != nil {
				return fmt.Errorf("failed to fetch folders: %w", err)
			}
			for _, folder := range folders {
				for _, l := range folder.Lists {
					entries = append(entries, lsEntry{ID: l.ID, Name: l.Name, Folder: folder.Name})
				}
			}

			if jsonFlags.WantsJSON() {
				return jsonFlags.OutputJSON(f.IOStreams.Out, entries)
			}

			if len(entries) == 0 {
				fmt.Fprintln(f.IOStreams.Out, "No lists found in this space.")
				return nil
			}

			cs := f.IOStreams.ColorScheme()
			tp := tableprinter.New(f.IOStreams)

			for _, e := range entries {
				folder := cs.Gray("(no folder)")
				if e.Folder != "" {
					folder = e.Folder
				}
				tp.AddField(e.ID)
				tp.AddField(e.Name)
				tp.AddField(folder)
				tp.EndRow()
			}

			if err := tp.Render(); err != nil {
				return err
			}

			fmt.Fprintln(f.IOStreams.Out)
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("---"))
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("Quick actions:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup task create --list-id <id>\n", cs.Gray("Create:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup task list --list-id <id>\n", cs.Gray("Tasks:"))

			return nil
		},
	}

	cmd.Flags().StringVar(&spaceID, "space", "", "Space ID (defaults to configured space)")
	cmdutil.AddJSONFlags(cmd, &jsonFlags)
	return cmd
}

// API response types

type apiList struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type apiFolder struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Lists []apiList `json:"lists"`
}

func getFolderlessLists(client *api.Client, spaceID string) ([]apiList, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s/list", spaceID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Lists []apiList `json:"lists"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Lists, nil
}

func getFolders(client *api.Client, spaceID string) ([]apiFolder, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s/folder", spaceID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Folders []apiFolder `json:"folders"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Folders, nil
}

package tag

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type createOptions struct {
	factory *cmdutil.Factory
	names   []string
	spaceID string
}

// NewCmdTagCreate returns the "tag create" command.
func NewCmdTagCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "create <name> [<name>...]",
		Short: "Create tags in a space",
		Long: `Create one or more tags in a ClickUp space.

If a tag already exists, it is skipped with a message. Uses the default space
from your config unless --space-id is provided.`,
		Example: `  # Create a single tag
  clickup tag create feat:search

  # Create multiple tags at once
  clickup tag create feat:search feat:maps fix:auth

  # Create in a specific space
  clickup tag create my-tag --space-id 12345678`,
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.names = args
			return runTagCreate(opts)
		},
	}

	cmd.Flags().StringVar(&opts.spaceID, "space-id", "", "Space ID (defaults to configured space)")

	return cmd
}

func runTagCreate(opts *createOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	spaceID := opts.spaceID
	if spaceID == "" {
		cfg, err := opts.factory.Config()
		if err != nil {
			return err
		}
		cwd, _ := os.Getwd()
		spaceID = cfg.SpaceForDir(cwd)
		if spaceID == "" {
			return fmt.Errorf("no space configured. Use --space-id or run 'clickup config set space <id>'")
		}
	}

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	// Fetch existing tags to check for duplicates.
	existing, err := cmdutil.FetchSpaceTags(client, spaceID)
	if err != nil {
		return fmt.Errorf("failed to fetch existing tags: %w", err)
	}

	existingSet := make(map[string]bool, len(existing))
	for _, t := range existing {
		existingSet[strings.ToLower(t)] = true
	}

	var created int
	for _, name := range opts.names {
		if existingSet[strings.ToLower(name)] {
			fmt.Fprintf(ios.Out, "%s Tag %q already exists, skipping\n", cs.Yellow("!"), name)
			continue
		}

		if err := cmdutil.CreateSpaceTag(client, spaceID, name); err != nil {
			return fmt.Errorf("failed to create tag %q: %w", name, err)
		}

		existingSet[strings.ToLower(name)] = true
		created++
		fmt.Fprintf(ios.Out, "%s Created tag %s\n", cs.Green("!"), cs.Bold(name))
	}

	if created == 0 {
		fmt.Fprintf(ios.Out, "\nAll tags already exist.\n")
	} else {
		fmt.Fprintf(ios.Out, "\n%s Created %d tag(s)\n", cs.Green("!"), created)
	}

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup tag list\n", cs.Gray("List:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task edit <id> --add-tags %s\n", cs.Gray("Use:"), opts.names[0])

	return nil
}

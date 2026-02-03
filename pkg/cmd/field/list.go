package field

import (
	"context"
	"fmt"
	"strings"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type listOptions struct {
	listID    string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdFieldList returns the "field list" command.
func NewCmdFieldList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List custom fields available in a list",
		Long: `Display all custom fields accessible in the specified ClickUp list.

Shows the field name, type, field ID (needed for API calls), and any
available options for dropdown or label fields.`,
		Example: `  # List custom fields for a specific list
  clickup field list --list-id 901234567

  # Output as JSON
  clickup field list --list-id 901234567 --json`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.listID == "" {
				return fmt.Errorf("required flag --list-id not set")
			}
			return runFieldList(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.listID, "list-id", "", "ClickUp list ID (required)")
	_ = cmd.MarkFlagRequired("list-id")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runFieldList(f *cmdutil.Factory, opts *listOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	fields, _, err := client.Clickup.CustomFields.GetAccessibleCustomFields(ctx, opts.listID)
	if err != nil {
		return fmt.Errorf("failed to fetch custom fields: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, fields)
	}

	if len(fields) == 0 {
		fmt.Fprintf(ios.Out, "No custom fields found for list %s\n", opts.listID)
		return nil
	}

	fmt.Fprintf(ios.Out, "%s for list %s:\n\n", cs.Bold("Custom Fields"), opts.listID)

	for _, field := range fields {
		fmt.Fprintf(ios.Out, "  %s %s\n", cs.Bold(field.Name), cs.Gray("("+field.Type+")"))
		fmt.Fprintf(ios.Out, "    ID: %s\n", field.ID)

		options := extractOptions(field)
		if len(options) > 0 {
			fmt.Fprintf(ios.Out, "    Options: %s\n", strings.Join(options, ", "))
		}

		fmt.Fprintln(ios.Out)
	}

	fmt.Fprintf(ios.Out, "%s\n", cs.Gray(fmt.Sprintf("Total: %d fields", len(fields))))

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task edit <id> --field \"Name=value\"\n", cs.Gray("Set:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task create --list-id %s --field \"Name=value\"\n", cs.Gray("Create:"), opts.listID)
	fmt.Fprintf(ios.Out, "  %s  clickup field list --list-id %s --json\n", cs.Gray("JSON:"), opts.listID)

	return nil
}

// extractOptions returns option names for dropdown and label fields.
func extractOptions(field clickup.CustomField) []string {
	tc, ok := field.TypeConfig.(map[string]interface{})
	if !ok {
		return nil
	}
	opts, ok := tc["options"].([]interface{})
	if !ok {
		return nil
	}
	var names []string
	for _, o := range opts {
		if m, ok := o.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok {
				names = append(names, name)
			}
		}
	}
	return names
}

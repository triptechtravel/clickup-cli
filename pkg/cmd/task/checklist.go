package task

import (
	"context"
	"fmt"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdChecklist returns the "task checklist" parent command.
func NewCmdChecklist(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checklist <command>",
		Short: "Manage task checklists",
		Long: `Add, remove, and manage checklists and their items on ClickUp tasks.

To find checklist and item IDs, use: clickup task view <task-id> --json`,
	}

	cmd.AddCommand(newCmdChecklistAdd(f))
	cmd.AddCommand(newCmdChecklistRemove(f))
	cmd.AddCommand(newCmdChecklistItem(f))

	return cmd
}

// --- Checklist Add ---

func newCmdChecklistAdd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <task-id> <checklist-name>",
		Short: "Create a checklist on a task",
		Example: `  clickup task checklist add 86abc123 "Deploy Steps"`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChecklistAdd(f, args[0], args[1])
		},
	}
	return cmd
}

func runChecklistAdd(f *cmdutil.Factory, taskID string, name string) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	checklist, _, err := client.Clickup.Checklists.CreateChecklist(ctx, taskID, nil, &clickup.ChecklistRequest{
		Name: name,
	})
	if err != nil {
		return fmt.Errorf("failed to create checklist: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Created checklist %s on task %s\n",
		cs.Green("!"), cs.Bold(checklist.Name), cs.Bold(taskID))
	fmt.Fprintf(ios.Out, "  Checklist ID: %s\n", checklist.ID)

	return nil
}

// --- Checklist Remove ---

func newCmdChecklistRemove(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <checklist-id>",
		Short: "Delete a checklist",
		Example: `  clickup task checklist remove b955c4dc-b8ee-4488-b0c1-example`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChecklistRemove(f, args[0])
		},
	}
	return cmd
}

func runChecklistRemove(f *cmdutil.Factory, checklistID string) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	_, err = client.Clickup.Checklists.DeleteChecklist(ctx, checklistID)
	if err != nil {
		return fmt.Errorf("failed to delete checklist: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Deleted checklist %s\n",
		cs.Green("!"), cs.Bold(checklistID))

	return nil
}

// --- Checklist Item ---

func newCmdChecklistItem(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "item <command>",
		Short: "Manage checklist items",
		Long:  "Add, resolve, and remove items from a checklist.",
	}

	cmd.AddCommand(newCmdChecklistItemAdd(f))
	cmd.AddCommand(newCmdChecklistItemResolve(f))
	cmd.AddCommand(newCmdChecklistItemRemove(f))

	return cmd
}

func newCmdChecklistItemAdd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <checklist-id> <item-name>",
		Short: "Add an item to a checklist",
		Example: `  clickup task checklist item add b955c4dc-example "Run migrations"`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChecklistItemAdd(f, args[0], args[1])
		},
	}
	return cmd
}

func runChecklistItemAdd(f *cmdutil.Factory, checklistID string, itemName string) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	_, _, err = client.Clickup.Checklists.CreateChecklistItem(ctx, checklistID, &clickup.ChecklistItemRequest{
		Name: itemName,
	})
	if err != nil {
		return fmt.Errorf("failed to add checklist item: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Added item %s to checklist\n",
		cs.Green("!"), cs.Bold(itemName))

	return nil
}

func newCmdChecklistItemResolve(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve <checklist-id> <item-id>",
		Short: "Mark a checklist item as resolved",
		Example: `  clickup task checklist item resolve b955c4dc-example 21e08dc8-example`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChecklistItemResolve(f, args[0], args[1])
		},
	}
	return cmd
}

func runChecklistItemResolve(f *cmdutil.Factory, checklistID string, itemID string) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	_, _, err = client.Clickup.Checklists.EditChecklistItem(ctx, checklistID, itemID, &clickup.ChecklistItemRequest{
		Resolved: true,
	})
	if err != nil {
		return fmt.Errorf("failed to resolve checklist item: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Marked item %s as resolved\n",
		cs.Green("!"), cs.Bold(itemID))

	return nil
}

func newCmdChecklistItemRemove(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <checklist-id> <item-id>",
		Short: "Remove an item from a checklist",
		Example: `  clickup task checklist item remove b955c4dc-example 21e08dc8-example`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChecklistItemRemove(f, args[0], args[1])
		},
	}
	return cmd
}

func runChecklistItemRemove(f *cmdutil.Factory, checklistID string, itemID string) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	_, err = client.Clickup.Checklists.DeleteChecklistItem(ctx, checklistID, itemID)
	if err != nil {
		return fmt.Errorf("failed to remove checklist item: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Removed item %s\n",
		cs.Green("!"), cs.Bold(itemID))

	return nil
}

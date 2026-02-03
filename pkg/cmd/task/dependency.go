package task

import (
	"context"
	"fmt"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdDependency returns the "task dependency" parent command.
func NewCmdDependency(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dependency <command>",
		Short: "Manage task dependencies",
		Long:  "Add or remove dependencies between ClickUp tasks.",
	}

	cmd.AddCommand(newCmdDependencyAdd(f))
	cmd.AddCommand(newCmdDependencyRemove(f))

	return cmd
}

type dependencyAddOptions struct {
	dependsOn   string
	blocks      string
}

func newCmdDependencyAdd(f *cmdutil.Factory) *cobra.Command {
	opts := &dependencyAddOptions{}

	cmd := &cobra.Command{
		Use:   "add <task-id>",
		Short: "Add a dependency to a task",
		Long: `Add a dependency relationship between two tasks.

Use --depends-on to indicate this task waits on another task.
Use --blocks to indicate this task blocks another task.`,
		Example: `  # This task depends on (waits for) another task
  clickup task dependency add 86abc123 --depends-on 86def456

  # This task blocks another task
  clickup task dependency add 86abc123 --blocks 86def456`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.dependsOn == "" && opts.blocks == "" {
				return fmt.Errorf("either --depends-on or --blocks must be specified")
			}
			return runDependencyAdd(f, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.dependsOn, "depends-on", "", "Task ID that this task depends on (waits for)")
	cmd.Flags().StringVar(&opts.blocks, "blocks", "", "Task ID that this task blocks")

	return cmd
}

func runDependencyAdd(f *cmdutil.Factory, taskID string, opts *dependencyAddOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	req := &clickup.AddDependencyRequest{}

	if opts.dependsOn != "" {
		req.DependsOn = opts.dependsOn
	}
	if opts.blocks != "" {
		req.DependencyOf = opts.blocks
	}

	_, err = client.Clickup.Dependencies.AddDependency(ctx, taskID, req, nil)
	if err != nil {
		return fmt.Errorf("failed to add dependency: %w", err)
	}

	if opts.dependsOn != "" {
		fmt.Fprintf(ios.Out, "%s Task %s now depends on %s\n",
			cs.Green("!"), cs.Bold(taskID), cs.Bold(opts.dependsOn))
	} else {
		fmt.Fprintf(ios.Out, "%s Task %s now blocks %s\n",
			cs.Green("!"), cs.Bold(taskID), cs.Bold(opts.blocks))
	}

	return nil
}

type dependencyRemoveOptions struct {
	dependsOn   string
	blocks      string
}

func newCmdDependencyRemove(f *cmdutil.Factory) *cobra.Command {
	opts := &dependencyRemoveOptions{}

	cmd := &cobra.Command{
		Use:   "remove <task-id>",
		Short: "Remove a dependency from a task",
		Long: `Remove a dependency relationship between two tasks.

Use --depends-on to remove a "waits for" relationship.
Use --blocks to remove a "blocks" relationship.`,
		Example: `  # Remove depends-on relationship
  clickup task dependency remove 86abc123 --depends-on 86def456

  # Remove blocks relationship
  clickup task dependency remove 86abc123 --blocks 86def456`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.dependsOn == "" && opts.blocks == "" {
				return fmt.Errorf("either --depends-on or --blocks must be specified")
			}
			return runDependencyRemove(f, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.dependsOn, "depends-on", "", "Task ID to remove depends-on relationship with")
	cmd.Flags().StringVar(&opts.blocks, "blocks", "", "Task ID to remove blocks relationship with")

	return cmd
}

func runDependencyRemove(f *cmdutil.Factory, taskID string, opts *dependencyRemoveOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	deleteOpts := &clickup.DeleteDependencyOptions{}

	if opts.dependsOn != "" {
		deleteOpts.DependsOn = opts.dependsOn
	}
	if opts.blocks != "" {
		deleteOpts.DependencyOf = opts.blocks
	}

	_, err = client.Clickup.Dependencies.DeleteDependency(ctx, taskID, deleteOpts)
	if err != nil {
		return fmt.Errorf("failed to remove dependency: %w", err)
	}

	if opts.dependsOn != "" {
		fmt.Fprintf(ios.Out, "%s Removed dependency: %s no longer depends on %s\n",
			cs.Green("!"), cs.Bold(taskID), cs.Bold(opts.dependsOn))
	} else {
		fmt.Fprintf(ios.Out, "%s Removed dependency: %s no longer blocks %s\n",
			cs.Green("!"), cs.Bold(taskID), cs.Bold(opts.blocks))
	}

	return nil
}

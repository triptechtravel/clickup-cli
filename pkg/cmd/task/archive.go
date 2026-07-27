package task

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type archiveOptions struct {
	cascade bool
	archive bool // false = unarchive
}

// NewCmdArchive returns the "task archive" command.
func NewCmdArchive(f *cmdutil.Factory) *cobra.Command {
	opts := &archiveOptions{archive: true}

	cmd := &cobra.Command{
		Use:   "archive <task-id>...",
		Short: "Archive one or more tasks",
		Long: `Archive tasks so they no longer appear in list views.

Archiving is reversible with 'clickup task unarchive'.

Subtasks are NOT archived with their parent. ClickUp does not cascade, so
archiving a parent silently leaves its children active. This command reports
any subtasks it left behind; pass --cascade to include them.`,
		Example: `  # Archive a task
  clickup task archive 86abc123

  # Archive several at once
  clickup task archive 86abc1 86abc2 86abc3

  # Archive a parent and all its subtasks
  clickup task archive 86abc123 --cascade`,
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArchive(f, opts, args)
		},
	}

	cmd.Flags().BoolVar(&opts.cascade, "cascade", false, "Also archive every subtask of the named tasks")
	return cmd
}

// NewCmdUnarchive returns the "task unarchive" command.
func NewCmdUnarchive(f *cmdutil.Factory) *cobra.Command {
	opts := &archiveOptions{archive: false}

	cmd := &cobra.Command{
		Use:   "unarchive <task-id>...",
		Short: "Restore one or more archived tasks",
		Long: `Restore archived tasks so they appear in list views again.

As with archiving, subtasks are not included unless --cascade is given.`,
		Example: `  # Restore a task
  clickup task unarchive 86abc123

  # Restore a parent and all its subtasks
  clickup task unarchive 86abc123 --cascade`,
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArchive(f, opts, args)
		},
	}

	cmd.Flags().BoolVar(&opts.cascade, "cascade", false, "Also restore every subtask of the named tasks")
	return cmd
}

func runArchive(f *cmdutil.Factory, opts *archiveOptions, ids []string) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()
	ctx := context.Background()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	// Custom task IDs (CU-abc123) need custom_task_ids + team_id on every
	// request, exactly as task edit does. Without this, archive silently only
	// worked for native IDs.
	cfg, err := f.Config()
	if err != nil {
		return err
	}

	verb, verbed := "archive", "Archived"
	if !opts.archive {
		verb, verbed = "unarchive", "Restored"
	}

	// A target carries whether its ID is a custom one: the update call needs
	// custom_task_ids + team_id, and the fallback path below can hand us a
	// custom ID when the task could not be read.
	type target struct {
		id       string
		isCustom bool
	}
	var (
		targets []target
		orphans []clickup.Task
		failed  int
	)

	// Resolve each named task, and either collect or report its subtasks.
	for _, raw := range ids {
		parsed := git.ParseTaskID(raw)
		qs := cmdutil.CustomIDTaskQuery(cfg, parsed.IsCustomID)
		if qs == "" {
			qs = "?include_subtasks=true"
		} else {
			qs += "&include_subtasks=true"
		}
		task, err := apiv2.GetTaskLocal(ctx, client, parsed.ID, qs)
		if err != nil {
			// Reading subtasks is best-effort: if it fails we still act on what
			// was named, but we must not pretend we checked.
			fmt.Fprintf(ios.ErrOut, "%s could not read subtasks of %s: %v\n",
				cs.Yellow("!"), parsed.ID, err)
			targets = append(targets, target{id: parsed.ID, isCustom: parsed.IsCustomID})
			continue
		}

		// task.ID is always the native ID, so no custom params are needed.
		targets = append(targets, target{id: task.ID})

		for _, sub := range task.Subtasks {
			// Only children whose state would actually change are interesting.
			if sub.Archived == opts.archive {
				continue
			}
			if opts.cascade {
				targets = append(targets, target{id: sub.ID})
			} else {
				orphans = append(orphans, sub)
			}
		}
	}

	for _, t := range targets {
		archived := opts.archive
		req := &clickupv2.UpdateTaskJSONRequest{Archived: &archived}

		var params []apiv2.UpdateTaskParams
		if t.isCustom {
			teamID, convErr := strconv.ParseFloat(cfg.Workspace, 64)
			if convErr != nil {
				fmt.Fprintf(ios.ErrOut, "%s cannot %s custom ID %s: no valid workspace configured\n",
					cs.Red("x"), verb, t.id)
				failed++
				continue
			}
			params = append(params, apiv2.UpdateTaskParams{CustomTaskIds: true, TeamId: teamID})
		}

		if _, err := apiv2.UpdateTask(ctx, client, t.id, req, params...); err != nil {
			fmt.Fprintf(ios.ErrOut, "%s failed to %s %s: %v\n", cs.Red("x"), verb, t.id, err)
			failed++
			continue
		}
		fmt.Fprintf(ios.ErrOut, "%s %s %s\n", cs.Green("✓"), verbed, t.id)
	}

	// The report that would have caught the three orphans left behind on
	// 2026-07-27. Loud, on stderr, names the IDs — a count alone is not
	// actionable, and silence is what let them survive.
	if len(orphans) > 0 {
		noun, was := "subtasks were", "them"
		if len(orphans) == 1 {
			noun, was = "subtask was", "it"
		}
		fmt.Fprintf(ios.ErrOut, "\n%s %d %s not %sd and remain active:\n",
			cs.Yellow("!"), len(orphans), noun, verb)
		for _, sub := range orphans {
			fmt.Fprintf(ios.ErrOut, "    %s  %s\n", sub.ID, sub.Name)
		}
		fmt.Fprintf(ios.ErrOut, "  Re-run with %s to include %s.\n", cs.Bold("--cascade"), was)
	}

	// Unarchiving cannot see what it is missing: ClickUp omits archived subtasks
	// from include_subtasks=true, so there is no way to enumerate or cascade to
	// them. Only say so when the user asked for a cascade — that is the moment
	// the limitation actually costs them something. Printing it on every
	// unarchive would make it wallpaper, and a warning nobody reads is the same
	// as no warning.
	if !opts.archive && opts.cascade {
		fmt.Fprintf(ios.ErrOut,
			"\n%s --cascade cannot restore archived subtasks: the API does not return\n"+
				"  them, so they are invisible to this command. Restore them by ID.\n",
			cs.Yellow("!"))
	}

	if failed > 0 {
		return fmt.Errorf("%d of %d tasks failed to %s", failed, len(targets), verb)
	}
	return nil
}

package apiv2

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
)

// LinkedScanProgress reports the shape of a workspace scan before it runs, so
// the caller can disclose the cost rather than appearing to hang.
type LinkedScanProgress struct {
	Lists int
}

// FindLinkedTasks returns tasks that appear in listID but are homed elsewhere.
//
// ClickUp has no endpoint for this. Neither GET /list/{id}/task nor
// GET /team/{id}/task?list_ids[] returns multi-list tasks — both filter by home
// list only (verified 2026-07-27; see api/GO_CLICKUP_GAPS.md). The only way to
// answer "what appears in this list" is to walk every list in the workspace and
// inspect each task's locations[].
//
// This is inherently O(workspace). onProgress is called once, before the task
// fetches begin, so the caller can say what it is about to do.
func FindLinkedTasks(
	ctx context.Context,
	client *api.Client,
	listID string,
	includeClosed bool,
	onProgress func(LinkedScanProgress),
) ([]clickup.Task, error) {
	teams, err := GetTeamsLocal(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate workspaces: %w", err)
	}

	// Collect every list id in the workspace, skipping the target itself.
	var listIDs []string
	seen := map[string]bool{listID: true}
	addList := func(id string) {
		if !seen[id] {
			seen[id] = true
			listIDs = append(listIDs, id)
		}
	}

	for _, team := range teams {
		spaces, err := GetSpacesLocal(ctx, client, team.ID, false)
		if err != nil {
			return nil, fmt.Errorf("failed to enumerate spaces: %w", err)
		}
		for _, space := range spaces {
			folderless, err := GetFolderlessListsLocal(ctx, client, space.ID, false)
			if err != nil {
				return nil, fmt.Errorf("failed to enumerate lists in space %s: %w", space.ID, err)
			}
			for _, l := range folderless {
				addList(l.ID)
			}

			folders, err := GetFoldersLocal(ctx, client, space.ID, false)
			if err != nil {
				return nil, fmt.Errorf("failed to enumerate folders in space %s: %w", space.ID, err)
			}
			for _, folder := range folders {
				for _, l := range folder.Lists {
					addList(l.ID)
				}
			}
		}
	}

	if onProgress != nil {
		onProgress(LinkedScanProgress{Lists: len(listIDs)})
	}

	qs := ""
	if includeClosed {
		qs = "?include_closed=true"
	}

	// Bounded concurrency: a workspace can hold hundreds of lists, and the
	// client's rate limiter handles backpressure.
	const workers = 8
	var (
		mu    sync.Mutex
		found []clickup.Task
		errs  []error
		wg    sync.WaitGroup
	)
	jobs := make(chan string)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				tasks, err := GetTasksLocal(ctx, client, id, qs)
				mu.Lock()
				if err != nil {
					// One unreadable list must not sink the scan; report at the end.
					errs = append(errs, fmt.Errorf("list %s: %w", id, err))
				} else {
					for _, task := range tasks {
						for _, loc := range task.Locations {
							if loc.ID == listID {
								found = append(found, task)
								break
							}
						}
					}
				}
				mu.Unlock()
			}
		}()
	}
	for _, id := range listIDs {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil, ctx.Err()
		case jobs <- id:
		}
	}
	close(jobs)
	wg.Wait()

	// Partial results are still useful, but silently returning them would repeat
	// the exact bug this scan exists to fix: presenting an incomplete view as a
	// complete one. Hand the failures back so the caller can say so.
	if len(errs) > 0 && len(found) == 0 {
		return nil, errs[0]
	}
	return found, errors.Join(errs...)
}

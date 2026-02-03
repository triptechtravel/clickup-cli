package comment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type addOptions struct {
	factory *cmdutil.Factory
	taskID  string
	body    string
	editor  bool
}

// NewCmdAdd returns the "comment add" command.
func NewCmdAdd(f *cmdutil.Factory) *cobra.Command {
	opts := &addOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "add [TASK] [BODY]",
		Short: "Add a comment to a task",
		Long: `Add a comment to a ClickUp task.

If TASK is not provided, the task ID is auto-detected from the current git branch.
If BODY is not provided (or --editor is used), your editor opens for composing the comment.

Use @username in the body to @mention workspace members. Usernames are resolved
against your workspace member list (see 'clickup member list') with case-insensitive
matching. Resolved mentions become real ClickUp @mentions that send notifications.`,
		Example: `  # Add a comment to the task detected from the current branch
  clickup comment add "" "Fixed the login bug"

  # Add a comment to a specific task
  clickup comment add abc123 "Deployed to staging"

  # Mention a teammate (triggers a real ClickUp notification)
  clickup comment add abc123 "Hey @Isaac can you review this?"

  # Mention multiple people
  clickup comment add abc123 "@Alice @Bob this is ready for QA"

  # Open your editor to compose the comment
  clickup comment add --editor`,
		Args:              cobra.MaximumNArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 1 {
				opts.taskID = args[0]
			}
			if len(args) >= 2 {
				opts.body = args[1]
			}
			return addRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.editor, "editor", "e", false, "Open editor to compose comment body")

	return cmd
}

func addRun(opts *addOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve task ID from git branch if not provided.
	taskID := opts.taskID
	if taskID == "" {
		gitCtx, err := opts.factory.GitContext()
		if err != nil {
			return fmt.Errorf("could not detect git context: %w\n\n%s", err,
				"Tip: provide the task ID as the first argument")
		}
		if gitCtx.TaskID == nil {
			return fmt.Errorf("%s", git.BranchNamingSuggestion(gitCtx.Branch))
		}
		taskID = gitCtx.TaskID.ID
		fmt.Fprintf(ios.ErrOut, "Detected task %s from branch %s\n", cs.Bold(taskID), cs.Cyan(gitCtx.Branch))
	}

	// Resolve comment body.
	body := opts.body
	if body == "" || opts.editor {
		p := prompter.New(ios)
		var err error
		body, err = p.Editor("Comment body", body, "*.md")
		if err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}
	}

	if body == "" {
		return fmt.Errorf("comment body cannot be empty")
	}

	// Build and send the API request.
	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.clickup.com/api/v2/task/%s/comment", taskID)

	// Build comment payload, resolving @mentions to real ClickUp user tags.
	var payload []byte
	if strings.Contains(body, "@") {
		if members, mErr := fetchWorkspaceMembers(opts.factory, client); mErr == nil && len(members) > 0 {
			if blocks, resolved := buildCommentBlocks(body, members); len(resolved) > 0 {
				for _, name := range resolved {
					fmt.Fprintf(ios.ErrOut, "Mentioning %s\n", cs.Bold("@"+name))
				}
				payload, err = json.Marshal(map[string]interface{}{"comment": blocks})
				if err != nil {
					return fmt.Errorf("failed to marshal comment: %w", err)
				}
			}
		}
	}
	if payload == nil {
		payload, err = json.Marshal(map[string]string{"comment_text": body})
		if err != nil {
			return fmt.Errorf("failed to marshal comment: %w", err)
		}
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.DoRequest(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	fmt.Fprintf(ios.Out, "%s Comment added to task %s\n", cs.Green("!"), cs.Bold(taskID))

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup comment list %s\n", cs.Gray("List:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task activity %s\n", cs.Gray("Activity:"), taskID)

	return nil
}

// mentionBlock represents a single block in a ClickUp structured comment.
type mentionBlock struct {
	Text string       `json:"text,omitempty"`
	Type string       `json:"type,omitempty"`
	User *mentionUser `json:"user,omitempty"`
}

// mentionUser represents a user reference in a structured comment tag.
type mentionUser struct {
	ID int `json:"id"`
}

// workspaceMember holds a member's display username and ClickUp user ID.
type workspaceMember struct {
	Username string
	ID       int
}

// fetchWorkspaceMembers returns a map of lowercase username to workspaceMember
// for all members in the configured workspace.
func fetchWorkspaceMembers(f *cmdutil.Factory, client *api.Client) (map[string]workspaceMember, error) {
	cfg, err := f.Config()
	if err != nil {
		return nil, err
	}
	if cfg.Workspace == "" {
		return nil, fmt.Errorf("no workspace configured")
	}

	ctx := context.Background()
	teams, _, err := client.Clickup.Teams.GetTeams(ctx)
	if err != nil {
		return nil, err
	}

	members := make(map[string]workspaceMember)
	for _, team := range teams {
		if team.ID != cfg.Workspace {
			continue
		}
		for _, m := range team.Members {
			members[strings.ToLower(m.User.Username)] = workspaceMember{
				Username: m.User.Username,
				ID:       m.User.ID,
			}
		}
		break
	}

	return members, nil
}

// isWordChar returns true if the byte is a letter, digit, or underscore.
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// buildCommentBlocks parses @mentions in body and returns structured comment
// blocks with resolved user tags. Returns the blocks and a list of resolved
// display usernames. If no mentions are resolved, both return values are nil.
func buildCommentBlocks(body string, members map[string]workspaceMember) ([]mentionBlock, []string) {
	// Sort members by username length (longest first) for greedy matching.
	type sortedMember struct {
		lower  string
		member workspaceMember
	}
	var sorted []sortedMember
	for lower, m := range members {
		sorted = append(sorted, sortedMember{lower, m})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].lower) > len(sorted[j].lower)
	})

	// Find all @mention positions in the body.
	type mentionPos struct {
		start    int
		end      int
		username string
		userID   int
	}

	bodyLower := strings.ToLower(body)
	var mentions []mentionPos

	for i := 0; i < len(body); i++ {
		if body[i] != '@' || i+1 >= len(body) {
			continue
		}
		afterAt := bodyLower[i+1:]
		for _, sm := range sorted {
			if !strings.HasPrefix(afterAt, sm.lower) {
				continue
			}
			// Check for word boundary after the username.
			endPos := i + 1 + len(sm.lower)
			if endPos < len(body) && isWordChar(body[endPos]) {
				continue
			}
			mentions = append(mentions, mentionPos{
				start:    i,
				end:      endPos,
				username: sm.member.Username,
				userID:   sm.member.ID,
			})
			i = endPos - 1 // -1 because loop increments
			break
		}
	}

	if len(mentions) == 0 {
		return nil, nil
	}

	// Build the structured comment blocks.
	var blocks []mentionBlock
	var resolved []string
	pos := 0

	for _, m := range mentions {
		if m.start > pos {
			blocks = append(blocks, mentionBlock{Text: body[pos:m.start]})
		}
		blocks = append(blocks, mentionBlock{
			Type: "tag",
			User: &mentionUser{ID: m.userID},
		})
		resolved = append(resolved, m.username)
		pos = m.end
	}

	if pos < len(body) {
		blocks = append(blocks, mentionBlock{Text: body[pos:]})
	}

	return blocks, resolved
}

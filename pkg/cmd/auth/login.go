package auth

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/auth"
	"github.com/triptechtravel/clickup-cli/internal/browser"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type loginOptions struct {
	factory   *cmdutil.Factory
	oauth     bool
	withToken bool
}

// NewCmdLogin returns the "auth login" command.
func NewCmdLogin(f *cmdutil.Factory) *cobra.Command {
	opts := &loginOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with ClickUp",
		Long: `Authenticate with a ClickUp account.

By default, this command prompts for a personal API token.
Get your token from ClickUp > Settings > ClickUp API > API tokens.

In non-interactive environments (CI), pipe a token via stdin:
  echo "pk_12345" | clickup auth login --with-token

To use OAuth instead (requires a registered OAuth app):
  clickup auth login --oauth`,
		Example: `  # Interactive token entry (default)
  clickup auth login

  # Pipe token for CI
  echo "pk_12345" | clickup auth login --with-token

  # Use OAuth browser flow
  clickup auth login --oauth`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return loginRun(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.oauth, "oauth", false, "Use OAuth browser flow (requires registered OAuth app)")
	cmd.Flags().BoolVar(&opts.withToken, "with-token", false, "Read token from standard input (for CI)")

	return cmd
}

func loginRun(opts *loginOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	var token string
	var method string

	switch {
	case opts.withToken:
		// Read token from stdin (CI mode).
		scanner := bufio.NewScanner(ios.In)
		if !scanner.Scan() {
			return fmt.Errorf("failed to read token from stdin")
		}
		token = strings.TrimSpace(scanner.Text())
		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}
		method = "token"

	case opts.oauth:
		// OAuth browser flow.
		cfg, err := opts.factory.Config()
		if err != nil {
			return err
		}

		clientID := auth.DefaultClientID
		clientSecret := auth.DefaultClientSecret

		// Allow config overrides (future extensibility).
		_ = cfg

		fmt.Fprintln(ios.Out, "Opening browser for ClickUp authorization...")
		authURL := auth.GetAuthURL(clientID, "")
		_ = browser.Open(authURL)

		var oauthErr error
		token, oauthErr = auth.OAuthFlow(clientID, clientSecret)
		if oauthErr != nil {
			return fmt.Errorf("OAuth flow failed: %w", oauthErr)
		}
		method = "oauth"

	default:
		// Interactive personal API token entry (default).
		fmt.Fprintln(ios.Out, "Get your API token from: ClickUp > Settings > ClickUp API > API tokens")
		fmt.Fprintln(ios.Out)

		p := prompter.New(ios)
		var err error
		token, err = p.Password("Paste your API token:")
		if err != nil {
			return fmt.Errorf("could not read token: %w", err)
		}
		token = strings.TrimSpace(token)
		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}
		method = "token"
	}

	// Validate the token.
	fmt.Fprintln(ios.Out, "Validating token...")
	user, err := auth.ValidateToken(token)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	// Store the token.
	if err := auth.StoreToken(token, method); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Logged in as %s (%s)\n",
		cs.Green("!"),
		cs.Bold(user.Username),
		user.Email,
	)

	// Select a workspace.
	if err := selectWorkspace(opts, token); err != nil {
		fmt.Fprintf(ios.ErrOut, "Warning: could not set workspace: %v\n", err)
		fmt.Fprintln(ios.ErrOut, "You can set it later with 'clickup config set workspace <id>'")
	}

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup space select\n", cs.Gray("Space:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task recent\n", cs.Gray("Recent:"))
	fmt.Fprintf(ios.Out, "  %s  clickup inbox\n", cs.Gray("Inbox:"))
	fmt.Fprintf(ios.Out, "  %s  clickup auth status\n", cs.Gray("Status:"))

	return nil
}

func selectWorkspace(opts *loginOptions, token string) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Create a fresh API client with the newly obtained token. We cannot use
	// the Factory's cached client because it was initialised before authentication.
	client := api.NewClient(token)

	teams, _, err := client.Clickup.Teams.GetTeams(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	if len(teams) == 0 {
		return fmt.Errorf("no workspaces found for this account")
	}

	cfg, err := opts.factory.Config()
	if err != nil {
		return err
	}

	if len(teams) == 1 {
		// Auto-select the only workspace.
		cfg.Workspace = teams[0].ID
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Fprintf(ios.Out, "%s Workspace set to %s (%s)\n",
			cs.Green("!"),
			cs.Bold(teams[0].Name),
			teams[0].ID,
		)
		return nil
	}

	// Multiple workspaces -- prompt the user to pick one.
	options := make([]string, len(teams))
	for i, t := range teams {
		options[i] = fmt.Sprintf("%s (%s)", t.Name, t.ID)
	}

	p := prompter.New(ios)
	idx, err := p.Select("Choose a default workspace:", options)
	if err != nil {
		return fmt.Errorf("workspace selection failed: %w", err)
	}

	cfg.Workspace = teams[idx].ID
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Workspace set to %s (%s)\n",
		cs.Green("!"),
		cs.Bold(teams[idx].Name),
		teams[idx].ID,
	)
	return nil
}

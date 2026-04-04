package cmdutil

import (
	"sync"

	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/auth"
	"github.com/triptechtravel/clickup-cli/internal/config"
	gitpkg "github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/iostreams"
)

// Factory provides lazy-initialized dependencies to commands.
type Factory struct {
	IOStreams *iostreams.IOStreams

	// Test overrides — when set, skip real initialization.
	apiClientOverride  *api.Client
	configOverride     *config.Config
	gitContextOverride *gitpkg.RepoContext

	configOnce sync.Once
	config     *config.Config
	configErr  error

	clientOnce sync.Once
	client     *api.Client
	clientErr  error

	gitOnce sync.Once
	gitCtx  *gitpkg.RepoContext
	gitErr  error
}

// NewFactory creates a new Factory with the given IOStreams.
func NewFactory(ios *iostreams.IOStreams) *Factory {
	return &Factory{
		IOStreams: ios,
	}
}

// Config returns the loaded configuration (cached after first call).
func (f *Factory) Config() (*config.Config, error) {
	if f.configOverride != nil {
		return f.configOverride, nil
	}
	f.configOnce.Do(func() {
		f.config, f.configErr = config.Load()
	})
	return f.config, f.configErr
}

// ApiClient returns an authenticated API client (cached after first call).
func (f *Factory) ApiClient() (*api.Client, error) {
	if f.apiClientOverride != nil {
		return f.apiClientOverride, nil
	}
	f.clientOnce.Do(func() {
		token, err := auth.GetToken()
		if err != nil {
			f.clientErr = err
			return
		}
		f.client = api.NewClient(token)
	})
	return f.client, f.clientErr
}

// GitContext returns the detected git context (cached after first call).
func (f *Factory) GitContext() (*gitpkg.RepoContext, error) {
	if f.gitContextOverride != nil {
		return f.gitContextOverride, nil
	}
	f.gitOnce.Do(func() {
		f.gitCtx, f.gitErr = gitpkg.DetectContext()
	})
	return f.gitCtx, f.gitErr
}

// SetAPIClient sets a test override for the API client.
func (f *Factory) SetAPIClient(c *api.Client) {
	f.apiClientOverride = c
}

// SetConfig sets a test override for the configuration.
func (f *Factory) SetConfig(c *config.Config) {
	f.configOverride = c
}

// SetGitContext sets a test override for the git context.
func (f *Factory) SetGitContext(c *gitpkg.RepoContext) {
	f.gitContextOverride = c
}

// GitClient returns a new git client.
func (f *Factory) GitClient() *gitpkg.Client {
	return gitpkg.NewClient()
}

package apiv2_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
)

// The contract generated flag sets must honour. Both cases below were real
// defects found reviewing the generator, and both fail silently or fatally
// rather than visibly, so they are worth pinning.

// A command that binds a field itself keeps full ownership of it. Apply must
// not write its own never-bound zero value over the command's real one.
func TestFlags_SkippedFieldIsNotApplied(t *testing.T) {
	cmd := &cobra.Command{Use: "x", Run: func(*cobra.Command, []string) {}}
	var handWritten string
	cmd.Flags().StringVar(&handWritten, "name", "", "hand-written, with real ergonomics")

	var f apiv2.UpdateTaskFlags
	f.Register(cmd, "name")

	cmd.SetArgs([]string{"--name", "a real name"})
	require.NoError(t, cmd.Execute())

	var req clickupv2.UpdateTaskJSONRequest
	f.Apply(&req)

	assert.Nil(t, req.Name,
		"Apply must skip fields Register did not bind; otherwise it overwrites the "+
			"command's value with an empty string")
	assert.Equal(t, "a real name", handWritten)
}

// pflag panics on a redefined flag, which would abort the whole CLI at startup
// rather than one command. Register must tolerate a name already in use.
func TestFlags_RegisterToleratesExistingFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	cmd.Flags().String("archived", "", "hand-written")

	var f apiv2.UpdateTaskFlags
	assert.NotPanics(t, func() { f.Register(cmd) },
		"registering over an existing flag must not panic")
}

// Applying without Register must be inert rather than panicking on a nil map.
func TestFlags_ApplyWithoutRegisterIsInert(t *testing.T) {
	var f apiv2.UpdateTaskFlags
	var req clickupv2.UpdateTaskJSONRequest
	assert.NotPanics(t, func() { f.Apply(&req) })
	assert.Nil(t, req.Archived)
}

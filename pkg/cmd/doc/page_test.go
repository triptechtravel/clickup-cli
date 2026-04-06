package doc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdPage_SubCommands(t *testing.T) {
	cmd := NewCmdPage(nil)

	assert.Equal(t, "page <command>", cmd.Use)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}

	assert.True(t, subNames["list"])
	assert.True(t, subNames["view"])
	assert.True(t, subNames["create"])
	assert.True(t, subNames["edit"])
}

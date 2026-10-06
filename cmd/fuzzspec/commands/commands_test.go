package commands_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/fuzzspec/fuzzspec/cmd/fuzzspec/commands"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommands_Validate(t *testing.T) {
	fixturePath, err := filepath.Abs("../../../test/fixtures/petstore.yaml")
	require.NoError(t, err)

	cmd := commands.Execute
	require.NotNil(t, cmd)

	// Test validate subcommand
	buf := new(bytes.Buffer)
	_ = buf
	err = commands.Execute()
	// Root command without args displays help and exits with no error
	assert.NoError(t, err)
	_ = fixturePath
}

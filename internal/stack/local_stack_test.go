package stack

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStack(t *testing.T) {
	t.Run("Load", func(t *testing.T) {
		config := localRemoteRepo(t)
		var localStack LocalStack
		require.NoError(t, localStack.Load(config))
		require.Len(t, localStack.Commits, 2)
		assert.Equal(t, "D", localStack.Commits[0].Oneline())
		assert.Equal(t, "Unique-D", localStack.Commits[0].UID)
		assert.Equal(t, "C", localStack.Commits[1].Oneline())
		assert.Equal(t, "", localStack.Commits[1].UID)
	})
}

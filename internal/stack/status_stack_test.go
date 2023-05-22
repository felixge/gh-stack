package stack

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatusStack(t *testing.T) {
	t.Run("Load", func(t *testing.T) {
		config := localRemoteRepo(t)
		var statusStack StatusStack
		require.NoError(t, statusStack.Load(config))
		fmt.Println(statusStack.String())
	})
}

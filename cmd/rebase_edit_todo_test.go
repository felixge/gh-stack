package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRewordCommits(t *testing.T) {
	msg := `pick 8c7a653 updates
pick c132f28 add update command
pick c053af6 hack hack hack

# a comment`

	expectedMsg := `reword 8c7a653 updates
pick c132f28 add update command
reword c053af6 hack hack hack

# a comment`

	// check simple happy path
	updatedMsg, err := rewordCommits(msg, "8c7a653", "c053af6")
	require.NoError(t, err)
	require.Equal(t, expectedMsg, updatedMsg)

	// check that commits match also match if msg contains a short version of them
	updatedMsg, err = rewordCommits(msg, "8c7a653123456", "c053af6")
	require.NoError(t, err)
	require.Equal(t, expectedMsg, updatedMsg)

	// check that non-matching commits produce an error
	_, err = rewordCommits(msg, "8c7a653", "c053af6", "unknownCommit")
	require.ErrorContains(t, err, "unknownCommit")
}

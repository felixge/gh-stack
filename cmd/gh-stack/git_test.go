package main

import (
	"errors"
	"testing"
)

func TestParseCommitUID(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedUID   string
		expectedError error
	}{
		{
			name: "Valid input",
			input: `This is a sample text.
Commit-UID: 123abc
Another line here.`,
			expectedUID:   "123abc",
			expectedError: nil,
		},
		{
			name: "No matching line",
			input: `This is a sample text.
Another line here.`,
			expectedUID:   "",
			expectedError: nil,
		},
		{
			name: "Multiple matching lines",
			input: `This is a sample text.
Commit-UID: 123abc
Commit-UID: 456def
Another line here.`,
			expectedUID:   "",
			expectedError: errors.New("multiple Commit-UID trailers"),
		},
		{
			name: "Valid input with different format",
			input: `Commit-UID: abc-123
This is a sample text.`,
			expectedUID:   "abc-123",
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			commitUID, err := ParseCommitUID(tc.input)

			if commitUID != tc.expectedUID || (err != nil && err.Error() != tc.expectedError.Error() || (err == nil && tc.expectedError != nil)) {
				t.Errorf("Expected UID: '%s', error: '%v', but got UID: '%s', error: '%v'",
					tc.expectedUID, tc.expectedError, commitUID, err)
			}
		})
	}
}

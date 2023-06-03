package cmd

import "testing"

func TestParseGitHubRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "HTTPS URL",
			remoteURL: "https://github.com/username/repo.git",
			wantOwner: "username",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "SSH URL",
			remoteURL: "git@github.com:username/repo.git",
			wantOwner: "username",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "Invalid URL",
			remoteURL: "invalidURL",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseGitHubRemoteURL(tt.remoteURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGitHubURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if owner != tt.wantOwner {
				t.Errorf("parseGitHubURL() owner = %v, want %v", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("parseGitHubURL() repo = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}

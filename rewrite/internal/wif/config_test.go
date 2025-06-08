package wif

import (
	"testing"
)

func TestConfigSetDefaults(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		wantPoolID string
		wantError  bool
	}{
		{
			name: "valid repository",
			config: Config{
				Project:    "my-project",
				Repository: "owner/repo",
			},
			wantPoolID: "owner-repo-pool",
			wantError:  false,
		},
		{
			name: "long repository name",
			config: Config{
				Project:    "my-project",
				Repository: "very-long-owner-name/very-long-repository-name-that-exceeds-limits",
			},
			wantError: false, // Should truncate, not error
		},
		{
			name: "invalid repository format",
			config: Config{
				Project:    "my-project",
				Repository: "invalid-repo-format",
			},
			wantError: true,
		},
		{
			name: "empty repository",
			config: Config{
				Project:    "my-project",
				Repository: "",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.SetDefaults()

			if tt.wantError && err == nil {
				t.Errorf("SetDefaults() expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("SetDefaults() unexpected error: %v", err)
			}

			if !tt.wantError && tt.wantPoolID != "" && tt.config.PoolID != tt.wantPoolID {
				t.Errorf("SetDefaults() PoolID = %v, want %v", tt.config.PoolID, tt.wantPoolID)
			}
		})
	}
}

func TestGetGitHubAudience(t *testing.T) {
	config := Config{
		Repository: "owner/repo",
	}

	expected := "https://github.com/owner"
	actual := config.GetGitHubAudience()

	if actual != expected {
		t.Errorf("GetGitHubAudience() = %v, want %v", actual, expected)
	}
}

func TestGeneratePoolID(t *testing.T) {
	tests := []struct {
		repository string
		want       string
	}{
		{"owner/repo", "owner-repo-pool"},
		{"owner_with_underscores/repo_name", "owner-with-underscores-repo-name-pool"},
		{"Owner/Repo", "owner-repo-pool"}, // Should be lowercase
	}

	for _, tt := range tests {
		t.Run(tt.repository, func(t *testing.T) {
			config := Config{Repository: tt.repository}
			actual := config.generatePoolID()

			if actual != tt.want {
				t.Errorf("generatePoolID() = %v, want %v", actual, tt.want)
			}

			// Ensure it meets length requirements
			if len(actual) < 3 || len(actual) > 32 {
				t.Errorf("generatePoolID() length %d not in range 3-32", len(actual))
			}
		})
	}
}

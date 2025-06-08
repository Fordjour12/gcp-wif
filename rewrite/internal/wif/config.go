package wif

import (
	"fmt"
	"regexp"
	"strings"
)

// Config holds the essential configuration for WIF setup
type Config struct {
	Project    string
	Repository string
	SAEmail    string
	PoolID     string
	ProviderID string
}

// SetDefaults validates input and sets smart defaults
func (c *Config) SetDefaults() error {
	// Validate repository format
	if err := c.validateRepository(); err != nil {
		return err
	}

	// Generate smart defaults
	if c.PoolID == "" {
		c.PoolID = c.generatePoolID()
	}

	if c.SAEmail == "" {
		c.SAEmail = c.generateServiceAccountEmail()
	}

	// Validate generated IDs
	if err := c.validateIDs(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateRepository() error {
	if c.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	if !strings.Contains(c.Repository, "/") {
		return fmt.Errorf("repository must be in format 'owner/repo'")
	}

	parts := strings.Split(c.Repository, "/")
	if len(parts) != 2 {
		return fmt.Errorf("repository must contain exactly one slash")
	}

	// Basic validation for GitHub username/org and repo name
	owner, repo := parts[0], parts[1]
	if owner == "" || repo == "" {
		return fmt.Errorf("both owner and repository name are required")
	}

	// Simple regex validation
	validName := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
	if !validName.MatchString(owner) || !validName.MatchString(repo) {
		return fmt.Errorf("invalid repository format")
	}

	return nil
}

func (c *Config) generatePoolID() string {
	// Convert "owner/repo" to "owner-repo-pool"
	clean := strings.ReplaceAll(c.Repository, "/", "-")
	clean = strings.ReplaceAll(clean, "_", "-")
	clean = strings.ToLower(clean)

	poolID := clean + "-pool"

	// Ensure it's within limits (3-32 chars)
	if len(poolID) > 32 {
		// Truncate intelligently
		if len(clean) > 27 { // 32 - len("-pool")
			clean = clean[:27]
		}
		poolID = clean + "-pool"
	}

	return poolID
}

func (c *Config) generateServiceAccountEmail() string {
	// Convert "owner/repo" to "github-owner-repo"
	clean := strings.ReplaceAll(c.Repository, "/", "-")
	clean = strings.ReplaceAll(clean, "_", "-")
	clean = strings.ToLower(clean)

	saName := "github-" + clean

	// Ensure service account name is valid (max 30 chars before @)
	if len(saName) > 30 {
		// Keep the pattern but truncate
		parts := strings.Split(clean, "-")
		if len(parts) >= 2 {
			// Try to keep owner and truncate repo
			owner := parts[0]
			if len(owner) > 20 {
				owner = owner[:20]
			}
			saName = "github-" + owner
		} else {
			saName = "github-" + clean[:20]
		}
	}

	return fmt.Sprintf("%s@%s.iam.gserviceaccount.com", saName, c.Project)
}

func (c *Config) validateIDs() error {
	// Validate Pool ID (3-32 chars, lowercase, hyphens, no start/end hyphens)
	if len(c.PoolID) < 3 || len(c.PoolID) > 32 {
		return fmt.Errorf("pool ID must be 3-32 characters")
	}

	poolRegex := regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
	if !poolRegex.MatchString(c.PoolID) {
		return fmt.Errorf("pool ID must start with lowercase letter, contain only lowercase letters/digits/hyphens, and not end with hyphen")
	}

	// Validate Provider ID
	if len(c.ProviderID) < 3 || len(c.ProviderID) > 32 {
		return fmt.Errorf("provider ID must be 3-32 characters")
	}

	if !poolRegex.MatchString(c.ProviderID) {
		return fmt.Errorf("provider ID must start with lowercase letter, contain only lowercase letters/digits/hyphens, and not end with hyphen")
	}

	return nil
}

// GetRepositoryOwner returns the owner part of the repository
func (c *Config) GetRepositoryOwner() string {
	parts := strings.Split(c.Repository, "/")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// GetGitHubAudience returns the GitHub-specific audience for OIDC
func (c *Config) GetGitHubAudience() string {
	return fmt.Sprintf("https://github.com/%s", c.GetRepositoryOwner())
}

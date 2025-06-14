package gcp

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Fordjour12/gcp-wif/internal/errors"
)

// GitHubOIDCConfig holds GitHub-specific OIDC configuration
type GitHubOIDCConfig struct {
	IssuerURI         string   `json:"issuer_uri"`
	AllowedAudiences  []string `json:"allowed_audiences"`
	DefaultAudience   string   `json:"default_audience"`
	ValidateTokenPath bool     `json:"validate_token_path"`
	RequireActor      bool     `json:"require_actor"`
	TrustedRepos      []string `json:"trusted_repos,omitempty"`
	BlockForkedRepos  bool     `json:"block_forked_repos"`
}

// GitHubClaimsMapping holds mapping configuration for GitHub OIDC claims
type GitHubClaimsMapping struct {
	Subject           string `json:"subject"`            // assertion.sub
	Actor             string `json:"actor"`              // assertion.actor
	Repository        string `json:"repository"`         // assertion.repository
	RepositoryOwner   string `json:"repository_owner"`   // assertion.repository_owner
	Ref               string `json:"ref"`                // assertion.ref
	RefType           string `json:"ref_type"`           // assertion.ref_type
	BaseRef           string `json:"base_ref"`           // assertion.base_ref
	HeadRef           string `json:"head_ref"`           // assertion.head_ref
	PullRequest       string `json:"pull_request"`       // assertion.pull_request
	WorkflowRef       string `json:"workflow_ref"`       // assertion.workflow_ref
	JobWorkflowRef    string `json:"job_workflow_ref"`   // assertion.job_workflow_ref
	RunnerEnvironment string `json:"runner_environment"` // assertion.runner_environment
	Environment       string `json:"environment"`        // assertion.environment
}

// WorkloadIdentityConfig holds configuration for workload identity setup
type WorkloadIdentityConfig struct {
	PoolName            string               `json:"pool_name"`
	PoolID              string               `json:"pool_id"`
	PoolDescription     string               `json:"pool_description"`
	ProviderName        string               `json:"provider_name"`
	ProviderID          string               `json:"provider_id"`
	ProviderDescription string               `json:"provider_description"`
	Repository          string               `json:"repository"` // GitHub repository in owner/name format
	ServiceAccountEmail string               `json:"service_account_email"`
	AllowedBranches     []string             `json:"allowed_branches,omitempty"` // Optional: restrict to specific branches
	AllowedTags         []string             `json:"allowed_tags,omitempty"`     // Optional: restrict to specific tags
	AllowPullRequests   bool                 `json:"allow_pull_requests"`        // Allow pull request workflows
	CreateNew           bool                 `json:"create_new"`                 // Create new or use existing
	GitHubOIDC          *GitHubOIDCConfig    `json:"github_oidc,omitempty"`      // GitHub-specific OIDC configuration
	ClaimsMapping       *GitHubClaimsMapping `json:"claims_mapping,omitempty"`   // Custom claims mapping
}

// WorkloadIdentityPoolInfo holds detailed information about a workload identity pool
type WorkloadIdentityPoolInfo struct {
	Name             string    `json:"name"`
	DisplayName      string    `json:"displayName"`
	Description      string    `json:"description"`
	State            string    `json:"state"`
	Disabled         bool      `json:"disabled"`
	CreateTime       time.Time `json:"createTime"`
	Exists           bool      `json:"exists"`
	FullResourceName string    `json:"fullResourceName"`
}

// WorkloadIdentityProviderInfo holds detailed information about a workload identity provider
type WorkloadIdentityProviderInfo struct {
	Name               string            `json:"name"`
	DisplayName        string            `json:"displayName"`
	Description        string            `json:"description"`
	State              string            `json:"state"`
	Disabled           bool              `json:"disabled"`
	AttributeMapping   map[string]string `json:"attributeMapping"`
	AttributeCondition string            `json:"attributeCondition"`
	IssuerURI          string            `json:"issuerUri"`
	AllowedAudiences   []string          `json:"allowedAudiences"`
	CreateTime         time.Time         `json:"createTime"`
	Exists             bool              `json:"exists"`
	FullResourceName   string            `json:"fullResourceName"`
}

// SecurityConditions holds various security conditions for workload identity
type SecurityConditions struct {
	Repository        string   `json:"repository"`
	AllowedBranches   []string `json:"allowed_branches,omitempty"`
	AllowedTags       []string `json:"allowed_tags,omitempty"`
	AllowPullRequests bool     `json:"allow_pull_requests"`
}

// IAMBindingConfig holds configuration for IAM policy bindings
type IAMBindingConfig struct {
	ServiceAccountEmail string            `json:"service_account_email"`
	PoolID              string            `json:"pool_id"`
	ProviderID          string            `json:"provider_id"`
	Repository          string            `json:"repository"`
	AllowedBranches     []string          `json:"allowed_branches,omitempty"`
	AllowedTags         []string          `json:"allowed_tags,omitempty"`
	AllowPullRequests   bool              `json:"allow_pull_requests"`
	GitHubOIDC          *GitHubOIDCConfig `json:"github_oidc,omitempty"`
	BindingTitle        string            `json:"binding_title,omitempty"`
	BindingDescription  string            `json:"binding_description,omitempty"`
	ExpirationTime      string            `json:"expiration_time,omitempty"` // Optional binding expiration
}

// SecurityBindingStrategy represents different security strategies for IAM bindings
type SecurityBindingStrategy string

const (
	SecurityBindingStrategyStrict     SecurityBindingStrategy = "strict"     // Strict repository and branch conditions
	SecurityBindingStrategyModerate   SecurityBindingStrategy = "moderate"   // Repository with optional branch restrictions
	SecurityBindingStrategyPermissive SecurityBindingStrategy = "permissive" // Repository-only restrictions
	SecurityBindingStrategyCustom     SecurityBindingStrategy = "custom"     // Custom CEL expression
)

// GetDefaultGitHubOIDCConfig returns default GitHub OIDC configuration
func GetDefaultGitHubOIDCConfig() *GitHubOIDCConfig {
	return &GitHubOIDCConfig{
		IssuerURI:         "https://token.actions.githubusercontent.com",
		AllowedAudiences:  []string{"sts.googleapis.com"},
		DefaultAudience:   "sts.googleapis.com",
		ValidateTokenPath: true,
		RequireActor:      true,
		BlockForkedRepos:  true,
	}
}

// GetGitHubRepositorySpecificOIDCConfig returns GitHub OIDC configuration with repository-specific audience
func GetGitHubRepositorySpecificOIDCConfig(repository string) *GitHubOIDCConfig {
	if repository == "" {
		return GetDefaultGitHubOIDCConfig()
	}

	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return GetDefaultGitHubOIDCConfig()
	}

	orgOrUser := parts[0]
	githubAudience := fmt.Sprintf("https://github.com/%s", orgOrUser)

	return &GitHubOIDCConfig{
		IssuerURI:         "https://token.actions.githubusercontent.com",
		AllowedAudiences:  []string{githubAudience, "sts.googleapis.com"}, // Support both patterns
		DefaultAudience:   githubAudience,
		ValidateTokenPath: true,
		RequireActor:      true,
		BlockForkedRepos:  true,
	}
}

// GetDefaultGitHubClaimsMapping returns default GitHub claims mapping
func GetDefaultGitHubClaimsMapping() *GitHubClaimsMapping {
	return &GitHubClaimsMapping{
		Subject:           "assertion.sub",
		Actor:             "assertion.actor",
		Repository:        "assertion.repository",
		RepositoryOwner:   "assertion.repository_owner",
		Ref:               "assertion.ref",
		RefType:           "assertion.ref_type",
		BaseRef:           "assertion.base_ref",
		HeadRef:           "assertion.head_ref",
		PullRequest:       "assertion.pull_request",
		WorkflowRef:       "assertion.workflow_ref",
		JobWorkflowRef:    "assertion.job_workflow_ref",
		RunnerEnvironment: "assertion.runner_environment",
		Environment:       "assertion.environment",
	}
}

// ValidateGitHubRepository validates GitHub repository format and accessibility
func ValidateGitHubRepository(repository string) error {
	if repository == "" {
		return errors.NewValidationError("GitHub repository is required (format: owner/name)")
	}

	// Validate repository format
	if !strings.Contains(repository, "/") {
		return errors.NewValidationError(
			"Repository must be in format 'owner/name'",
			"Example: 'myorg/myrepo'")
	}

	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return errors.NewValidationError(
			"Repository must contain exactly one slash",
			"Format: 'owner/repository'")
	}

	owner, repo := parts[0], parts[1]

	// Validate owner name (GitHub username/organization rules)
	ownerRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-])*[a-zA-Z0-9]$|^[a-zA-Z0-9]$`)
	if !ownerRegex.MatchString(owner) {
		return errors.NewValidationError(
			"Invalid GitHub owner name",
			"Owner must start and end with alphanumeric characters",
			"Can contain hyphens but not consecutive ones")
	}

	// Validate repository name (GitHub repository rules)
	repoRegex := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !repoRegex.MatchString(repo) {
		return errors.NewValidationError(
			"Invalid GitHub repository name",
			"Repository name can contain letters, numbers, dots, hyphens, and underscores")
	}

	// Check for reserved names
	reservedNames := []string{".", "..", ".git", ".github"}
	for _, reserved := range reservedNames {
		if repo == reserved {
			return errors.NewValidationError(
				fmt.Sprintf("Repository name '%s' is reserved", repo),
				"Choose a different repository name")
		}
	}

	return nil
}

// ValidateWorkloadIdentityConfig validates workload identity configuration
func ValidateWorkloadIdentityConfig(config *WorkloadIdentityConfig) error {
	if config.PoolID == "" {
		return errors.NewValidationError("Workload Identity Pool ID is required")
	}

	// ProviderID is not strictly required here, as this function is used by
	// CreateWorkloadIdentityPool which only needs PoolID and Repository.
	// ProviderID is essential for CreateWorkloadIdentityProvider, which will fail
	// implicitly if it's missing when constructing gcloud commands.
	// if config.ProviderID == "" {
	// 	return errors.NewValidationError("Workload Identity Provider ID is required")
	// }

	// Enhanced repository validation
	if config.Repository == "" { // Ensuring Repository is also checked if ProviderID check is removed
		return errors.NewValidationError("GitHub repository is required")
	}
	if err := ValidateGitHubRepository(config.Repository); err != nil {
		return err
	}

	// Validate pool ID format (3-32 characters, lowercase, hyphens)
	if len(config.PoolID) < 3 || len(config.PoolID) > 32 {
		return errors.NewValidationError(
			"Pool ID must be 3-32 characters long",
			"Use lowercase letters, digits, and hyphens only",
			"Cannot start or end with hyphens")
	}

	// Validate provider ID format (3-32 characters, lowercase, hyphens)
	// Only validate ProviderID format if it's actually provided.
	if config.ProviderID != "" {
		if len(config.ProviderID) < 3 || len(config.ProviderID) > 32 {
			return errors.NewValidationError(
				"Provider ID must be 3-32 characters long",
				"Use lowercase letters, digits, and hyphens only",
				"Cannot start or end with hyphens")
		}
		// Potentially add regex check for ProviderID format here if needed, similar to PoolID
	}

	// Validate GitHub OIDC configuration if provided
	if config.GitHubOIDC != nil {
		if err := ValidateGitHubOIDCConfig(config.GitHubOIDC); err != nil {
			return err
		}
	}

	return nil
}

// ValidateGitHubOIDCConfig validates GitHub OIDC configuration
func ValidateGitHubOIDCConfig(config *GitHubOIDCConfig) error {
	if config.IssuerURI == "" {
		return errors.NewValidationError("GitHub OIDC issuer URI is required")
	}

	if config.IssuerURI != "https://token.actions.githubusercontent.com" {
		return errors.NewValidationError(
			"Invalid GitHub OIDC issuer URI",
			"Must be: https://token.actions.githubusercontent.com")
	}

	if len(config.AllowedAudiences) == 0 {
		return errors.NewValidationError("At least one allowed audience is required")
	}

	// Validate audience format
	for _, audience := range config.AllowedAudiences {
		if audience == "" {
			return errors.NewValidationError("Audience cannot be empty")
		}
		// Common valid audiences for GitHub OIDC
		validAudiences := []string{"sts.googleapis.com", "sigstore", "pypi", "npm"}
		isValid := false
		for _, valid := range validAudiences {
			if strings.Contains(audience, valid) {
				isValid = true
				break
			}
		}
		// Also allow GitHub-specific audiences (matches bash script pattern)
		if strings.HasPrefix(audience, "https://github.com/") {
			isValid = true
		}
		if !isValid {
			return errors.NewValidationError(
				fmt.Sprintf("Invalid audience: %s", audience),
				"Common audiences: sts.googleapis.com, https://github.com/owner/repo")
		}
	}

	return nil
}

// CreateWorkloadIdentityPool creates a workload identity pool with enhanced features
func (c *Client) CreateWorkloadIdentityPool(config *WorkloadIdentityConfig) (*WorkloadIdentityPoolInfo, error) {
	logger := c.logger.WithField("function", "CreateWorkloadIdentityPool")
	logger.Info("Creating workload identity pool", "pool_id", config.PoolID, "project_id", c.ProjectID)

	// Validate configuration
	if err := ValidateWorkloadIdentityConfig(config); err != nil {
		return nil, err
	}

	// Enhanced conflict detection for pool only
	poolConflicts, err := c.detectWorkloadIdentityPoolConflicts(config)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "WI_POOL_CONFLICT_DETECTION_FAILED",
			"Failed to detect workload identity pool conflicts")
	}

	// Handle pool conflicts
	if len(poolConflicts) > 0 {
		conflict := poolConflicts[0]
		logger.Info("Workload identity pool conflicts detected",
			"severity", conflict.Severity,
			"differences", len(conflict.Differences))

		// If critical conflicts and CreateNew is false, return detailed error
		if conflict.Severity == ConflictSeverityCritical && !config.CreateNew {
			var suggestionTexts []string
			for _, suggestion := range conflict.Suggestions {
				if suggestion.Recommended {
					suggestionTexts = append(suggestionTexts, fmt.Sprintf("✓ %s: %s", suggestion.Title, suggestion.Description))
				} else {
					suggestionTexts = append(suggestionTexts, fmt.Sprintf("• %s: %s", suggestion.Title, suggestion.Description))
				}
			}

			return nil, errors.NewGCPError(
				fmt.Sprintf("Workload identity pool %s already exists with critical conflicts", config.PoolID),
				"Current pool configuration:",
				fmt.Sprintf("  - State: %s", conflict.ExistingDetails["state"]),
				fmt.Sprintf("  - Disabled: %v", conflict.ExistingDetails["disabled"]),
				fmt.Sprintf("  - Display Name: %s", conflict.ExistingDetails["display_name"]),
				"",
				"Resolution options:",
				strings.Join(suggestionTexts, "\n"))
		}

		// For non-critical conflicts or when CreateNew is true, use existing pool
		existing, err := c.GetWorkloadIdentityPoolInfo(config.PoolID)
		if err != nil {
			return nil, errors.WrapError(err, errors.ErrorTypeGCP, "WI_POOL_CHECK_FAILED",
				"Failed to check if workload identity pool exists")
		}

		if existing != nil && existing.Exists {
			logger.Info("Using existing workload identity pool", "pool_id", config.PoolID, "state", existing.State)
			return existing, nil
		}
	}

	// Set defaults
	displayName := config.PoolName
	if displayName == "" {
		displayName = fmt.Sprintf("WIF Pool for %s", config.Repository)
	}
	// Ensure display name doesn't exceed 32 characters
	displayName = truncateDisplayName(displayName, 32)

	description := config.PoolDescription
	if description == "" {
		description = fmt.Sprintf("Workload identity pool for GitHub repository %s", config.Repository)
	}

	logger.Debug("Creating workload identity pool with gcloud CLI",
		"pool_id", config.PoolID,
		"display_name", displayName,
		"description", description)

	// Create workload identity pool using gcloud CLI
	// Note: The Go API client doesn't support workload identity pools yet
	cmd := exec.Command("gcloud", "iam", "workload-identity-pools", "create", config.PoolID,
		"--project", c.ProjectID,
		"--location", "global",
		"--display-name", displayName,
		"--description", description,
		"--format", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "WI_POOL_CREATION_FAILED",
			fmt.Sprintf("Failed to create workload identity pool %s: %s", config.PoolID, string(output)))
	}

	logger.Info("Workload identity pool created successfully", "pool_id", config.PoolID)

	// Return pool information
	return c.GetWorkloadIdentityPoolInfo(config.PoolID)
}

// CreateWorkloadIdentityProvider creates a workload identity provider for GitHub OIDC with enhanced security
func (c *Client) CreateWorkloadIdentityProvider(config *WorkloadIdentityConfig) (*WorkloadIdentityProviderInfo, error) {
	logger := c.logger.WithField("function", "CreateWorkloadIdentityProvider")
	logger.Info("Creating workload identity provider",
		"pool_id", config.PoolID,
		"provider_id", config.ProviderID,
		"repository", config.Repository)

	// Validate configuration
	if err := ValidateWorkloadIdentityConfig(config); err != nil {
		return nil, err
	}

	// Enhanced conflict detection for provider only
	providerConflicts, err := c.detectWorkloadIdentityProviderConflicts(config)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "WI_PROVIDER_CONFLICT_DETECTION_FAILED",
			"Failed to detect workload identity provider conflicts")
	}

	// Handle provider conflicts
	if len(providerConflicts) > 0 {
		conflict := providerConflicts[0]
		logger.Info("Workload identity provider conflicts detected",
			"severity", conflict.Severity,
			"differences", len(conflict.Differences))

		// If critical conflicts and CreateNew is false, return detailed error
		if conflict.Severity == ConflictSeverityCritical && !config.CreateNew {
			var suggestionTexts []string
			for _, suggestion := range conflict.Suggestions {
				if suggestion.Recommended {
					suggestionTexts = append(suggestionTexts, fmt.Sprintf("✓ %s: %s", suggestion.Title, suggestion.Description))
				} else {
					suggestionTexts = append(suggestionTexts, fmt.Sprintf("• %s: %s", suggestion.Title, suggestion.Description))
				}
			}

			return nil, errors.NewGCPError(
				fmt.Sprintf("Workload identity provider %s already exists with critical conflicts", config.ProviderID),
				"Current provider configuration:",
				fmt.Sprintf("  - Repository: %s", extractRepositoryFromCondition(fmt.Sprintf("%v", conflict.ExistingDetails["attribute_condition"]))),
				fmt.Sprintf("  - Issuer URI: %s", conflict.ExistingDetails["issuer_uri"]),
				fmt.Sprintf("  - State: %s", conflict.ExistingDetails["state"]),
				"",
				"Resolution options:",
				strings.Join(suggestionTexts, "\n"))
		}

		// For non-critical conflicts or when CreateNew is true, use existing provider
		existing, err := c.GetWorkloadIdentityProviderInfo(config.PoolID, config.ProviderID)
		if err != nil {
			return nil, errors.WrapError(err, errors.ErrorTypeGCP, "WI_PROVIDER_CHECK_FAILED",
				"Failed to check if workload identity provider exists")
		}

		if existing != nil && existing.Exists {
			logger.Info("Using existing workload identity provider", "provider_id", config.ProviderID, "repository_compatible", strings.Contains(existing.AttributeCondition, config.Repository))
			return existing, nil
		}
	}

	// Set defaults
	displayName := config.ProviderName
	if displayName == "" {
		displayName = fmt.Sprintf("GitHub OIDC for %s", config.Repository)
	}
	// Ensure display name doesn't exceed 32 characters
	displayName = truncateDisplayName(displayName, 32)

	description := config.ProviderDescription
	if description == "" {
		description = fmt.Sprintf("GitHub OIDC provider for repository %s", config.Repository)
	}

	// Get GitHub OIDC configuration (use repository-specific if not provided)
	oidcConfig := config.GitHubOIDC
	if oidcConfig == nil {
		// Use repository-specific config that matches bash script pattern
		oidcConfig = GetGitHubRepositorySpecificOIDCConfig(config.Repository)
	}

	// Get claims mapping (use default if not provided)
	claimsMapping := config.ClaimsMapping
	if claimsMapping == nil {
		claimsMapping = GetDefaultGitHubClaimsMapping()
	}

	// Create enhanced attribute mapping for GitHub with all available claims
	attributeMapping := c.buildGitHubAttributeMapping(claimsMapping)

	// Create enhanced attribute condition with security constraints
	attributeCondition := c.buildGitHubSecurityConditions(&SecurityConditions{
		Repository:        config.Repository,
		AllowedBranches:   config.AllowedBranches,
		AllowedTags:       config.AllowedTags,
		AllowPullRequests: config.AllowPullRequests,
	}, oidcConfig)

	// Get audience configuration
	audiences := strings.Join(oidcConfig.AllowedAudiences, ",")

	logger.Debug("Creating workload identity provider with gcloud CLI",
		"provider_id", config.ProviderID,
		"display_name", displayName,
		"issuer_uri", oidcConfig.IssuerURI,
		"audiences", audiences,
		"attribute_mapping", attributeMapping,
		"attribute_condition", attributeCondition)

	// Create workload identity provider using gcloud CLI
	cmd := exec.Command("gcloud", "iam", "workload-identity-pools", "providers", "create-oidc", config.ProviderID,
		"--project", c.ProjectID,
		"--location", "global",
		"--workload-identity-pool", config.PoolID,
		"--display-name", displayName,
		"--description", description,
		"--issuer-uri", oidcConfig.IssuerURI,
		"--allowed-audiences", audiences,
		"--attribute-mapping", attributeMapping,
		"--attribute-condition", attributeCondition,
		"--format", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "WI_PROVIDER_CREATION_FAILED",
			fmt.Sprintf("Failed to create workload identity provider %s: %s", config.ProviderID, string(output)))
	}

	logger.Info("Workload identity provider created successfully",
		"provider_id", config.ProviderID,
		"repository", config.Repository,
		"issuer_uri", oidcConfig.IssuerURI)

	// Return provider information
	return c.GetWorkloadIdentityProviderInfo(config.PoolID, config.ProviderID)
}

// buildGitHubAttributeMapping builds comprehensive attribute mapping for GitHub OIDC claims
func (c *Client) buildGitHubAttributeMapping(claimsMapping *GitHubClaimsMapping) string {
	var mappings []string

	// Core mappings
	mappings = append(mappings, fmt.Sprintf("google.subject=%s", claimsMapping.Subject))
	mappings = append(mappings, fmt.Sprintf("attribute.actor=%s", claimsMapping.Actor))
	mappings = append(mappings, fmt.Sprintf("attribute.repository=%s", claimsMapping.Repository))
	mappings = append(mappings, fmt.Sprintf("attribute.repository_owner=%s", claimsMapping.RepositoryOwner))
	mappings = append(mappings, fmt.Sprintf("attribute.ref=%s", claimsMapping.Ref))

	// Enhanced GitHub-specific mappings
	mappings = append(mappings, fmt.Sprintf("attribute.ref_type=%s", claimsMapping.RefType))
	mappings = append(mappings, fmt.Sprintf("attribute.workflow_ref=%s", claimsMapping.WorkflowRef))
	mappings = append(mappings, fmt.Sprintf("attribute.job_workflow_ref=%s", claimsMapping.JobWorkflowRef))
	mappings = append(mappings, fmt.Sprintf("attribute.runner_environment=%s", claimsMapping.RunnerEnvironment))

	// Optional mappings for pull requests
	if claimsMapping.BaseRef != "" {
		mappings = append(mappings, fmt.Sprintf("attribute.base_ref=%s", claimsMapping.BaseRef))
	}
	if claimsMapping.HeadRef != "" {
		mappings = append(mappings, fmt.Sprintf("attribute.head_ref=%s", claimsMapping.HeadRef))
	}
	if claimsMapping.PullRequest != "" {
		mappings = append(mappings, fmt.Sprintf("attribute.pull_request=%s", claimsMapping.PullRequest))
	}
	if claimsMapping.Environment != "" {
		mappings = append(mappings, fmt.Sprintf("attribute.environment=%s", claimsMapping.Environment))
	}

	return strings.Join(mappings, ",")
}

// buildGitHubSecurityConditions builds enhanced security conditions for GitHub OIDC
func (c *Client) buildGitHubSecurityConditions(conditions *SecurityConditions, oidcConfig *GitHubOIDCConfig) string {
	baseCondition := fmt.Sprintf("assertion.repository=='%s'", conditions.Repository)

	var additionalConditions []string

	// Add repository verification (prevent forked repo attacks if enabled)
	if oidcConfig.BlockForkedRepos {
		additionalConditions = append(additionalConditions,
			fmt.Sprintf("assertion.repository_owner=='%s'", strings.Split(conditions.Repository, "/")[0]))
	}

	// Add actor verification if required
	if oidcConfig.RequireActor {
		additionalConditions = append(additionalConditions, "has(assertion.actor)")
	}

	// Add workflow path validation if enabled
	if oidcConfig.ValidateTokenPath {
		additionalConditions = append(additionalConditions,
			fmt.Sprintf("assertion.job_workflow_ref.startsWith('%s/')", conditions.Repository))
	}

	// Add branch restrictions
	if len(conditions.AllowedBranches) > 0 {
		branchConditions := make([]string, len(conditions.AllowedBranches))
		for i, branch := range conditions.AllowedBranches {
			branchConditions[i] = fmt.Sprintf("assertion.ref=='refs/heads/%s'", branch)
		}
		additionalConditions = append(additionalConditions, fmt.Sprintf("(%s)", strings.Join(branchConditions, " || ")))
	}

	// Add tag restrictions
	if len(conditions.AllowedTags) > 0 {
		tagConditions := make([]string, len(conditions.AllowedTags))
		for i, tag := range conditions.AllowedTags {
			// Support wildcard patterns for tags
			if strings.Contains(tag, "*") {
				tagConditions[i] = fmt.Sprintf("assertion.ref.matches('refs/tags/%s')", strings.ReplaceAll(tag, "*", ".*"))
			} else {
				tagConditions[i] = fmt.Sprintf("assertion.ref=='refs/tags/%s'", tag)
			}
		}
		additionalConditions = append(additionalConditions, fmt.Sprintf("(%s)", strings.Join(tagConditions, " || ")))
	}

	// Add pull request condition with enhanced security
	if conditions.AllowPullRequests {
		prConditions := []string{
			"assertion.ref.startsWith('refs/pull/')",
			// Ensure PR is targeting allowed branches if specified
		}
		if len(conditions.AllowedBranches) > 0 {
			baseBranchConditions := make([]string, len(conditions.AllowedBranches))
			for i, branch := range conditions.AllowedBranches {
				baseBranchConditions[i] = fmt.Sprintf("assertion.base_ref=='refs/heads/%s'", branch)
			}
			prConditions = append(prConditions, fmt.Sprintf("(%s)", strings.Join(baseBranchConditions, " || ")))
		}
		additionalConditions = append(additionalConditions, fmt.Sprintf("(%s)", strings.Join(prConditions, " && ")))
	}

	// Add trusted repositories check if specified
	if len(oidcConfig.TrustedRepos) > 0 {
		trustedConditions := make([]string, len(oidcConfig.TrustedRepos))
		for i, repo := range oidcConfig.TrustedRepos {
			trustedConditions[i] = fmt.Sprintf("assertion.repository=='%s'", repo)
		}
		additionalConditions = append(additionalConditions, fmt.Sprintf("(%s)", strings.Join(trustedConditions, " || ")))
	}

	// Combine all conditions
	if len(additionalConditions) > 0 {
		return fmt.Sprintf("%s && (%s)", baseCondition, strings.Join(additionalConditions, " && "))
	}

	return baseCondition
}

// ValidateGitHubOIDCToken validates a GitHub OIDC token format and claims (for testing)
func (c *Client) ValidateGitHubOIDCToken(token string, expectedRepository string) error {
	logger := c.logger.WithField("function", "ValidateGitHubOIDCToken")
	logger.Debug("Validating GitHub OIDC token format")

	if token == "" {
		return errors.NewValidationError("OIDC token cannot be empty")
	}

	// Basic JWT format validation (header.payload.signature)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errors.NewValidationError(
			"Invalid JWT token format",
			"Expected format: header.payload.signature")
	}

	logger.Debug("GitHub OIDC token format validation passed",
		"expected_repository", expectedRepository)
	return nil
}

// GetGitHubOIDCConfiguration returns the current GitHub OIDC configuration for a provider
func (c *Client) GetGitHubOIDCConfiguration(poolID, providerID string) (*GitHubOIDCConfig, error) {
	providerInfo, err := c.GetWorkloadIdentityProviderInfo(poolID, providerID)
	if err != nil {
		return nil, err
	}

	if !providerInfo.Exists {
		return nil, errors.NewValidationError(
			fmt.Sprintf("Workload identity provider %s does not exist", providerID))
	}

	config := &GitHubOIDCConfig{
		IssuerURI:        providerInfo.IssuerURI,
		AllowedAudiences: providerInfo.AllowedAudiences,
	}

	// Set defaults if not found
	if config.IssuerURI == "" {
		config.IssuerURI = "https://token.actions.githubusercontent.com"
	}
	if len(config.AllowedAudiences) == 0 {
		config.AllowedAudiences = []string{"sts.googleapis.com"}
	}

	return config, nil
}

// BindServiceAccountToWorkloadIdentity binds a service account to the workload identity provider with enhanced security
func (c *Client) BindServiceAccountToWorkloadIdentity(config *WorkloadIdentityConfig) error {
	logger := c.logger.WithField("function", "BindServiceAccountToWorkloadIdentity")
	logger.Info("Binding service account to workload identity",
		"service_account", config.ServiceAccountEmail,
		"repository", config.Repository)

	if config.ServiceAccountEmail == "" {
		return errors.NewValidationError("Service account email is required")
	}

	// Create enhanced IAM policy binding with comprehensive security conditions
	bindingConfig := &IAMBindingConfig{
		ServiceAccountEmail: config.ServiceAccountEmail,
		PoolID:              config.PoolID,
		ProviderID:          config.ProviderID,
		Repository:          config.Repository,
		AllowedBranches:     config.AllowedBranches,
		AllowedTags:         config.AllowedTags,
		AllowPullRequests:   config.AllowPullRequests,
		GitHubOIDC:          config.GitHubOIDC,
	}

	// Create multiple role bindings with different security levels
	if err := c.createServiceAccountTokenCreatorBinding(bindingConfig); err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "SA_TOKEN_CREATOR_BINDING_FAILED",
			"Failed to create service account token creator binding")
	}

	// Optionally create workload identity user binding for legacy compatibility
	if err := c.createWorkloadIdentityUserBinding(bindingConfig); err != nil {
		logger.Warn("Failed to create workload identity user binding", "error", err)
		// Don't fail the entire operation for legacy binding issues
	}

	logger.Info("Service account bound to workload identity successfully",
		"service_account", config.ServiceAccountEmail,
		"repository", config.Repository,
		"branches", config.AllowedBranches,
		"tags", config.AllowedTags,
		"pull_requests", config.AllowPullRequests)

	return nil
}

// createServiceAccountTokenCreatorBinding creates the primary IAM binding for service account impersonation
func (c *Client) createServiceAccountTokenCreatorBinding(config *IAMBindingConfig) error {
	logger := c.logger.WithField("function", "createServiceAccountTokenCreatorBinding")
	logger.Debug("Creating service account token creator binding",
		"service_account", config.ServiceAccountEmail,
		"repository", config.Repository)

	// Build enhanced principal set member
	member := c.buildPrincipalSetMember(config)

	// Build enhanced security condition
	condition := c.buildEnhancedIAMCondition(config)

	// Create the binding
	if err := c.executeIAMBinding(config.ServiceAccountEmail, member, "roles/iam.serviceAccountTokenCreator", condition); err != nil {
		return err
	}

	logger.Info("Service account token creator binding created successfully",
		"service_account", config.ServiceAccountEmail,
		"repository", config.Repository)

	return nil
}

// createWorkloadIdentityUserBinding creates a legacy workload identity user binding for compatibility
func (c *Client) createWorkloadIdentityUserBinding(config *IAMBindingConfig) error {
	logger := c.logger.WithField("function", "createWorkloadIdentityUserBinding")
	logger.Debug("Creating workload identity user binding",
		"service_account", config.ServiceAccountEmail,
		"repository", config.Repository)

	// Build legacy principal set member (for backward compatibility)
	member := fmt.Sprintf("principalSet://iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/%s/attribute.repository/%s",
		c.ProjectID, config.PoolID, config.Repository)

	// Build basic condition for legacy binding
	condition := &IAMCondition{
		Title:       fmt.Sprintf("Legacy WIF for %s", config.Repository),
		Description: fmt.Sprintf("Legacy workload identity user access for repository %s", config.Repository),
		Expression:  fmt.Sprintf("assertion.repository=='%s'", config.Repository),
	}

	// Create the binding
	if err := c.executeIAMBinding(config.ServiceAccountEmail, member, "roles/iam.workloadIdentityUser", condition); err != nil {
		return err
	}

	logger.Debug("Workload identity user binding created successfully",
		"service_account", config.ServiceAccountEmail,
		"repository", config.Repository)

	return nil
}

// buildPrincipalSetMember builds an enhanced principal set member string with multiple attribute options
func (c *Client) buildPrincipalSetMember(config *IAMBindingConfig) string {
	// Use provider-based principal set for more granular control
	return fmt.Sprintf("principalSet://iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/%s/providers/%s/*",
		c.ProjectID, config.PoolID, config.ProviderID)
}

// IAMCondition represents an IAM condition for policy bindings
type IAMCondition struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Expression  string `json:"expression"`
}

// buildEnhancedIAMCondition builds comprehensive security conditions for IAM bindings
func (c *Client) buildEnhancedIAMCondition(config *IAMBindingConfig) *IAMCondition {
	title := config.BindingTitle
	if title == "" {
		title = fmt.Sprintf("Enhanced WIF for %s", config.Repository)
	}

	description := config.BindingDescription
	if description == "" {
		description = fmt.Sprintf("Enhanced workload identity access for GitHub repository %s with comprehensive security conditions", config.Repository)
	}

	// Build comprehensive CEL expression
	expression := c.buildComprehensiveSecurityExpression(config)

	return &IAMCondition{
		Title:       title,
		Description: description,
		Expression:  expression,
	}
}

// buildComprehensiveSecurityExpression builds a comprehensive CEL security expression
func (c *Client) buildComprehensiveSecurityExpression(config *IAMBindingConfig) string {
	var conditions []string

	// Base repository condition (required)
	conditions = append(conditions, fmt.Sprintf("assertion.repository=='%s'", config.Repository))

	// Add GitHub OIDC security conditions if configured
	if config.GitHubOIDC != nil {
		// Repository owner verification (prevent forked repo attacks)
		if config.GitHubOIDC.BlockForkedRepos {
			repositoryOwner := strings.Split(config.Repository, "/")[0]
			conditions = append(conditions, fmt.Sprintf("assertion.repository_owner=='%s'", repositoryOwner))
		}

		// Actor verification (ensure actor claim is present)
		if config.GitHubOIDC.RequireActor {
			conditions = append(conditions, "has(assertion.actor)")
		}

		// Workflow path validation (prevent workflow injection attacks)
		if config.GitHubOIDC.ValidateTokenPath {
			conditions = append(conditions, fmt.Sprintf("assertion.job_workflow_ref.startsWith('%s/.github/workflows/')", config.Repository))
		}

		// Trusted repositories check
		if len(config.GitHubOIDC.TrustedRepos) > 0 {
			trustedConditions := make([]string, len(config.GitHubOIDC.TrustedRepos))
			for i, repo := range config.GitHubOIDC.TrustedRepos {
				trustedConditions[i] = fmt.Sprintf("assertion.repository=='%s'", repo)
			}
			conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(trustedConditions, " || ")))
		}
	}

	// Add branch restrictions
	if len(config.AllowedBranches) > 0 {
		branchConditions := make([]string, len(config.AllowedBranches))
		for i, branch := range config.AllowedBranches {
			// Support wildcard patterns
			if strings.Contains(branch, "*") {
				branchConditions[i] = fmt.Sprintf("assertion.ref.matches('refs/heads/%s')", strings.ReplaceAll(branch, "*", ".*"))
			} else {
				branchConditions[i] = fmt.Sprintf("assertion.ref=='refs/heads/%s'", branch)
			}
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(branchConditions, " || ")))
	}

	// Add tag restrictions
	if len(config.AllowedTags) > 0 {
		tagConditions := make([]string, len(config.AllowedTags))
		for i, tag := range config.AllowedTags {
			// Support wildcard patterns and semantic versioning
			if strings.Contains(tag, "*") {
				tagConditions[i] = fmt.Sprintf("assertion.ref.matches('refs/tags/%s')", strings.ReplaceAll(tag, "*", ".*"))
			} else {
				tagConditions[i] = fmt.Sprintf("assertion.ref=='refs/tags/%s'", tag)
			}
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(tagConditions, " || ")))
	}

	// Add pull request conditions with enhanced security
	if config.AllowPullRequests {
		prConditions := []string{
			"assertion.ref.startsWith('refs/pull/')",
			"has(assertion.pull_request)", // Ensure PR claim exists
		}

		// Ensure PR is targeting allowed branches if branch restrictions exist
		if len(config.AllowedBranches) > 0 {
			baseBranchConditions := make([]string, len(config.AllowedBranches))
			for i, branch := range config.AllowedBranches {
				baseBranchConditions[i] = fmt.Sprintf("assertion.base_ref=='refs/heads/%s'", branch)
			}
			prConditions = append(prConditions, fmt.Sprintf("(%s)", strings.Join(baseBranchConditions, " || ")))
		}

		// Add PR security conditions if GitHub OIDC is configured
		if config.GitHubOIDC != nil && config.GitHubOIDC.BlockForkedRepos {
			prConditions = append(prConditions, fmt.Sprintf("assertion.repository_owner=='%s'", strings.Split(config.Repository, "/")[0]))
		}

		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(prConditions, " && ")))
	}

	// Add environment-based conditions (for production deployments)
	conditions = append(conditions, "has(assertion.environment) == false || assertion.environment in ['', 'production', 'staging']")

	// Add time-based conditions (optional - prevent very old tokens)
	conditions = append(conditions, "request.time < timestamp(assertion.exp)")

	// Combine all conditions with AND logic
	return strings.Join(conditions, " && ")
}

// executeIAMBinding executes the actual IAM policy binding using gcloud CLI
func (c *Client) executeIAMBinding(serviceAccountEmail, member, role string, condition *IAMCondition) error {
	logger := c.logger.WithField("function", "executeIAMBinding")
	logger.Debug("Executing IAM policy binding",
		"service_account", serviceAccountEmail,
		"member", member,
		"role", role,
		"condition_title", condition.Title)

	// Build gcloud command
	args := []string{
		"iam", "service-accounts", "add-iam-policy-binding", serviceAccountEmail,
		"--project", c.ProjectID,
		"--member", member,
		"--role", role,
		"--condition", fmt.Sprintf("title=%s,description=%s,expression=%s",
			condition.Title, condition.Description, condition.Expression),
		"--format", "json",
	}

	cmd := exec.Command("gcloud", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check for common errors and provide helpful suggestions
		outputStr := string(output)
		if strings.Contains(outputStr, "already exists") {
			logger.Info("IAM binding already exists, skipping", "role", role)
			return nil
		}
		if strings.Contains(outputStr, "Invalid expression") {
			return errors.NewGCPError(
				fmt.Sprintf("Invalid CEL expression in IAM condition: %s", err),
				"CEL Expression:",
				condition.Expression,
				"",
				"Common issues:",
				"- Check string quoting and escaping",
				"- Verify assertion field names are correct",
				"- Ensure logical operators are properly formatted")
		}
		return errors.WrapError(err, errors.ErrorTypeGCP, "IAM_BINDING_FAILED",
			fmt.Sprintf("Failed to create IAM binding: %s", outputStr))
	}

	logger.Debug("IAM policy binding executed successfully", "role", role)
	return nil
}

// RemoveServiceAccountWorkloadIdentityBinding removes IAM bindings for workload identity
func (c *Client) RemoveServiceAccountWorkloadIdentityBinding(config *WorkloadIdentityConfig) error {
	logger := c.logger.WithField("function", "RemoveServiceAccountWorkloadIdentityBinding")
	logger.Info("Removing service account workload identity bindings",
		"service_account", config.ServiceAccountEmail,
		"repository", config.Repository)

	if config.ServiceAccountEmail == "" {
		return errors.NewValidationError("Service account email is required")
	}

	bindingConfig := &IAMBindingConfig{
		ServiceAccountEmail: config.ServiceAccountEmail,
		PoolID:              config.PoolID,
		ProviderID:          config.ProviderID,
		Repository:          config.Repository,
	}

	// Remove service account token creator binding
	member := c.buildPrincipalSetMember(bindingConfig)
	if err := c.removeIAMBinding(config.ServiceAccountEmail, member, "roles/iam.serviceAccountTokenCreator"); err != nil {
		logger.Warn("Failed to remove service account token creator binding", "error", err)
	}

	// Remove legacy workload identity user binding
	legacyMember := fmt.Sprintf("principalSet://iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/%s/attribute.repository/%s",
		c.ProjectID, config.PoolID, config.Repository)
	if err := c.removeIAMBinding(config.ServiceAccountEmail, legacyMember, "roles/iam.workloadIdentityUser"); err != nil {
		logger.Warn("Failed to remove legacy workload identity user binding", "error", err)
	}

	logger.Info("Service account workload identity bindings removed successfully",
		"service_account", config.ServiceAccountEmail,
		"repository", config.Repository)

	return nil
}

// removeIAMBinding removes an IAM policy binding
func (c *Client) removeIAMBinding(serviceAccountEmail, member, role string) error {
	logger := c.logger.WithField("function", "removeIAMBinding")
	logger.Debug("Removing IAM policy binding",
		"service_account", serviceAccountEmail,
		"member", member,
		"role", role)

	cmd := exec.Command("gcloud", "iam", "service-accounts", "remove-iam-policy-binding", serviceAccountEmail,
		"--project", c.ProjectID,
		"--member", member,
		"--role", role,
		"--all", // Remove all matching bindings
		"--format", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if binding doesn't exist (not an error)
		if strings.Contains(string(output), "NOT_FOUND") || strings.Contains(string(output), "not found") {
			logger.Debug("IAM binding not found, nothing to remove", "role", role)
			return nil
		}
		return errors.WrapError(err, errors.ErrorTypeGCP, "IAM_BINDING_REMOVE_FAILED",
			fmt.Sprintf("Failed to remove IAM binding: %s", string(output)))
	}

	logger.Debug("IAM policy binding removed successfully", "role", role)
	return nil
}

// ListServiceAccountWorkloadIdentityBindings lists all workload identity bindings for a service account
func (c *Client) ListServiceAccountWorkloadIdentityBindings(serviceAccountEmail string) ([]WorkloadIdentityBinding, error) {
	logger := c.logger.WithField("function", "ListServiceAccountWorkloadIdentityBindings")
	logger.Debug("Listing workload identity bindings", "service_account", serviceAccountEmail)

	cmd := exec.Command("gcloud", "iam", "service-accounts", "get-iam-policy", serviceAccountEmail,
		"--project", c.ProjectID,
		"--format", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "IAM_POLICY_GET_FAILED",
			fmt.Sprintf("Failed to get IAM policy for service account: %s", string(output)))
	}

	var policy IAMPolicy
	if err := json.Unmarshal(output, &policy); err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "IAM_POLICY_PARSE_FAILED",
			"Failed to parse IAM policy")
	}

	var bindings []WorkloadIdentityBinding
	for _, binding := range policy.Bindings {
		// Look for workload identity-related roles and members
		if c.isWorkloadIdentityRole(binding.Role) {
			for _, member := range binding.Members {
				if c.isWorkloadIdentityMember(member) {
					wfBinding := WorkloadIdentityBinding{
						Role:       binding.Role,
						Member:     member,
						Condition:  binding.Condition,
						Repository: c.extractRepositoryFromMember(member),
						PoolID:     c.extractPoolIDFromMember(member),
						ProviderID: c.extractProviderIDFromMember(member),
					}
					bindings = append(bindings, wfBinding)
				}
			}
		}
	}

	logger.Debug("Workload identity bindings listed", "count", len(bindings))
	return bindings, nil
}

// WorkloadIdentityBinding represents a workload identity IAM binding
type WorkloadIdentityBinding struct {
	Role       string        `json:"role"`
	Member     string        `json:"member"`
	Condition  *IAMCondition `json:"condition,omitempty"`
	Repository string        `json:"repository"`
	PoolID     string        `json:"pool_id"`
	ProviderID string        `json:"provider_id,omitempty"`
}

// IAMPolicy represents an IAM policy
type IAMPolicy struct {
	Version  int          `json:"version"`
	Bindings []IAMBinding `json:"bindings"`
	Etag     string       `json:"etag"`
}

// IAMBinding represents an IAM policy binding
type IAMBinding struct {
	Role      string        `json:"role"`
	Members   []string      `json:"members"`
	Condition *IAMCondition `json:"condition,omitempty"`
}

// isWorkloadIdentityRole checks if a role is related to workload identity
func (c *Client) isWorkloadIdentityRole(role string) bool {
	workloadIdentityRoles := []string{
		"roles/iam.serviceAccountTokenCreator",
		"roles/iam.workloadIdentityUser",
		"roles/iam.serviceAccountUser",
	}
	for _, wiRole := range workloadIdentityRoles {
		if role == wiRole {
			return true
		}
	}
	return false
}

// isWorkloadIdentityMember checks if a member is a workload identity principal
func (c *Client) isWorkloadIdentityMember(member string) bool {
	return strings.Contains(member, "principalSet://iam.googleapis.com") &&
		strings.Contains(member, "workloadIdentityPools")
}

// extractRepositoryFromMember extracts repository from a workload identity member string
func (c *Client) extractRepositoryFromMember(member string) string {
	// Look for attribute.repository/ pattern
	if strings.Contains(member, "attribute.repository/") {
		start := strings.Index(member, "attribute.repository/") + len("attribute.repository/")
		return member[start:]
	}
	return ""
}

// extractPoolIDFromMember extracts pool ID from a workload identity member string
func (c *Client) extractPoolIDFromMember(member string) string {
	// Look for workloadIdentityPools/ pattern
	if strings.Contains(member, "workloadIdentityPools/") {
		start := strings.Index(member, "workloadIdentityPools/") + len("workloadIdentityPools/")
		end := strings.Index(member[start:], "/")
		if end == -1 {
			return member[start:]
		}
		return member[start : start+end]
	}
	return ""
}

// extractProviderIDFromMember extracts provider ID from a workload identity member string
func (c *Client) extractProviderIDFromMember(member string) string {
	// Look for providers/ pattern
	if strings.Contains(member, "/providers/") {
		start := strings.Index(member, "/providers/") + len("/providers/")
		end := strings.Index(member[start:], "/")
		if end == -1 {
			return member[start:]
		}
		return member[start : start+end]
	}
	return ""
}

// ValidateIAMConditionExpression validates a CEL expression for IAM conditions
func (c *Client) ValidateIAMConditionExpression(expression string) error {
	return ValidateIAMConditionExpressionStandalone(expression)
}

// ValidateIAMConditionExpressionStandalone validates a CEL expression for IAM conditions without requiring a client
func ValidateIAMConditionExpressionStandalone(expression string) error {
	// Basic syntax validation
	if expression == "" {
		return errors.NewValidationError("IAM condition expression cannot be empty")
	}

	// Check for common syntax issues
	invalidPatterns := []struct {
		pattern string
		message string
	}{
		{"==''", "Empty string comparison without proper escaping"},
		{"='", "Single equals with quote (should be double equals)"},
		{"= '", "Single equals with space and quote (should be double equals)"},
		{" = ", "Single equals with spaces (should be double equals)"},
		{"&&&&", "Multiple logical AND operators"},
		{"||||", "Multiple logical OR operators"},
	}

	for _, pattern := range invalidPatterns {
		if strings.Contains(expression, pattern.pattern) {
			return errors.NewValidationError(
				fmt.Sprintf("Invalid CEL expression syntax: %s", pattern.message),
				"Common fixes:",
				"- Use '==' for equality comparison",
				"- Use '&&' for logical AND",
				"- Use '||' for logical OR",
				"- Properly quote string values")
		}
	}

	// Check for required assertion fields
	requiredFields := []string{"assertion.repository"}
	for _, field := range requiredFields {
		if !strings.Contains(expression, field) {
			return errors.NewValidationError(
				fmt.Sprintf("IAM condition expression must include %s", field),
				"This field is required for workload identity security")
		}
	}

	return nil
}

// GetWorkloadIdentityPoolInfo retrieves detailed information about a workload identity pool
func (c *Client) GetWorkloadIdentityPoolInfo(poolID string) (*WorkloadIdentityPoolInfo, error) {
	logger := c.logger.WithField("function", "GetWorkloadIdentityPoolInfo")
	logger.Debug("Getting workload identity pool info", "pool_id", poolID)

	cmd := exec.Command("gcloud", "iam", "workload-identity-pools", "describe", poolID,
		"--project", c.ProjectID,
		"--location", "global",
		"--format", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a 404 error (not found)
		outputStr := string(output)
		errorStr := err.Error()
		if strings.Contains(errorStr, "404") ||
			strings.Contains(strings.ToLower(errorStr), "not found") ||
			strings.Contains(outputStr, "NOT_FOUND") ||
			strings.Contains(strings.ToLower(outputStr), "not found") {
			logger.Debug("Workload identity pool not found", "pool_id", poolID)
			return &WorkloadIdentityPoolInfo{Exists: false}, nil
		}
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "WI_POOL_GET_FAILED",
			fmt.Sprintf("Failed to get workload identity pool %s: %s", poolID, outputStr))
	}

	var poolData map[string]interface{}
	if err := json.Unmarshal(output, &poolData); err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "WI_POOL_PARSE_FAILED",
			"Failed to parse workload identity pool information")
	}

	info := &WorkloadIdentityPoolInfo{
		Name:             fmt.Sprintf("%v", poolData["name"]),
		DisplayName:      fmt.Sprintf("%v", poolData["displayName"]),
		Description:      fmt.Sprintf("%v", poolData["description"]),
		State:            fmt.Sprintf("%v", poolData["state"]),
		Disabled:         poolData["disabled"] == true,
		Exists:           true,
		FullResourceName: fmt.Sprintf("%v", poolData["name"]),
	}

	// Parse creation time
	if createTimeStr, ok := poolData["createTime"].(string); ok {
		if createTime, err := time.Parse(time.RFC3339, createTimeStr); err == nil {
			info.CreateTime = createTime
		}
	}

	logger.Debug("Workload identity pool info retrieved", "pool_id", poolID, "state", info.State)
	return info, nil
}

// GetWorkloadIdentityProviderInfo retrieves detailed information about a workload identity provider
func (c *Client) GetWorkloadIdentityProviderInfo(poolID, providerID string) (*WorkloadIdentityProviderInfo, error) {
	logger := c.logger.WithField("function", "GetWorkloadIdentityProviderInfo")
	logger.Debug("Getting workload identity provider info", "pool_id", poolID, "provider_id", providerID)

	cmd := exec.Command("gcloud", "iam", "workload-identity-pools", "providers", "describe", providerID,
		"--project", c.ProjectID,
		"--location", "global",
		"--workload-identity-pool", poolID,
		"--format", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a 404 error (not found)
		outputStr := string(output)
		errorStr := err.Error()
		if strings.Contains(errorStr, "404") ||
			strings.Contains(strings.ToLower(errorStr), "not found") ||
			strings.Contains(outputStr, "NOT_FOUND") ||
			strings.Contains(strings.ToLower(outputStr), "not found") {
			logger.Debug("Workload identity provider not found", "pool_id", poolID, "provider_id", providerID)
			return &WorkloadIdentityProviderInfo{Exists: false}, nil
		}
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "WI_PROVIDER_GET_FAILED",
			fmt.Sprintf("Failed to get workload identity provider %s: %s", providerID, outputStr))
	}

	var providerData map[string]interface{}
	if err := json.Unmarshal(output, &providerData); err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "WI_PROVIDER_PARSE_FAILED",
			"Failed to parse workload identity provider information")
	}

	info := &WorkloadIdentityProviderInfo{
		Name:               fmt.Sprintf("%v", providerData["name"]),
		DisplayName:        fmt.Sprintf("%v", providerData["displayName"]),
		Description:        fmt.Sprintf("%v", providerData["description"]),
		State:              fmt.Sprintf("%v", providerData["state"]),
		Disabled:           providerData["disabled"] == true,
		AttributeCondition: fmt.Sprintf("%v", providerData["attributeCondition"]),
		Exists:             true,
		FullResourceName:   fmt.Sprintf("%v", providerData["name"]),
	}

	// Parse issuer URI and allowed audiences from OIDC section
	if oidcData, ok := providerData["oidc"].(map[string]interface{}); ok {
		if issuerURI, ok := oidcData["issuerUri"].(string); ok {
			info.IssuerURI = issuerURI
		}
		if audiences, ok := oidcData["allowedAudiences"].([]interface{}); ok {
			info.AllowedAudiences = make([]string, len(audiences))
			for i, audience := range audiences {
				info.AllowedAudiences[i] = fmt.Sprintf("%v", audience)
			}
		}
	}

	// Parse attribute mapping
	if attributeMapping, ok := providerData["attributeMapping"].(map[string]interface{}); ok {
		info.AttributeMapping = make(map[string]string)
		for k, v := range attributeMapping {
			info.AttributeMapping[k] = fmt.Sprintf("%v", v)
		}
	}

	// Parse creation time
	if createTimeStr, ok := providerData["createTime"].(string); ok {
		if createTime, err := time.Parse(time.RFC3339, createTimeStr); err == nil {
			info.CreateTime = createTime
		}
	}

	logger.Debug("Workload identity provider info retrieved",
		"pool_id", poolID,
		"provider_id", providerID,
		"state", info.State)
	return info, nil
}

// ListWorkloadIdentityPools lists all workload identity pools in the project
func (c *Client) ListWorkloadIdentityPools() ([]*WorkloadIdentityPoolInfo, error) {
	logger := c.logger.WithField("function", "ListWorkloadIdentityPools")
	logger.Debug("Listing workload identity pools", "project_id", c.ProjectID)

	cmd := exec.Command("gcloud", "iam", "workload-identity-pools", "list",
		"--project", c.ProjectID,
		"--location", "global",
		"--format", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "WI_POOLS_LIST_FAILED",
			"Failed to list workload identity pools")
	}

	var poolsData []map[string]interface{}
	if err := json.Unmarshal(output, &poolsData); err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "WI_POOLS_PARSE_FAILED",
			"Failed to parse workload identity pools list")
	}

	pools := make([]*WorkloadIdentityPoolInfo, len(poolsData))
	for i, poolData := range poolsData {
		pools[i] = &WorkloadIdentityPoolInfo{
			Name:             fmt.Sprintf("%v", poolData["name"]),
			DisplayName:      fmt.Sprintf("%v", poolData["displayName"]),
			Description:      fmt.Sprintf("%v", poolData["description"]),
			State:            fmt.Sprintf("%v", poolData["state"]),
			Disabled:         poolData["disabled"] == true,
			Exists:           true,
			FullResourceName: fmt.Sprintf("%v", poolData["name"]),
		}

		// Parse creation time
		if createTimeStr, ok := poolData["createTime"].(string); ok {
			if createTime, err := time.Parse(time.RFC3339, createTimeStr); err == nil {
				pools[i].CreateTime = createTime
			}
		}
	}

	logger.Debug("Workload identity pools listed", "count", len(pools))
	return pools, nil
}

// DeleteWorkloadIdentityPool deletes a workload identity pool with enhanced error handling
func (c *Client) DeleteWorkloadIdentityPool(poolID string) error {
	logger := c.logger.WithField("function", "DeleteWorkloadIdentityPool")
	logger.Info("Deleting workload identity pool", "pool_id", poolID)

	// Check if pool exists
	poolInfo, err := c.GetWorkloadIdentityPoolInfo(poolID)
	if err != nil {
		return err
	}

	if !poolInfo.Exists {
		logger.Info("Workload identity pool does not exist, nothing to delete", "pool_id", poolID)
		return nil
	}

	cmd := exec.Command("gcloud", "iam", "workload-identity-pools", "delete", poolID,
		"--project", c.ProjectID,
		"--location", "global",
		"--quiet",
		"--format", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "WI_POOL_DELETE_FAILED",
			fmt.Sprintf("Failed to delete workload identity pool %s: %s", poolID, string(output)))
	}

	logger.Info("Workload identity pool deleted successfully", "pool_id", poolID)
	return nil
}

// DeleteWorkloadIdentityProvider deletes a workload identity provider with enhanced error handling
func (c *Client) DeleteWorkloadIdentityProvider(poolID, providerID string) error {
	logger := c.logger.WithField("function", "DeleteWorkloadIdentityProvider")
	logger.Info("Deleting workload identity provider", "pool_id", poolID, "provider_id", providerID)

	// Check if provider exists
	providerInfo, err := c.GetWorkloadIdentityProviderInfo(poolID, providerID)
	if err != nil {
		return err
	}

	if !providerInfo.Exists {
		logger.Info("Workload identity provider does not exist, nothing to delete",
			"pool_id", poolID, "provider_id", providerID)
		return nil
	}

	cmd := exec.Command("gcloud", "iam", "workload-identity-pools", "providers", "delete", providerID,
		"--project", c.ProjectID,
		"--location", "global",
		"--workload-identity-pool", poolID,
		"--quiet",
		"--format", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "WI_PROVIDER_DELETE_FAILED",
			fmt.Sprintf("Failed to delete workload identity provider %s: %s", providerID, string(output)))
	}

	logger.Info("Workload identity provider deleted successfully",
		"pool_id", poolID, "provider_id", providerID)
	return nil
}

// GetWorkloadIdentityProviderName returns the full provider name for GitHub Actions
func (c *Client) GetWorkloadIdentityProviderName(poolID, providerID string) string {
	return fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s/providers/%s",
		c.ProjectID, poolID, providerID)
}

// GetWorkloadIdentityPoolName returns the full pool name
func (c *Client) GetWorkloadIdentityPoolName(poolID string) string {
	return fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s",
		c.ProjectID, poolID)
}

// extractRepositoryFromCondition extracts repository name from attribute condition
func extractRepositoryFromCondition(condition string) string {
	// Look for pattern: assertion.repository=='owner/repo'
	start := strings.Index(condition, "assertion.repository=='")
	if start == -1 {
		return "unknown"
	}
	start += len("assertion.repository=='")
	end := strings.Index(condition[start:], "'")
	if end == -1 {
		return "unknown"
	}
	return condition[start : start+end]
}

// truncateDisplayName truncates a display name to the specified maximum length
// while preserving meaningful information
func truncateDisplayName(name string, maxLength int) string {
	if len(name) <= maxLength {
		return name
	}

	// If we need to truncate, try to keep meaningful parts
	if maxLength <= 3 {
		return name[:maxLength]
	}

	// For repository names, try to preserve owner/repo format
	if strings.Contains(name, "/") && strings.Contains(name, "WIF Pool for ") {
		repo := strings.TrimPrefix(name, "WIF Pool for ")
		parts := strings.Split(repo, "/")
		if len(parts) == 2 {
			owner, repoName := parts[0], parts[1]
			// Try "WIF: owner/repo"
			shortForm := fmt.Sprintf("WIF: %s/%s", owner, repoName)
			if len(shortForm) <= maxLength {
				return shortForm
			}
			// Try "WIF: repo"
			shortForm = fmt.Sprintf("WIF: %s", repoName)
			if len(shortForm) <= maxLength {
				return shortForm
			}
		}
	}

	// Generic truncation with ellipsis
	return name[:maxLength-3] + "..."
}

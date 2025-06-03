// Package config handles configuration structure and JSON file operations
// for the GCP Workload Identity Federation CLI tool.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/logging"
)

// Config represents the complete configuration for setting up Workload Identity Federation
type Config struct {
	// Metadata about the configuration
	Version string `json:"version" validate:"required"`

	// Project configuration
	Project ProjectConfig `json:"project" validate:"required"`

	// Repository configuration
	Repository RepositoryConfig `json:"repository" validate:"required"`

	// Service Account configuration
	ServiceAccount ServiceAccountConfig `json:"service_account" validate:"required"`

	// Workload Identity configuration
	WorkloadIdentity WorkloadIdentityConfig `json:"workload_identity" validate:"required"`

	// Cloud Run configuration (optional)
	CloudRun CloudRunConfig `json:"cloud_run,omitempty"`

	// GitHub Actions workflow configuration
	Workflow WorkflowConfig `json:"workflow,omitempty"`

	// Advanced configuration
	Advanced AdvancedConfig `json:"advanced,omitempty"`
}

// ProjectConfig holds GCP project related configuration
type ProjectConfig struct {
	ID     string `json:"id" validate:"required"`
	Number string `json:"number,omitempty"`
	Region string `json:"region,omitempty"`
}

// RepositoryConfig holds GitHub repository configuration
type RepositoryConfig struct {
	Owner       string   `json:"owner" validate:"required"`
	Name        string   `json:"name" validate:"required"`
	Ref         string   `json:"ref,omitempty"` // Optional branch/tag filter
	Branches    []string `json:"branches,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	PullRequest bool     `json:"pull_request,omitempty"`
}

// ServiceAccountConfig holds service account configuration
type ServiceAccountConfig struct {
	Name        string   `json:"name" validate:"required"`
	DisplayName string   `json:"display_name,omitempty"`
	Description string   `json:"description,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	CreateNew   bool     `json:"create_new"`
}

// WorkloadIdentityConfig holds workload identity pool and provider configuration
type WorkloadIdentityConfig struct {
	PoolName         string            `json:"pool_name" validate:"required"`
	PoolID           string            `json:"pool_id" validate:"required"`
	ProviderName     string            `json:"provider_name" validate:"required"`
	ProviderID       string            `json:"provider_id" validate:"required"`
	AttributeMapping map[string]string `json:"attribute_mapping,omitempty"`
	Conditions       []string          `json:"conditions,omitempty"`
}

// CloudRunConfig holds Cloud Run service configuration
type CloudRunConfig struct {
	ServiceName  string            `json:"service_name,omitempty"`
	Region       string            `json:"region,omitempty"`
	Registry     string            `json:"registry,omitempty"`
	Image        string            `json:"image,omitempty"`
	Port         int               `json:"port,omitempty"`
	EnvVars      map[string]string `json:"env_vars,omitempty"`
	CPULimit     string            `json:"cpu_limit,omitempty"`
	MemoryLimit  string            `json:"memory_limit,omitempty"`
	MaxInstances int               `json:"max_instances,omitempty"`
	MinInstances int               `json:"min_instances,omitempty"`
}

// WorkflowConfig holds GitHub Actions workflow configuration
type WorkflowConfig struct {
	Name           string            `json:"name,omitempty"`
	Filename       string            `json:"filename,omitempty"`
	Path           string            `json:"path,omitempty"`
	Triggers       []string          `json:"triggers,omitempty"`
	Environment    string            `json:"environment,omitempty"`
	Secrets        map[string]string `json:"secrets,omitempty"`
	Variables      map[string]string `json:"variables,omitempty"`
	DockerRegistry string            `json:"docker_registry,omitempty"`
	DockerImage    string            `json:"docker_image,omitempty"`
}

// AdvancedConfig holds advanced configuration options
type AdvancedConfig struct {
	DryRun           bool     `json:"dry_run,omitempty"`
	SkipValidation   bool     `json:"skip_validation,omitempty"`
	ForceUpdate      bool     `json:"force_update,omitempty"`
	BackupExisting   bool     `json:"backup_existing,omitempty"`
	CleanupOnFailure bool     `json:"cleanup_on_failure,omitempty"`
	EnableAPIs       []string `json:"enable_apis,omitempty"`
	Timeout          string   `json:"timeout,omitempty"`
}

// ValidationResult represents the result of configuration validation
type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
	Info     []ValidationInfo    `json:"info,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationInfo represents validation information
type ValidationInfo struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Regular expressions for validation
var (
	gcpProjectIDRegex     = regexp.MustCompile(`^[a-z][-a-z0-9]{4,28}[a-z0-9]$`)
	serviceAccountRegex   = regexp.MustCompile(`^[a-z][-a-z0-9]{4,28}[a-z0-9]$`)
	githubOwnerRegex      = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
	githubRepoRegex       = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	workloadIdentityRegex = regexp.MustCompile(`^[a-z][-a-z0-9]{2,30}[a-z0-9]$`)
	cloudRunServiceRegex  = regexp.MustCompile(`^[a-z][-a-z0-9]{0,61}[a-z0-9]$`)
)

// DefaultRoles returns the default IAM roles for the service account
func DefaultRoles() []string {
	return []string{
		"roles/run.admin",
		"roles/storage.admin",
		"roles/artifactregistry.admin",
	}
}

// DefaultConfig creates a new configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Project: ProjectConfig{
			Region: "us-central1",
		},
		ServiceAccount: ServiceAccountConfig{
			Roles:     DefaultRoles(),
			CreateNew: true,
		},
		CloudRun: CloudRunConfig{
			Region:       "us-central1",
			Port:         8080,
			CPULimit:     "1",
			MemoryLimit:  "1Gi",
			MaxInstances: 100,
			MinInstances: 0,
		},
		Workflow: WorkflowConfig{
			Name:     "Deploy to Cloud Run",
			Filename: "deploy.yml",
			Path:     ".github/workflows",
			Triggers: []string{"push", "pull_request"},
		},
		Advanced: AdvancedConfig{
			BackupExisting:   true,
			CleanupOnFailure: true,
			Timeout:          "30m",
		},
	}
}

// NewConfig creates a new configuration with the provided parameters
func NewConfig(projectID, repoOwner, repoName string) *Config {
	config := DefaultConfig()
	config.Project.ID = projectID
	config.Repository.Owner = repoOwner
	config.Repository.Name = repoName

	// Generate default names based on repository
	repoSlug := strings.ToLower(strings.ReplaceAll(repoName, "_", "-"))
	ownerSlug := strings.ToLower(repoOwner)

	// Ensure names fit within GCP limits by truncating if needed
	maxSegmentLength := 10 // Leave room for prefixes and suffixes
	if len(ownerSlug) > maxSegmentLength {
		ownerSlug = ownerSlug[:maxSegmentLength]
	}
	if len(repoSlug) > maxSegmentLength {
		repoSlug = repoSlug[:maxSegmentLength]
	}

	config.ServiceAccount.Name = fmt.Sprintf("github-%s-%s", ownerSlug, repoSlug)
	config.WorkloadIdentity.PoolName = fmt.Sprintf("GitHub Pool for %s/%s", repoOwner, repoName)
	config.WorkloadIdentity.PoolID = fmt.Sprintf("gh-%s-%s-pool", ownerSlug, repoSlug)
	config.WorkloadIdentity.ProviderName = fmt.Sprintf("GitHub Provider for %s/%s", repoOwner, repoName)
	config.WorkloadIdentity.ProviderID = fmt.Sprintf("gh-%s-%s-provider", ownerSlug, repoSlug)

	if config.CloudRun.ServiceName == "" {
		config.CloudRun.ServiceName = repoSlug
	}

	config.SetDefaults()
	return config
}

// LoadFromFile loads configuration from a JSON file with validation
func LoadFromFile(filePath string) (*Config, error) {
	logger := logging.WithField("config_file", filePath)
	logger.Debug("Loading configuration from file")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewConfigurationError(
				fmt.Sprintf("Configuration file not found: %s", filePath),
				"Create a configuration file using interactive mode",
				"Use --config flag to specify a different config file path",
				"Run with --interactive flag to create a new configuration")
		}
		return nil, errors.WrapError(err, errors.ErrorTypeFileSystem, "CONFIG_READ_FAILED",
			fmt.Sprintf("Failed to read configuration file: %s", filePath))
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.NewConfigurationError(
			fmt.Sprintf("Invalid JSON in configuration file: %s", filePath),
			"Check the JSON syntax in your configuration file",
			"Ensure all brackets and quotes are properly closed",
			"Use a JSON validator to check your configuration",
			"Run with --interactive flag to recreate the configuration")
	}

	logger.Info("Configuration loaded successfully", "version", config.Version)

	// Validate the loaded configuration
	result := config.ValidateSchema()
	if !result.Valid {
		var errorMessages []string
		for _, valErr := range result.Errors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", valErr.Field, valErr.Message))
		}
		return nil, errors.NewConfigurationError(
			"Configuration validation failed",
			errorMessages...)
	}

	// Log warnings if any
	for _, warning := range result.Warnings {
		logger.Warn("Configuration warning", "field", warning.Field, "message", warning.Message)
	}

	config.SetDefaults()
	return &config, nil
}

// SaveToFile saves configuration to a JSON file
func (c *Config) SaveToFile(filePath string) error {
	logger := logging.WithField("config_file", filePath)
	logger.Debug("Saving configuration to file")

	// Validate before saving
	result := c.ValidateSchema()
	if !result.Valid {
		var errorMessages []string
		for _, valErr := range result.Errors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", valErr.Field, valErr.Message))
		}
		return errors.NewConfigurationError(
			"Cannot save invalid configuration",
			errorMessages...)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return errors.WrapError(err, errors.ErrorTypeFileSystem, "CONFIG_DIR_CREATE_FAILED",
			"Failed to create directory for configuration file")
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeInternal, "CONFIG_MARSHAL_FAILED",
			"Failed to serialize configuration to JSON")
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return errors.WrapError(err, errors.ErrorTypeFileSystem, "CONFIG_WRITE_FAILED",
			fmt.Sprintf("Failed to write configuration file: %s", filePath))
	}

	logger.Info("Configuration saved successfully")
	return nil
}

// SetDefaults applies default values to the configuration
func (c *Config) SetDefaults() {
	// Set version if not specified
	if c.Version == "" {
		c.Version = "1.0"
	}

	// Set default project region
	if c.Project.Region == "" {
		c.Project.Region = "us-central1"
	}

	// Set default service account roles
	if len(c.ServiceAccount.Roles) == 0 {
		c.ServiceAccount.Roles = DefaultRoles()
	}

	// Set default service account display name and description
	if c.ServiceAccount.DisplayName == "" {
		c.ServiceAccount.DisplayName = fmt.Sprintf("GitHub Actions SA for %s", c.GetRepoFullName())
	}
	if c.ServiceAccount.Description == "" {
		c.ServiceAccount.Description = "Service account for GitHub Actions Workload Identity Federation"
	}

	// Set default workload identity attribute mapping
	if c.WorkloadIdentity.AttributeMapping == nil {
		c.WorkloadIdentity.AttributeMapping = map[string]string{
			"google.subject":             "assertion.sub",
			"attribute.actor":            "assertion.actor",
			"attribute.repository":       "assertion.repository",
			"attribute.repository_owner": "assertion.repository_owner",
		}
	}

	// Set default repository conditions
	if len(c.WorkloadIdentity.Conditions) == 0 {
		c.WorkloadIdentity.Conditions = []string{
			fmt.Sprintf("assertion.repository=='%s'", c.GetRepoFullName()),
		}
	}

	// Set Cloud Run defaults if service name is provided
	if c.CloudRun.ServiceName != "" {
		if c.CloudRun.Region == "" {
			c.CloudRun.Region = c.Project.Region
		}
		if c.CloudRun.Registry == "" {
			c.CloudRun.Registry = fmt.Sprintf("%s-docker.pkg.dev/%s/cloud-run-source-deploy",
				c.CloudRun.Region, c.Project.ID)
		}
		if c.CloudRun.Port == 0 {
			c.CloudRun.Port = 8080
		}
		if c.CloudRun.CPULimit == "" {
			c.CloudRun.CPULimit = "1"
		}
		if c.CloudRun.MemoryLimit == "" {
			c.CloudRun.MemoryLimit = "1Gi"
		}
	}

	// Set workflow defaults
	if c.Workflow.Name == "" {
		c.Workflow.Name = "Deploy to Cloud Run"
	}
	if c.Workflow.Filename == "" {
		c.Workflow.Filename = "deploy.yml"
	}
	if c.Workflow.Path == "" {
		c.Workflow.Path = ".github/workflows"
	}
	if len(c.Workflow.Triggers) == 0 {
		c.Workflow.Triggers = []string{"push"}
	}

	// Set advanced defaults
	if c.Advanced.Timeout == "" {
		c.Advanced.Timeout = "30m"
	}
}

// isValidGitHubOwner validates GitHub username/organization name
func isValidGitHubOwner(owner string) bool {
	// Basic regex check first
	if !githubOwnerRegex.MatchString(owner) {
		return false
	}

	// Cannot start or end with hyphen
	if strings.HasPrefix(owner, "-") || strings.HasSuffix(owner, "-") {
		return false
	}

	// Cannot have consecutive hyphens
	if strings.Contains(owner, "--") {
		return false
	}

	return true
}

// ValidateSchema performs comprehensive JSON schema validation
func (c *Config) ValidateSchema() *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
		Info:     []ValidationInfo{},
	}

	// Validate project configuration
	c.validateProject(result)

	// Validate repository configuration
	c.validateRepository(result)

	// Validate service account configuration
	c.validateServiceAccount(result)

	// Validate workload identity configuration
	c.validateWorkloadIdentity(result)

	// Validate Cloud Run configuration
	c.validateCloudRun(result)

	// Validate workflow configuration
	c.validateWorkflow(result)

	result.Valid = len(result.Errors) == 0
	return result
}

// validateProject validates project configuration
func (c *Config) validateProject(result *ValidationResult) {
	if c.Project.ID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field: "project.id", Value: "", Message: "Project ID is required", Code: "REQUIRED",
		})
		return
	}

	if !gcpProjectIDRegex.MatchString(c.Project.ID) {
		result.Errors = append(result.Errors, ValidationError{
			Field: "project.id", Value: c.Project.ID,
			Message: "Project ID must be 6-30 characters, start with lowercase letter, and contain only lowercase letters, digits, and hyphens",
			Code:    "INVALID_FORMAT",
		})
	}

	if len(c.Project.ID) < 6 || len(c.Project.ID) > 30 {
		result.Errors = append(result.Errors, ValidationError{
			Field: "project.id", Value: c.Project.ID,
			Message: "Project ID must be 6-30 characters long",
			Code:    "INVALID_LENGTH",
		})
	}
}

// validateRepository validates repository configuration
func (c *Config) validateRepository(result *ValidationResult) {
	if c.Repository.Owner == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field: "repository.owner", Value: "", Message: "Repository owner is required", Code: "REQUIRED",
		})
	} else {
		if !isValidGitHubOwner(c.Repository.Owner) {
			result.Errors = append(result.Errors, ValidationError{
				Field: "repository.owner", Value: c.Repository.Owner,
				Message: "Repository owner must contain only alphanumeric characters and hyphens, cannot start or end with hyphen, and cannot have consecutive hyphens",
				Code:    "INVALID_FORMAT",
			})
		}
		if len(c.Repository.Owner) > 39 {
			result.Errors = append(result.Errors, ValidationError{
				Field: "repository.owner", Value: c.Repository.Owner,
				Message: "Repository owner cannot be longer than 39 characters",
				Code:    "INVALID_LENGTH",
			})
		}
	}

	if c.Repository.Name == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field: "repository.name", Value: "", Message: "Repository name is required", Code: "REQUIRED",
		})
	} else {
		if !githubRepoRegex.MatchString(c.Repository.Name) {
			result.Errors = append(result.Errors, ValidationError{
				Field: "repository.name", Value: c.Repository.Name,
				Message: "Repository name must contain only alphanumeric characters, dots, hyphens, and underscores",
				Code:    "INVALID_FORMAT",
			})
		}
		if len(c.Repository.Name) > 100 {
			result.Errors = append(result.Errors, ValidationError{
				Field: "repository.name", Value: c.Repository.Name,
				Message: "Repository name cannot be longer than 100 characters",
				Code:    "INVALID_LENGTH",
			})
		}
	}
}

// validateServiceAccount validates service account configuration
func (c *Config) validateServiceAccount(result *ValidationResult) {
	if c.ServiceAccount.Name == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field: "service_account.name", Value: "", Message: "Service account name is required", Code: "REQUIRED",
		})
	} else {
		if !serviceAccountRegex.MatchString(c.ServiceAccount.Name) {
			result.Errors = append(result.Errors, ValidationError{
				Field: "service_account.name", Value: c.ServiceAccount.Name,
				Message: "Service account name must be 6-30 characters, start with lowercase letter, and contain only lowercase letters, digits, and hyphens",
				Code:    "INVALID_FORMAT",
			})
		}
	}

	// Validate roles
	for _, role := range c.ServiceAccount.Roles {
		if !strings.HasPrefix(role, "roles/") {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "service_account.roles",
				Message: fmt.Sprintf("Role '%s' should start with 'roles/'", role),
			})
		}
	}
}

// validateWorkloadIdentity validates workload identity configuration
func (c *Config) validateWorkloadIdentity(result *ValidationResult) {
	if c.WorkloadIdentity.PoolID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field: "workload_identity.pool_id", Value: "", Message: "Workload Identity Pool ID is required", Code: "REQUIRED",
		})
	} else if !workloadIdentityRegex.MatchString(c.WorkloadIdentity.PoolID) {
		result.Errors = append(result.Errors, ValidationError{
			Field: "workload_identity.pool_id", Value: c.WorkloadIdentity.PoolID,
			Message: "Pool ID must be 3-32 characters, start with lowercase letter, and contain only lowercase letters, digits, and hyphens",
			Code:    "INVALID_FORMAT",
		})
	}

	if c.WorkloadIdentity.ProviderID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field: "workload_identity.provider_id", Value: "", Message: "Workload Identity Provider ID is required", Code: "REQUIRED",
		})
	} else if !workloadIdentityRegex.MatchString(c.WorkloadIdentity.ProviderID) {
		result.Errors = append(result.Errors, ValidationError{
			Field: "workload_identity.provider_id", Value: c.WorkloadIdentity.ProviderID,
			Message: "Provider ID must be 3-32 characters, start with lowercase letter, and contain only lowercase letters, digits, and hyphens",
			Code:    "INVALID_FORMAT",
		})
	}
}

// validateCloudRun validates Cloud Run configuration
func (c *Config) validateCloudRun(result *ValidationResult) {
	if c.CloudRun.ServiceName != "" {
		if !cloudRunServiceRegex.MatchString(c.CloudRun.ServiceName) {
			result.Errors = append(result.Errors, ValidationError{
				Field: "cloud_run.service_name", Value: c.CloudRun.ServiceName,
				Message: "Cloud Run service name must start and end with lowercase letter and contain only lowercase letters, digits, and hyphens",
				Code:    "INVALID_FORMAT",
			})
		}
		if len(c.CloudRun.ServiceName) > 63 {
			result.Errors = append(result.Errors, ValidationError{
				Field: "cloud_run.service_name", Value: c.CloudRun.ServiceName,
				Message: "Cloud Run service name cannot be longer than 63 characters",
				Code:    "INVALID_LENGTH",
			})
		}
	}

	if c.CloudRun.Port != 0 && (c.CloudRun.Port < 1 || c.CloudRun.Port > 65535) {
		result.Errors = append(result.Errors, ValidationError{
			Field: "cloud_run.port", Value: fmt.Sprintf("%d", c.CloudRun.Port),
			Message: "Port must be between 1 and 65535",
			Code:    "INVALID_RANGE",
		})
	}
}

// validateWorkflow validates workflow configuration
func (c *Config) validateWorkflow(result *ValidationResult) {
	if c.Workflow.Filename != "" && !strings.HasSuffix(c.Workflow.Filename, ".yml") && !strings.HasSuffix(c.Workflow.Filename, ".yaml") {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "workflow.filename",
			Message: "Workflow filename should end with .yml or .yaml",
		})
	}
}

// Validate checks if the configuration is valid (legacy method for compatibility)
func (c *Config) Validate() error {
	result := c.ValidateSchema()
	if !result.Valid {
		var errorMessages []string
		for _, valErr := range result.Errors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", valErr.Field, valErr.Message))
		}
		return errors.NewValidationError(
			"Configuration validation failed",
			errorMessages...)
	}
	return nil
}

// GetRepoFullName returns the full repository name in owner/name format
func (c *Config) GetRepoFullName() string {
	return c.Repository.Owner + "/" + c.Repository.Name
}

// GetServiceAccountEmail returns the service account email
func (c *Config) GetServiceAccountEmail() string {
	return fmt.Sprintf("%s@%s.iam.gserviceaccount.com", c.ServiceAccount.Name, c.Project.ID)
}

// GetWorkloadIdentityPoolName returns the full workload identity pool name
func (c *Config) GetWorkloadIdentityPoolName() string {
	return fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s", c.Project.ID, c.WorkloadIdentity.PoolID)
}

// GetWorkloadIdentityProviderName returns the full workload identity provider name
func (c *Config) GetWorkloadIdentityProviderName() string {
	return fmt.Sprintf("%s/providers/%s", c.GetWorkloadIdentityPoolName(), c.WorkloadIdentity.ProviderID)
}

// GetCloudRunURL returns the Cloud Run service URL
func (c *Config) GetCloudRunURL() string {
	if c.CloudRun.ServiceName == "" || c.CloudRun.Region == "" {
		return ""
	}
	return fmt.Sprintf("https://%s-%s.a.run.app", c.CloudRun.ServiceName, c.CloudRun.Region)
}

// GetWorkflowFilePath returns the full path to the workflow file
func (c *Config) GetWorkflowFilePath() string {
	return filepath.Join(c.Workflow.Path, c.Workflow.Filename)
}

// ToJSON converts the configuration to JSON string
func (c *Config) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON creates a configuration from JSON string
func FromJSON(jsonStr string) (*Config, error) {
	var config Config
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		return nil, err
	}
	config.SetDefaults()
	return &config, nil
}

// AutoDiscoverConfigFile attempts to find a configuration file in common locations
func AutoDiscoverConfigFile() (string, error) {
	logger := logging.WithField("function", "AutoDiscoverConfigFile")

	// List of possible configuration file names in order of preference
	candidates := []string{
		"wif-config.json",
		"gcp-wif.json",
		".gcp-wif.json",
		"config/wif.json",
		"config/gcp-wif.json",
		".wif/config.json",
		"examples/config.json",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			logger.Info("Configuration file auto-discovered", "file", candidate)
			return candidate, nil
		}
	}

	logger.Debug("No configuration file found in common locations")
	return "", errors.NewConfigurationError(
		"No configuration file found",
		"Create a configuration file using 'gcp-wif config init'",
		"Specify a configuration file with --config flag",
		"Use absolute path if the file is in a different location")
}

// MergeConfig merges another configuration into this one, with the other config taking precedence
func (c *Config) MergeConfig(other *Config) error {
	logger := logging.WithField("function", "MergeConfig")
	logger.Debug("Merging configuration")

	if other == nil {
		return errors.NewConfigurationError("Cannot merge nil configuration")
	}

	// Merge project configuration
	if other.Project.ID != "" {
		c.Project.ID = other.Project.ID
	}
	if other.Project.Number != "" {
		c.Project.Number = other.Project.Number
	}
	if other.Project.Region != "" {
		c.Project.Region = other.Project.Region
	}

	// Merge repository configuration
	if other.Repository.Owner != "" {
		c.Repository.Owner = other.Repository.Owner
	}
	if other.Repository.Name != "" {
		c.Repository.Name = other.Repository.Name
	}
	if other.Repository.Ref != "" {
		c.Repository.Ref = other.Repository.Ref
	}
	if len(other.Repository.Branches) > 0 {
		c.Repository.Branches = other.Repository.Branches
	}
	if len(other.Repository.Tags) > 0 {
		c.Repository.Tags = other.Repository.Tags
	}
	if other.Repository.PullRequest {
		c.Repository.PullRequest = other.Repository.PullRequest
	}

	// Merge service account configuration
	if other.ServiceAccount.Name != "" {
		c.ServiceAccount.Name = other.ServiceAccount.Name
	}
	if other.ServiceAccount.DisplayName != "" {
		c.ServiceAccount.DisplayName = other.ServiceAccount.DisplayName
	}
	if other.ServiceAccount.Description != "" {
		c.ServiceAccount.Description = other.ServiceAccount.Description
	}
	if len(other.ServiceAccount.Roles) > 0 {
		c.ServiceAccount.Roles = other.ServiceAccount.Roles
	}

	// Merge workload identity configuration
	if other.WorkloadIdentity.PoolName != "" {
		c.WorkloadIdentity.PoolName = other.WorkloadIdentity.PoolName
	}
	if other.WorkloadIdentity.PoolID != "" {
		c.WorkloadIdentity.PoolID = other.WorkloadIdentity.PoolID
	}
	if other.WorkloadIdentity.ProviderName != "" {
		c.WorkloadIdentity.ProviderName = other.WorkloadIdentity.ProviderName
	}
	if other.WorkloadIdentity.ProviderID != "" {
		c.WorkloadIdentity.ProviderID = other.WorkloadIdentity.ProviderID
	}
	if len(other.WorkloadIdentity.AttributeMapping) > 0 {
		if c.WorkloadIdentity.AttributeMapping == nil {
			c.WorkloadIdentity.AttributeMapping = make(map[string]string)
		}
		for k, v := range other.WorkloadIdentity.AttributeMapping {
			c.WorkloadIdentity.AttributeMapping[k] = v
		}
	}
	if len(other.WorkloadIdentity.Conditions) > 0 {
		c.WorkloadIdentity.Conditions = other.WorkloadIdentity.Conditions
	}

	// Merge Cloud Run configuration
	if other.CloudRun.ServiceName != "" {
		c.CloudRun.ServiceName = other.CloudRun.ServiceName
	}
	if other.CloudRun.Region != "" {
		c.CloudRun.Region = other.CloudRun.Region
	}
	if other.CloudRun.Registry != "" {
		c.CloudRun.Registry = other.CloudRun.Registry
	}
	if other.CloudRun.Image != "" {
		c.CloudRun.Image = other.CloudRun.Image
	}
	if other.CloudRun.Port != 0 {
		c.CloudRun.Port = other.CloudRun.Port
	}
	if len(other.CloudRun.EnvVars) > 0 {
		if c.CloudRun.EnvVars == nil {
			c.CloudRun.EnvVars = make(map[string]string)
		}
		for k, v := range other.CloudRun.EnvVars {
			c.CloudRun.EnvVars[k] = v
		}
	}
	if other.CloudRun.CPULimit != "" {
		c.CloudRun.CPULimit = other.CloudRun.CPULimit
	}
	if other.CloudRun.MemoryLimit != "" {
		c.CloudRun.MemoryLimit = other.CloudRun.MemoryLimit
	}
	if other.CloudRun.MaxInstances != 0 {
		c.CloudRun.MaxInstances = other.CloudRun.MaxInstances
	}
	if other.CloudRun.MinInstances != 0 {
		c.CloudRun.MinInstances = other.CloudRun.MinInstances
	}

	// Merge workflow configuration
	if other.Workflow.Name != "" {
		c.Workflow.Name = other.Workflow.Name
	}
	if other.Workflow.Filename != "" {
		c.Workflow.Filename = other.Workflow.Filename
	}
	if other.Workflow.Path != "" {
		c.Workflow.Path = other.Workflow.Path
	}
	if len(other.Workflow.Triggers) > 0 {
		c.Workflow.Triggers = other.Workflow.Triggers
	}
	if other.Workflow.Environment != "" {
		c.Workflow.Environment = other.Workflow.Environment
	}
	if len(other.Workflow.Secrets) > 0 {
		if c.Workflow.Secrets == nil {
			c.Workflow.Secrets = make(map[string]string)
		}
		for k, v := range other.Workflow.Secrets {
			c.Workflow.Secrets[k] = v
		}
	}
	if len(other.Workflow.Variables) > 0 {
		if c.Workflow.Variables == nil {
			c.Workflow.Variables = make(map[string]string)
		}
		for k, v := range other.Workflow.Variables {
			c.Workflow.Variables[k] = v
		}
	}
	if other.Workflow.DockerRegistry != "" {
		c.Workflow.DockerRegistry = other.Workflow.DockerRegistry
	}
	if other.Workflow.DockerImage != "" {
		c.Workflow.DockerImage = other.Workflow.DockerImage
	}

	// Merge advanced configuration
	if other.Advanced.DryRun {
		c.Advanced.DryRun = other.Advanced.DryRun
	}
	if other.Advanced.SkipValidation {
		c.Advanced.SkipValidation = other.Advanced.SkipValidation
	}
	if other.Advanced.ForceUpdate {
		c.Advanced.ForceUpdate = other.Advanced.ForceUpdate
	}
	if other.Advanced.BackupExisting {
		c.Advanced.BackupExisting = other.Advanced.BackupExisting
	}
	if other.Advanced.CleanupOnFailure {
		c.Advanced.CleanupOnFailure = other.Advanced.CleanupOnFailure
	}
	if len(other.Advanced.EnableAPIs) > 0 {
		c.Advanced.EnableAPIs = other.Advanced.EnableAPIs
	}
	if other.Advanced.Timeout != "" {
		c.Advanced.Timeout = other.Advanced.Timeout
	}

	// Update version if the other config has a newer version
	if other.Version != "" && other.Version != c.Version {
		c.Version = other.Version
	}

	logger.Info("Configuration merged successfully")
	return nil
}

// CloneConfig creates a deep copy of the configuration
func (c *Config) CloneConfig() (*Config, error) {
	logger := logging.WithField("function", "CloneConfig")
	logger.Debug("Cloning configuration")

	// Use JSON serialization for deep copy
	jsonStr, err := c.ToJSON()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeInternal, "CONFIG_CLONE_SERIALIZE_FAILED",
			"Failed to serialize configuration for cloning")
	}

	clone, err := FromJSON(jsonStr)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeInternal, "CONFIG_CLONE_DESERIALIZE_FAILED",
			"Failed to deserialize configuration for cloning")
	}

	logger.Debug("Configuration cloned successfully")
	return clone, nil
}

// MigrateToLatestVersion migrates configuration to the latest version format
func (c *Config) MigrateToLatestVersion() (*Config, error) {
	logger := logging.WithField("function", "MigrateToLatestVersion")
	logger.Info("Migrating configuration to latest version", "current_version", c.Version)

	// Clone the configuration first
	migrated, err := c.CloneConfig()
	if err != nil {
		return nil, err
	}

	// Version 1.0 migration (if needed)
	if c.Version == "" || c.Version == "0.9" || c.Version < "1.0" {
		logger.Info("Migrating to version 1.0")

		// Set version
		migrated.Version = "1.0"

		// Ensure required fields have default values
		migrated.SetDefaults()

		// Add new fields that might not exist in older versions
		if migrated.WorkloadIdentity.AttributeMapping == nil {
			migrated.WorkloadIdentity.AttributeMapping = map[string]string{
				"google.subject":             "assertion.sub",
				"attribute.actor":            "assertion.actor",
				"attribute.repository":       "assertion.repository",
				"attribute.repository_owner": "assertion.repository_owner",
			}
		}

		// Update security conditions if they're using old format
		if len(migrated.WorkloadIdentity.Conditions) == 0 {
			migrated.WorkloadIdentity.Conditions = []string{
				fmt.Sprintf("assertion.repository=='%s'", migrated.GetRepoFullName()),
			}
		}

		logger.Info("Migration to version 1.0 completed")
	}

	// Future version migrations can be added here
	// if c.Version == "1.0" && targetVersion == "1.1" { ... }

	logger.Info("Configuration migration completed", "new_version", migrated.Version)
	return migrated, nil
}

// LoadFromFileWithDiscovery loads configuration from file with auto-discovery fallback
func LoadFromFileWithDiscovery(filePath string) (*Config, error) {
	logger := logging.WithField("function", "LoadFromFileWithDiscovery")

	// If specific file is provided, try to load it
	if filePath != "" {
		return LoadFromFile(filePath)
	}

	// Otherwise, try auto-discovery
	discoveredFile, err := AutoDiscoverConfigFile()
	if err != nil {
		return nil, err
	}

	logger.Info("Using auto-discovered configuration file", "file", discoveredFile)
	return LoadFromFile(discoveredFile)
}

// SaveWithBackup saves configuration with automatic backup of existing file
func (c *Config) SaveWithBackup(filePath string) error {
	logger := logging.WithField("function", "SaveWithBackup")
	logger.Info("Saving configuration with backup", "file", filePath)

	// Check if file exists and create backup
	if _, err := os.Stat(filePath); err == nil {
		backupDir := filepath.Dir(filePath)
		timestamp := time.Now().Format("20060102-150405")
		basename := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		backupFile := filepath.Join(backupDir, fmt.Sprintf("%s-backup-%s.json", basename, timestamp))

		// Read existing file
		data, err := os.ReadFile(filePath)
		if err != nil {
			logger.Warn("Failed to read existing file for backup", "error", err)
		} else {
			// Write backup
			if err := os.WriteFile(backupFile, data, 0644); err != nil {
				logger.Warn("Failed to create backup file", "backup", backupFile, "error", err)
			} else {
				logger.Info("Backup created", "backup", backupFile)
			}
		}
	}

	// Save the new configuration
	return c.SaveToFile(filePath)
}

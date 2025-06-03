// Package validation provides input validation logic
// for the GCP Workload Identity Federation CLI tool.
package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// ValidationResult holds the result of validation
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// AddError adds a validation error
func (r *ValidationResult) AddError(field, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{Field: field, Message: message})
}

// Validator provides validation functions
type Validator struct{}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateGCPProjectID validates a GCP project ID
func (v *Validator) ValidateGCPProjectID(projectID string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if projectID == "" {
		result.AddError("project_id", "project ID is required")
		return result
	}

	// GCP project ID rules:
	// - Must be 6-30 characters
	// - Must start with lowercase letter
	// - Can contain lowercase letters, digits, and hyphens
	// - Cannot end with hyphen
	// - Cannot contain consecutive hyphens
	if len(projectID) < 6 || len(projectID) > 30 {
		result.AddError("project_id", "project ID must be 6-30 characters long")
	}

	projectIDRegex := regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
	if !projectIDRegex.MatchString(projectID) {
		result.AddError("project_id", "project ID must start with a lowercase letter, contain only lowercase letters, digits, and hyphens, and not end with a hyphen")
	}

	if strings.Contains(projectID, "--") {
		result.AddError("project_id", "project ID cannot contain consecutive hyphens")
	}

	return result
}

// ValidateGitHubRepository validates a GitHub repository name
func (v *Validator) ValidateGitHubRepository(owner, name string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if owner == "" {
		result.AddError("repository_owner", "repository owner is required")
	} else {
		// GitHub username rules:
		// - Cannot start or end with hyphen
		// - Can contain alphanumeric characters and hyphens
		// - Max 39 characters
		if len(owner) > 39 {
			result.AddError("repository_owner", "repository owner cannot be longer than 39 characters")
		}

		usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
		if !usernameRegex.MatchString(owner) {
			result.AddError("repository_owner", "repository owner must contain only alphanumeric characters and hyphens, and cannot start or end with a hyphen")
		}
	}

	if name == "" {
		result.AddError("repository_name", "repository name is required")
	} else {
		// GitHub repository name rules:
		// - Can contain alphanumeric characters, hyphens, underscores, and periods
		// - Cannot start with period or hyphen
		// - Max 100 characters
		if len(name) > 100 {
			result.AddError("repository_name", "repository name cannot be longer than 100 characters")
		}

		repoNameRegex := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
		if !repoNameRegex.MatchString(name) {
			result.AddError("repository_name", "repository name must start with alphanumeric character and contain only alphanumeric characters, hyphens, underscores, and periods")
		}
	}

	return result
}

// ValidateServiceAccountName validates a GCP service account name
func (v *Validator) ValidateServiceAccountName(name string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if name == "" {
		result.AddError("service_account_name", "service account name is required")
		return result
	}

	// GCP service account name rules:
	// - Must be 6-30 characters
	// - Must start with lowercase letter
	// - Can contain lowercase letters, digits, and hyphens
	// - Cannot end with hyphen
	if len(name) < 6 || len(name) > 30 {
		result.AddError("service_account_name", "service account name must be 6-30 characters long")
	}

	serviceAccountRegex := regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
	if !serviceAccountRegex.MatchString(name) {
		result.AddError("service_account_name", "service account name must start with a lowercase letter, contain only lowercase letters, digits, and hyphens, and not end with a hyphen")
	}

	return result
}

// ValidateWorkloadIdentityPoolID validates a workload identity pool ID
func (v *Validator) ValidateWorkloadIdentityPoolID(poolID string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if poolID == "" {
		result.AddError("workload_identity_pool_id", "workload identity pool ID is required")
		return result
	}

	// Workload identity pool ID rules:
	// - Must be 4-32 characters
	// - Must start with lowercase letter
	// - Can contain lowercase letters, digits, and hyphens
	// - Cannot end with hyphen
	if len(poolID) < 4 || len(poolID) > 32 {
		result.AddError("workload_identity_pool_id", "workload identity pool ID must be 4-32 characters long")
	}

	poolIDRegex := regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
	if !poolIDRegex.MatchString(poolID) {
		result.AddError("workload_identity_pool_id", "workload identity pool ID must start with a lowercase letter, contain only lowercase letters, digits, and hyphens, and not end with a hyphen")
	}

	return result
}

// ValidateWorkloadIdentityProviderID validates a workload identity provider ID
func (v *Validator) ValidateWorkloadIdentityProviderID(providerID string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if providerID == "" {
		result.AddError("workload_identity_provider_id", "workload identity provider ID is required")
		return result
	}

	// Workload identity provider ID rules (similar to pool ID):
	// - Must be 4-32 characters
	// - Must start with lowercase letter
	// - Can contain lowercase letters, digits, and hyphens
	// - Cannot end with hyphen
	if len(providerID) < 4 || len(providerID) > 32 {
		result.AddError("workload_identity_provider_id", "workload identity provider ID must be 4-32 characters long")
	}

	providerIDRegex := regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
	if !providerIDRegex.MatchString(providerID) {
		result.AddError("workload_identity_provider_id", "workload identity provider ID must start with a lowercase letter, contain only lowercase letters, digits, and hyphens, and not end with a hyphen")
	}

	return result
}

// ValidateCloudRunServiceName validates a Cloud Run service name
func (v *Validator) ValidateCloudRunServiceName(serviceName string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if serviceName == "" {
		// Cloud Run service name is optional
		return result
	}

	// Cloud Run service name rules:
	// - Must be 1-63 characters
	// - Must start with lowercase letter
	// - Can contain lowercase letters, digits, and hyphens
	// - Cannot end with hyphen
	if len(serviceName) > 63 {
		result.AddError("cloud_run_service_name", "Cloud Run service name cannot be longer than 63 characters")
	}

	serviceNameRegex := regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
	if !serviceNameRegex.MatchString(serviceName) {
		result.AddError("cloud_run_service_name", "Cloud Run service name must start with a lowercase letter, contain only lowercase letters, digits, and hyphens, and not end with a hyphen")
	}

	return result
}

// ValidateGCPRegion validates a GCP region
func (v *Validator) ValidateGCPRegion(region string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if region == "" {
		// Region is optional in some contexts
		return result
	}

	// GCP region format: lowercase letters and hyphens
	regionRegex := regexp.MustCompile(`^[a-z]+-[a-z]+[0-9]*$`)
	if !regionRegex.MatchString(region) {
		result.AddError("region", "region must be in the format 'area-zone' (e.g., 'us-central1', 'europe-west1')")
	}

	return result
}

// ValidateAll validates all configuration parameters
func (v *Validator) ValidateAll(config map[string]string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Validate project ID
	if projectResult := v.ValidateGCPProjectID(config["project"]); !projectResult.Valid {
		result.Errors = append(result.Errors, projectResult.Errors...)
		result.Valid = false
	}

	// Validate repository
	if repoResult := v.ValidateGitHubRepository(config["repository_owner"], config["repository_name"]); !repoResult.Valid {
		result.Errors = append(result.Errors, repoResult.Errors...)
		result.Valid = false
	}

	// Validate service account name
	if saResult := v.ValidateServiceAccountName(config["service_account_name"]); !saResult.Valid {
		result.Errors = append(result.Errors, saResult.Errors...)
		result.Valid = false
	}

	// Validate workload identity pool ID
	if poolResult := v.ValidateWorkloadIdentityPoolID(config["workload_identity_pool_id"]); !poolResult.Valid {
		result.Errors = append(result.Errors, poolResult.Errors...)
		result.Valid = false
	}

	// Validate workload identity provider ID
	if providerResult := v.ValidateWorkloadIdentityProviderID(config["workload_identity_provider_id"]); !providerResult.Valid {
		result.Errors = append(result.Errors, providerResult.Errors...)
		result.Valid = false
	}

	// Validate Cloud Run service name (optional)
	if crResult := v.ValidateCloudRunServiceName(config["cloud_run_service_name"]); !crResult.Valid {
		result.Errors = append(result.Errors, crResult.Errors...)
		result.Valid = false
	}

	// Validate GCP region (optional)
	if regionResult := v.ValidateGCPRegion(config["cloud_run_region"]); !regionResult.Valid {
		result.Errors = append(result.Errors, regionResult.Errors...)
		result.Valid = false
	}

	return result
}

// GetValidationErrors returns a formatted string of all validation errors
func (r *ValidationResult) GetValidationErrors() string {
	if r.Valid || len(r.Errors) == 0 {
		return ""
	}

	var errors []string
	for _, err := range r.Errors {
		errors = append(errors, err.Error())
	}

	return strings.Join(errors, "\n")
}

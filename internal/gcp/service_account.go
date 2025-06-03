package gcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/Fordjour12/gcp-wif/internal/errors"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
)

// ServiceAccountConfig holds configuration for service account creation
type ServiceAccountConfig struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Roles       []string `json:"roles"`
	CreateNew   bool     `json:"create_new"`
}

// ServiceAccountInfo holds detailed information about a service account
type ServiceAccountInfo struct {
	*iam.ServiceAccount
	ProjectRoles []string  `json:"project_roles"`
	CreatedAt    time.Time `json:"created_at"`
	LastModified time.Time `json:"last_modified"`
	Exists       bool      `json:"exists"`
}

// RoleBinding represents an IAM role binding
type RoleBinding struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}

// DefaultWorkloadIdentityRoles returns default IAM roles for Workload Identity Federation
func DefaultWorkloadIdentityRoles() []string {
	return []string{
		"roles/run.admin",                 // Cloud Run management
		"roles/storage.admin",             // Cloud Storage access
		"roles/artifactregistry.admin",    // Artifact Registry access
		"roles/cloudbuild.builds.builder", // Cloud Build access
		"roles/iam.serviceAccountUser",    // Service account usage
		"roles/iam.workloadIdentityUser",  // Workload Identity access
	}
}

// DefaultMinimalRoles returns minimal IAM roles for basic functionality
func DefaultMinimalRoles() []string {
	return []string{
		"roles/iam.serviceAccountUser",   // Service account usage
		"roles/iam.workloadIdentityUser", // Workload Identity access
	}
}

// ValidateServiceAccountConfig validates service account configuration
func ValidateServiceAccountConfig(config *ServiceAccountConfig) error {
	if config.Name == "" {
		return errors.NewValidationError("Service account name is required")
	}

	// Validate service account name format
	if len(config.Name) < 6 || len(config.Name) > 30 {
		return errors.NewValidationError(
			"Service account name must be 6-30 characters long",
			"Use lowercase letters, digits, and hyphens only",
			"Start with a lowercase letter")
	}

	// Validate roles if provided
	for _, role := range config.Roles {
		if !strings.HasPrefix(role, "roles/") {
			return errors.NewValidationError(
				fmt.Sprintf("Invalid role format: %s", role),
				"Roles must start with 'roles/'",
				"Example: roles/iam.serviceAccountUser")
		}
	}

	return nil
}

// CreateServiceAccount creates a new service account in the project with comprehensive error handling
func (c *Client) CreateServiceAccount(config *ServiceAccountConfig) (*ServiceAccountInfo, error) {
	logger := c.logger.WithField("function", "CreateServiceAccount")
	logger.Info("Creating service account", "name", config.Name, "project_id", c.ProjectID)

	// Validate configuration
	if err := ValidateServiceAccountConfig(config); err != nil {
		return nil, err
	}

	// Enhanced conflict detection
	conflictResult, err := c.DetectAllResourceConflicts(config)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "SA_CONFLICT_DETECTION_FAILED",
			"Failed to detect service account conflicts")
	}

	// Handle conflicts based on results
	if conflictResult.HasConflicts {
		logger.Info("Service account conflicts detected",
			"total", conflictResult.TotalConflicts,
			"critical", conflictResult.CriticalCount,
			"summary", conflictResult.Summary)

		// If critical conflicts and CreateNew is false, return error with suggestions
		if conflictResult.CriticalCount > 0 && !config.CreateNew {
			conflict := conflictResult.Conflicts[0] // Get first critical conflict
			var suggestionTexts []string
			for _, suggestion := range conflict.Suggestions {
				if suggestion.Recommended {
					suggestionTexts = append(suggestionTexts, fmt.Sprintf("✓ %s: %s", suggestion.Title, suggestion.Description))
				} else {
					suggestionTexts = append(suggestionTexts, fmt.Sprintf("• %s: %s", suggestion.Title, suggestion.Description))
				}
			}

			return nil, errors.NewGCPError(
				fmt.Sprintf("Service account %s already exists with critical conflicts", config.Name),
				"Current configuration:",
				fmt.Sprintf("  - Email: %s", conflict.ExistingDetails["email"]),
				fmt.Sprintf("  - Display Name: %s", conflict.ExistingDetails["display_name"]),
				fmt.Sprintf("  - Roles: %v", conflict.ExistingDetails["roles"]),
				"",
				"Resolution options:",
				strings.Join(suggestionTexts, "\n"))
		}

		// If we can proceed (no critical conflicts) or CreateNew is true, get existing and handle appropriately
		if conflictResult.CanProceed || config.CreateNew {
			existing, err := c.GetServiceAccountInfo(config.Name)
			if err != nil {
				return nil, errors.WrapError(err, errors.ErrorTypeGCP, "SA_EXISTENCE_CHECK_FAILED",
					"Failed to check if service account exists")
			}

			if existing != nil && existing.Exists {
				logger.Info("Using existing service account with resolved conflicts", "name", config.Name)

				// Apply any necessary updates based on conflict resolution
				needsUpdate := false
				for _, conflict := range conflictResult.Conflicts {
					for _, diff := range conflict.Differences {
						if diff.Field == "missing_roles" || diff.Field == "display_name" || diff.Field == "description" {
							needsUpdate = true
							break
						}
					}
				}

				if needsUpdate && config.CreateNew {
					logger.Info("Updating existing service account with new configuration", "name", config.Name)
					// Update display name and description if different
					if config.DisplayName != "" && existing.DisplayName != config.DisplayName {
						existing.DisplayName = config.DisplayName
					}
					if config.Description != "" && existing.Description != config.Description {
						existing.Description = config.Description
					}

					// Update service account
					_, err := c.UpdateServiceAccount(config.Name, existing.DisplayName, existing.Description)
					if err != nil {
						logger.Warn("Failed to update service account metadata", "error", err)
					}

					// Grant missing roles
					if len(config.Roles) > 0 {
						if err := c.GrantProjectRoles(existing.Email, config.Roles); err != nil {
							logger.Warn("Failed to grant additional roles", "error", err)
						} else {
							// Refresh service account info to get updated roles
							if refreshed, err := c.GetServiceAccountInfo(config.Name); err == nil {
								existing = refreshed
							}
						}
					}
				}

				return existing, nil
			}
		}
	}

	// Set defaults
	displayName := config.DisplayName
	if displayName == "" {
		displayName = fmt.Sprintf("GitHub Actions SA for %s", config.Name)
	}

	description := config.Description
	if description == "" {
		description = "Service account for GitHub Actions Workload Identity Federation"
	}

	// Create service account request
	request := &iam.CreateServiceAccountRequest{
		AccountId: config.Name,
		ServiceAccount: &iam.ServiceAccount{
			DisplayName: displayName,
			Description: description,
		},
	}

	logger.Debug("Creating service account with IAM API",
		"account_id", config.Name,
		"display_name", displayName,
		"description", description)

	// Create service account
	serviceAccount, err := c.IAMService.Projects.ServiceAccounts.Create(
		fmt.Sprintf("projects/%s", c.ProjectID),
		request,
	).Context(c.ctx).Do()

	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "SA_CREATION_FAILED",
			fmt.Sprintf("Failed to create service account %s", config.Name))
	}

	logger.Info("Service account created successfully",
		"name", serviceAccount.Name,
		"email", serviceAccount.Email,
		"unique_id", serviceAccount.UniqueId)

	// Grant IAM roles if specified
	if len(config.Roles) > 0 {
		logger.Info("Granting IAM roles to service account", "roles", config.Roles)
		if err := c.GrantProjectRoles(serviceAccount.Email, config.Roles); err != nil {
			// Log warning but don't fail - service account was created successfully
			logger.Warn("Failed to grant some IAM roles", "error", err)
		}
	}

	// Return detailed service account information
	return c.GetServiceAccountInfo(config.Name)
}

// GetServiceAccount retrieves an existing service account
func (c *Client) GetServiceAccount(name string) (*iam.ServiceAccount, error) {
	logger := c.logger.WithField("function", "GetServiceAccount")
	logger.Debug("Getting service account", "name", name)

	serviceAccountName := fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com",
		c.ProjectID, name, c.ProjectID)

	serviceAccount, err := c.IAMService.Projects.ServiceAccounts.Get(serviceAccountName).Context(c.ctx).Do()
	if err != nil {
		// Check if it's a 404 error (not found)
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			logger.Debug("Service account not found", "name", name)
			return nil, nil
		}
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "SA_GET_FAILED",
			fmt.Sprintf("Failed to get service account %s", name))
	}

	logger.Debug("Service account retrieved", "email", serviceAccount.Email)
	return serviceAccount, nil
}

// GetServiceAccountInfo retrieves detailed service account information including roles
func (c *Client) GetServiceAccountInfo(name string) (*ServiceAccountInfo, error) {
	logger := c.logger.WithField("function", "GetServiceAccountInfo")
	logger.Debug("Getting detailed service account information", "name", name)

	// Get basic service account info
	serviceAccount, err := c.GetServiceAccount(name)
	if err != nil {
		return nil, err
	}

	if serviceAccount == nil {
		return &ServiceAccountInfo{
			Exists: false,
		}, nil
	}

	// Get project roles for this service account
	roles, err := c.GetServiceAccountProjectRoles(serviceAccount.Email)
	if err != nil {
		logger.Warn("Failed to get service account roles", "error", err)
		roles = []string{} // Continue with empty roles rather than failing
	}

	// Parse creation time
	var createdAt time.Time
	if serviceAccount.Oauth2ClientId != "" {
		// Note: Oauth2ClientId is not the creation time, but we'll use current time as fallback
		createdAt = time.Now()
	}

	info := &ServiceAccountInfo{
		ServiceAccount: serviceAccount,
		ProjectRoles:   roles,
		CreatedAt:      createdAt,
		LastModified:   time.Now(),
		Exists:         true,
	}

	logger.Debug("Service account info retrieved",
		"email", serviceAccount.Email,
		"roles_count", len(roles))

	return info, nil
}

// GetServiceAccountProjectRoles retrieves all project-level roles for a service account
func (c *Client) GetServiceAccountProjectRoles(serviceAccountEmail string) ([]string, error) {
	logger := c.logger.WithField("function", "GetServiceAccountProjectRoles")
	logger.Debug("Getting project roles for service account", "email", serviceAccountEmail)

	// Get current IAM policy
	policy, err := c.ResourceManager.Projects.GetIamPolicy(c.ProjectID, &cloudresourcemanager.GetIamPolicyRequest{}).Context(c.ctx).Do()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "IAM_POLICY_GET_FAILED",
			"Failed to get project IAM policy")
	}

	var roles []string
	member := fmt.Sprintf("serviceAccount:%s", serviceAccountEmail)

	// Find all roles that include this service account
	for _, binding := range policy.Bindings {
		for _, bindingMember := range binding.Members {
			if bindingMember == member {
				roles = append(roles, binding.Role)
				break
			}
		}
	}

	logger.Debug("Found project roles for service account",
		"email", serviceAccountEmail,
		"roles", roles)

	return roles, nil
}

// DeleteServiceAccount deletes a service account with enhanced error handling
func (c *Client) DeleteServiceAccount(name string) error {
	logger := c.logger.WithField("function", "DeleteServiceAccount")
	logger.Info("Deleting service account", "name", name)

	// Check if service account exists
	serviceAccount, err := c.GetServiceAccount(name)
	if err != nil {
		return err
	}

	if serviceAccount == nil {
		logger.Info("Service account does not exist, nothing to delete", "name", name)
		return nil
	}

	// Revoke all project roles first
	roles, err := c.GetServiceAccountProjectRoles(serviceAccount.Email)
	if err != nil {
		logger.Warn("Failed to get service account roles for cleanup", "error", err)
	} else if len(roles) > 0 {
		logger.Info("Revoking project roles before deletion", "roles", roles)
		if err := c.RevokeProjectRoles(serviceAccount.Email, roles); err != nil {
			logger.Warn("Failed to revoke some roles during deletion", "error", err)
		}
	}

	serviceAccountName := fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com",
		c.ProjectID, name, c.ProjectID)

	_, err = c.IAMService.Projects.ServiceAccounts.Delete(serviceAccountName).Context(c.ctx).Do()
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "SA_DELETE_FAILED",
			fmt.Sprintf("Failed to delete service account %s", name))
	}

	logger.Info("Service account deleted successfully", "name", name)
	return nil
}

// GrantProjectRoles grants IAM roles to the service account at the project level with enhanced logic
func (c *Client) GrantProjectRoles(serviceAccountEmail string, roles []string) error {
	logger := c.logger.WithField("function", "GrantProjectRoles")
	logger.Info("Granting project roles to service account",
		"email", serviceAccountEmail,
		"roles", roles)

	if len(roles) == 0 {
		logger.Debug("No roles to grant")
		return nil
	}

	// Validate roles
	for _, role := range roles {
		if !strings.HasPrefix(role, "roles/") {
			return errors.NewValidationError(
				fmt.Sprintf("Invalid role format: %s", role),
				"Roles must start with 'roles/'")
		}
	}

	// Get current IAM policy
	policy, err := c.ResourceManager.Projects.GetIamPolicy(c.ProjectID, &cloudresourcemanager.GetIamPolicyRequest{}).Context(c.ctx).Do()
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "IAM_POLICY_GET_FAILED",
			"Failed to get project IAM policy")
	}

	member := fmt.Sprintf("serviceAccount:%s", serviceAccountEmail)
	modifiedRoles := []string{}

	// Add roles to policy
	for _, role := range roles {
		found := false

		// Check if binding already exists
		for _, binding := range policy.Bindings {
			if binding.Role == role {
				// Check if member is already in the binding
				memberExists := false
				for _, existingMember := range binding.Members {
					if existingMember == member {
						memberExists = true
						break
					}
				}

				if !memberExists {
					binding.Members = append(binding.Members, member)
					modifiedRoles = append(modifiedRoles, role)
				}
				found = true
				break
			}
		}

		// Create new binding if role doesn't exist
		if !found {
			policy.Bindings = append(policy.Bindings, &cloudresourcemanager.Binding{
				Role:    role,
				Members: []string{member},
			})
			modifiedRoles = append(modifiedRoles, role)
		}
	}

	if len(modifiedRoles) == 0 {
		logger.Info("All roles already granted, no changes needed")
		return nil
	}

	logger.Info("Applying IAM policy changes", "modified_roles", modifiedRoles)

	// Set updated IAM policy
	_, err = c.ResourceManager.Projects.SetIamPolicy(c.ProjectID, &cloudresourcemanager.SetIamPolicyRequest{
		Policy: policy,
	}).Context(c.ctx).Do()

	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "IAM_POLICY_SET_FAILED",
			"Failed to update project IAM policy")
	}

	logger.Info("Project roles granted successfully",
		"email", serviceAccountEmail,
		"granted_roles", modifiedRoles)

	return nil
}

// RevokeProjectRoles revokes IAM roles from the service account at the project level with enhanced logic
func (c *Client) RevokeProjectRoles(serviceAccountEmail string, roles []string) error {
	logger := c.logger.WithField("function", "RevokeProjectRoles")
	logger.Info("Revoking project roles from service account",
		"email", serviceAccountEmail,
		"roles", roles)

	if len(roles) == 0 {
		logger.Debug("No roles to revoke")
		return nil
	}

	// Get current IAM policy
	policy, err := c.ResourceManager.Projects.GetIamPolicy(c.ProjectID, &cloudresourcemanager.GetIamPolicyRequest{}).Context(c.ctx).Do()
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "IAM_POLICY_GET_FAILED",
			"Failed to get project IAM policy")
	}

	member := fmt.Sprintf("serviceAccount:%s", serviceAccountEmail)
	modifiedRoles := []string{}

	// Remove roles from policy
	for _, role := range roles {
		for i, binding := range policy.Bindings {
			if binding.Role == role {
				// Remove member from binding
				for j, existingMember := range binding.Members {
					if existingMember == member {
						binding.Members = append(binding.Members[:j], binding.Members[j+1:]...)
						modifiedRoles = append(modifiedRoles, role)
						break
					}
				}

				// Remove binding if no members left
				if len(binding.Members) == 0 {
					policy.Bindings = append(policy.Bindings[:i], policy.Bindings[i+1:]...)
				}
				break
			}
		}
	}

	if len(modifiedRoles) == 0 {
		logger.Info("No roles to revoke, no changes needed")
		return nil
	}

	logger.Info("Applying IAM policy changes", "revoked_roles", modifiedRoles)

	// Set updated IAM policy
	_, err = c.ResourceManager.Projects.SetIamPolicy(c.ProjectID, &cloudresourcemanager.SetIamPolicyRequest{
		Policy: policy,
	}).Context(c.ctx).Do()

	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "IAM_POLICY_SET_FAILED",
			"Failed to update project IAM policy")
	}

	logger.Info("Project roles revoked successfully",
		"email", serviceAccountEmail,
		"revoked_roles", modifiedRoles)

	return nil
}

// ListServiceAccounts lists all service accounts in the project
func (c *Client) ListServiceAccounts() ([]*iam.ServiceAccount, error) {
	logger := c.logger.WithField("function", "ListServiceAccounts")
	logger.Debug("Listing service accounts in project", "project_id", c.ProjectID)

	response, err := c.IAMService.Projects.ServiceAccounts.List(
		fmt.Sprintf("projects/%s", c.ProjectID),
	).Context(c.ctx).Do()

	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "SA_LIST_FAILED",
			"Failed to list service accounts")
	}

	logger.Debug("Service accounts listed", "count", len(response.Accounts))
	return response.Accounts, nil
}

// UpdateServiceAccount updates service account display name and description
func (c *Client) UpdateServiceAccount(name, displayName, description string) (*iam.ServiceAccount, error) {
	logger := c.logger.WithField("function", "UpdateServiceAccount")
	logger.Info("Updating service account", "name", name)

	serviceAccountName := fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com",
		c.ProjectID, name, c.ProjectID)

	// Prepare update request
	updateRequest := &iam.ServiceAccount{
		DisplayName: displayName,
		Description: description,
	}

	serviceAccount, err := c.IAMService.Projects.ServiceAccounts.Update(
		serviceAccountName,
		updateRequest,
	).Context(c.ctx).Do()

	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "SA_UPDATE_FAILED",
			fmt.Sprintf("Failed to update service account %s", name))
	}

	logger.Info("Service account updated successfully", "name", name)
	return serviceAccount, nil
}

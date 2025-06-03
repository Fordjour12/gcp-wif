package gcp

import (
	"fmt"
	"strings"
	"time"
)

// ConflictResolution represents different ways to handle resource conflicts
type ConflictResolution string

const (
	ConflictResolutionSkip      ConflictResolution = "skip"      // Skip creation, use existing
	ConflictResolutionOverwrite ConflictResolution = "overwrite" // Replace existing resource
	ConflictResolutionRename    ConflictResolution = "rename"    // Create with different name
	ConflictResolutionFail      ConflictResolution = "fail"      // Fail with error
	ConflictResolutionBackup    ConflictResolution = "backup"    // Backup existing, then overwrite
)

// ResourceConflict represents a conflict with an existing resource
type ResourceConflict struct {
	ResourceType    string                         `json:"resource_type"`
	ResourceName    string                         `json:"resource_name"`
	ResourceID      string                         `json:"resource_id"`
	ConflictType    string                         `json:"conflict_type"`
	ExistingDetails map[string]interface{}         `json:"existing_details"`
	ProposedDetails map[string]interface{}         `json:"proposed_details"`
	Differences     []ResourceDifference           `json:"differences"`
	Suggestions     []ConflictResolutionSuggestion `json:"suggestions"`
	Severity        ConflictSeverity               `json:"severity"`
	CanAutoResolve  bool                           `json:"can_auto_resolve"`
	CreatedAt       time.Time                      `json:"created_at"`
	LastModified    time.Time                      `json:"last_modified"`
}

// ResourceDifference represents a difference between existing and proposed resources
type ResourceDifference struct {
	Field         string      `json:"field"`
	ExistingValue interface{} `json:"existing_value"`
	ProposedValue interface{} `json:"proposed_value"`
	Severity      string      `json:"severity"` // "critical", "warning", "info"
	Description   string      `json:"description"`
}

// ConflictResolutionSuggestion provides suggestions for resolving conflicts
type ConflictResolutionSuggestion struct {
	Resolution  ConflictResolution `json:"resolution"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Pros        []string           `json:"pros"`
	Cons        []string           `json:"cons"`
	Commands    []string           `json:"commands,omitempty"`
	Automated   bool               `json:"automated"`
	Recommended bool               `json:"recommended"`
}

// ConflictSeverity represents the severity of a resource conflict
type ConflictSeverity string

const (
	ConflictSeverityLow      ConflictSeverity = "low"      // Minor differences, safe to proceed
	ConflictSeverityMedium   ConflictSeverity = "medium"   // Some differences, user should review
	ConflictSeverityHigh     ConflictSeverity = "high"     // Significant differences, caution needed
	ConflictSeverityCritical ConflictSeverity = "critical" // Major conflicts, manual intervention required
)

// ConflictDetectionResult holds results from conflict detection analysis
type ConflictDetectionResult struct {
	HasConflicts      bool               `json:"has_conflicts"`
	Conflicts         []ResourceConflict `json:"conflicts"`
	TotalConflicts    int                `json:"total_conflicts"`
	CriticalCount     int                `json:"critical_count"`
	HighCount         int                `json:"high_count"`
	MediumCount       int                `json:"medium_count"`
	LowCount          int                `json:"low_count"`
	CanProceed        bool               `json:"can_proceed"`
	RecommendedAction string             `json:"recommended_action"`
	Summary           string             `json:"summary"`
}

// ConflictDetectionConfig configures how conflict detection behaves
type ConflictDetectionConfig struct {
	EnableServiceAccounts      bool               `json:"enable_service_accounts"`
	EnableWorkloadIdentity     bool               `json:"enable_workload_identity"`
	EnableCloudRun             bool               `json:"enable_cloud_run"`
	EnableArtifactRegistry     bool               `json:"enable_artifact_registry"`
	DefaultResolution          ConflictResolution `json:"default_resolution"`
	AllowAutomaticResolution   bool               `json:"allow_automatic_resolution"`
	FailOnCriticalConflicts    bool               `json:"fail_on_critical_conflicts"`
	BackupExistingResources    bool               `json:"backup_existing_resources"`
	DetailedDifferenceAnalysis bool               `json:"detailed_difference_analysis"`
}

// DefaultConflictDetectionConfig returns default conflict detection configuration
func DefaultConflictDetectionConfig() *ConflictDetectionConfig {
	return &ConflictDetectionConfig{
		EnableServiceAccounts:      true,
		EnableWorkloadIdentity:     true,
		EnableCloudRun:             true,
		EnableArtifactRegistry:     true,
		DefaultResolution:          ConflictResolutionFail,
		AllowAutomaticResolution:   false,
		FailOnCriticalConflicts:    true,
		BackupExistingResources:    false,
		DetailedDifferenceAnalysis: true,
	}
}

// DetectAllResourceConflicts performs comprehensive conflict detection across all resource types
func (c *Client) DetectAllResourceConflicts(config interface{}) (*ConflictDetectionResult, error) {
	logger := c.logger.WithField("function", "DetectAllResourceConflicts")
	logger.Info("Starting comprehensive resource conflict detection")

	result := &ConflictDetectionResult{
		Conflicts: make([]ResourceConflict, 0),
	}

	// Detect service account conflicts
	switch cfg := config.(type) {
	case *ServiceAccountConfig:
		conflicts, err := c.detectServiceAccountConflicts(cfg)
		if err != nil {
			logger.Error("Service account conflict detection failed", "error", err)
			return nil, err
		}
		result.Conflicts = append(result.Conflicts, conflicts...)

	case *WorkloadIdentityConfig:
		// Detect workload identity conflicts
		wiConflicts, err := c.detectWorkloadIdentityConflicts(cfg)
		if err != nil {
			logger.Error("Workload identity conflict detection failed", "error", err)
			return nil, err
		}
		result.Conflicts = append(result.Conflicts, wiConflicts...)

		// Also check for service account conflicts if specified
		if cfg.ServiceAccountEmail != "" {
			saConfig := &ServiceAccountConfig{
				Name: extractServiceAccountName(cfg.ServiceAccountEmail),
			}
			saConflicts, err := c.detectServiceAccountConflicts(saConfig)
			if err != nil {
				logger.Warn("Service account conflict detection failed", "error", err)
			} else {
				result.Conflicts = append(result.Conflicts, saConflicts...)
			}
		}
	}

	// Analyze and categorize conflicts
	result.TotalConflicts = len(result.Conflicts)
	result.HasConflicts = result.TotalConflicts > 0

	for _, conflict := range result.Conflicts {
		switch conflict.Severity {
		case ConflictSeverityCritical:
			result.CriticalCount++
		case ConflictSeverityHigh:
			result.HighCount++
		case ConflictSeverityMedium:
			result.MediumCount++
		case ConflictSeverityLow:
			result.LowCount++
		}
	}

	// Determine if we can proceed
	result.CanProceed = result.CriticalCount == 0
	result.RecommendedAction = c.determineRecommendedAction(result)
	result.Summary = c.generateConflictSummary(result)

	logger.Info("Conflict detection completed",
		"total_conflicts", result.TotalConflicts,
		"critical", result.CriticalCount,
		"high", result.HighCount,
		"can_proceed", result.CanProceed)

	return result, nil
}

// detectServiceAccountConflicts detects conflicts with existing service accounts
func (c *Client) detectServiceAccountConflicts(config *ServiceAccountConfig) ([]ResourceConflict, error) {
	logger := c.logger.WithField("function", "detectServiceAccountConflicts")
	logger.Debug("Detecting service account conflicts", "name", config.Name)

	var conflicts []ResourceConflict

	// Check if service account exists
	existing, err := c.GetServiceAccountInfo(config.Name)
	if err != nil {
		return nil, err
	}

	if existing != nil && existing.Exists {
		conflict := ResourceConflict{
			ResourceType: "service_account",
			ResourceName: config.Name,
			ResourceID:   existing.Email,
			ConflictType: "resource_exists",
			ExistingDetails: map[string]interface{}{
				"email":        existing.Email,
				"display_name": existing.DisplayName,
				"description":  existing.Description,
				"created_at":   existing.CreatedAt,
				"roles":        existing.ProjectRoles,
			},
			ProposedDetails: map[string]interface{}{
				"name":         config.Name,
				"display_name": config.DisplayName,
				"description":  config.Description,
				"roles":        config.Roles,
			},
			CreatedAt:    existing.CreatedAt,
			LastModified: existing.LastModified,
		}

		// Analyze differences
		conflict.Differences = c.analyzeServiceAccountDifferences(existing, config)
		conflict.Severity = c.determineConflictSeverity(conflict.Differences)
		conflict.CanAutoResolve = conflict.Severity == ConflictSeverityLow
		conflict.Suggestions = c.generateServiceAccountResolutionSuggestions(&conflict, existing, config)

		conflicts = append(conflicts, conflict)
	}

	return conflicts, nil
}

// detectWorkloadIdentityConflicts detects conflicts with existing workload identity resources
func (c *Client) detectWorkloadIdentityConflicts(config *WorkloadIdentityConfig) ([]ResourceConflict, error) {
	logger := c.logger.WithField("function", "detectWorkloadIdentityConflicts")
	logger.Debug("Detecting workload identity conflicts",
		"pool_id", config.PoolID,
		"provider_id", config.ProviderID)

	var conflicts []ResourceConflict

	// Check workload identity pool conflicts
	if config.PoolID != "" {
		poolConflicts, err := c.detectWorkloadIdentityPoolConflicts(config)
		if err != nil {
			return nil, err
		}
		conflicts = append(conflicts, poolConflicts...)
	}

	// Check workload identity provider conflicts
	if config.ProviderID != "" && config.PoolID != "" {
		providerConflicts, err := c.detectWorkloadIdentityProviderConflicts(config)
		if err != nil {
			return nil, err
		}
		conflicts = append(conflicts, providerConflicts...)
	}

	return conflicts, nil
}

// detectWorkloadIdentityPoolConflicts detects conflicts with existing pools
func (c *Client) detectWorkloadIdentityPoolConflicts(config *WorkloadIdentityConfig) ([]ResourceConflict, error) {
	existing, err := c.GetWorkloadIdentityPoolInfo(config.PoolID)
	if err != nil {
		return nil, err
	}

	var conflicts []ResourceConflict

	if existing != nil && existing.Exists {
		conflict := ResourceConflict{
			ResourceType: "workload_identity_pool",
			ResourceName: config.PoolID,
			ResourceID:   existing.FullResourceName,
			ConflictType: "resource_exists",
			ExistingDetails: map[string]interface{}{
				"name":         existing.Name,
				"display_name": existing.DisplayName,
				"description":  existing.Description,
				"state":        existing.State,
				"disabled":     existing.Disabled,
				"created_at":   existing.CreateTime,
			},
			ProposedDetails: map[string]interface{}{
				"pool_id":     config.PoolID,
				"pool_name":   config.PoolName,
				"description": config.PoolDescription,
			},
			CreatedAt:    existing.CreateTime,
			LastModified: existing.CreateTime, // Pools don't have last modified
		}

		// Analyze differences
		conflict.Differences = c.analyzeWorkloadIdentityPoolDifferences(existing, config)
		conflict.Severity = c.determineConflictSeverity(conflict.Differences)
		conflict.CanAutoResolve = conflict.Severity == ConflictSeverityLow
		conflict.Suggestions = c.generateWorkloadIdentityPoolResolutionSuggestions(&conflict, existing, config)

		conflicts = append(conflicts, conflict)
	}

	return conflicts, nil
}

// detectWorkloadIdentityProviderConflicts detects conflicts with existing providers
func (c *Client) detectWorkloadIdentityProviderConflicts(config *WorkloadIdentityConfig) ([]ResourceConflict, error) {
	existing, err := c.GetWorkloadIdentityProviderInfo(config.PoolID, config.ProviderID)
	if err != nil {
		return nil, err
	}

	var conflicts []ResourceConflict

	if existing != nil && existing.Exists {
		conflict := ResourceConflict{
			ResourceType: "workload_identity_provider",
			ResourceName: config.ProviderID,
			ResourceID:   existing.FullResourceName,
			ConflictType: "resource_exists",
			ExistingDetails: map[string]interface{}{
				"name":                existing.Name,
				"display_name":        existing.DisplayName,
				"description":         existing.Description,
				"state":               existing.State,
				"disabled":            existing.Disabled,
				"issuer_uri":          existing.IssuerURI,
				"allowed_audiences":   existing.AllowedAudiences,
				"attribute_mapping":   existing.AttributeMapping,
				"attribute_condition": existing.AttributeCondition,
				"created_at":          existing.CreateTime,
			},
			ProposedDetails: map[string]interface{}{
				"provider_id":   config.ProviderID,
				"provider_name": config.ProviderName,
				"description":   config.ProviderDescription,
				"repository":    config.Repository,
			},
			CreatedAt:    existing.CreateTime,
			LastModified: existing.CreateTime, // Providers don't have last modified
		}

		// Analyze differences
		conflict.Differences = c.analyzeWorkloadIdentityProviderDifferences(existing, config)
		conflict.Severity = c.determineConflictSeverity(conflict.Differences)
		conflict.CanAutoResolve = conflict.Severity == ConflictSeverityLow
		conflict.Suggestions = c.generateWorkloadIdentityProviderResolutionSuggestions(&conflict, existing, config)

		conflicts = append(conflicts, conflict)
	}

	return conflicts, nil
}

// analyzeServiceAccountDifferences analyzes differences between existing and proposed service accounts
func (c *Client) analyzeServiceAccountDifferences(existing *ServiceAccountInfo, proposed *ServiceAccountConfig) []ResourceDifference {
	var differences []ResourceDifference

	// Check display name
	if proposed.DisplayName != "" && existing.DisplayName != proposed.DisplayName {
		differences = append(differences, ResourceDifference{
			Field:         "display_name",
			ExistingValue: existing.DisplayName,
			ProposedValue: proposed.DisplayName,
			Severity:      "info",
			Description:   "Display name will be updated",
		})
	}

	// Check description
	if proposed.Description != "" && existing.Description != proposed.Description {
		differences = append(differences, ResourceDifference{
			Field:         "description",
			ExistingValue: existing.Description,
			ProposedValue: proposed.Description,
			Severity:      "info",
			Description:   "Description will be updated",
		})
	}

	// Check roles
	if len(proposed.Roles) > 0 {
		existingRoles := make(map[string]bool)
		for _, role := range existing.ProjectRoles {
			existingRoles[role] = true
		}

		var missingRoles, extraRoles []string
		for _, role := range proposed.Roles {
			if !existingRoles[role] {
				missingRoles = append(missingRoles, role)
			}
		}

		proposedRoles := make(map[string]bool)
		for _, role := range proposed.Roles {
			proposedRoles[role] = true
		}

		for _, role := range existing.ProjectRoles {
			if !proposedRoles[role] {
				extraRoles = append(extraRoles, role)
			}
		}

		if len(missingRoles) > 0 {
			differences = append(differences, ResourceDifference{
				Field:         "missing_roles",
				ExistingValue: existing.ProjectRoles,
				ProposedValue: missingRoles,
				Severity:      "warning",
				Description:   fmt.Sprintf("Service account is missing %d role(s)", len(missingRoles)),
			})
		}

		if len(extraRoles) > 0 {
			differences = append(differences, ResourceDifference{
				Field:         "extra_roles",
				ExistingValue: extraRoles,
				ProposedValue: proposed.Roles,
				Severity:      "info",
				Description:   fmt.Sprintf("Service account has %d extra role(s)", len(extraRoles)),
			})
		}
	}

	return differences
}

// analyzeWorkloadIdentityPoolDifferences analyzes differences between existing and proposed pools
func (c *Client) analyzeWorkloadIdentityPoolDifferences(existing *WorkloadIdentityPoolInfo, proposed *WorkloadIdentityConfig) []ResourceDifference {
	var differences []ResourceDifference

	// Check display name
	if proposed.PoolName != "" && existing.DisplayName != proposed.PoolName {
		differences = append(differences, ResourceDifference{
			Field:         "display_name",
			ExistingValue: existing.DisplayName,
			ProposedValue: proposed.PoolName,
			Severity:      "warning",
			Description:   "Pool display name differs (cannot be updated)",
		})
	}

	// Check description
	if proposed.PoolDescription != "" && existing.Description != proposed.PoolDescription {
		differences = append(differences, ResourceDifference{
			Field:         "description",
			ExistingValue: existing.Description,
			ProposedValue: proposed.PoolDescription,
			Severity:      "warning",
			Description:   "Pool description differs (cannot be updated)",
		})
	}

	// Check state
	if existing.State != "ACTIVE" {
		differences = append(differences, ResourceDifference{
			Field:         "state",
			ExistingValue: existing.State,
			ProposedValue: "ACTIVE",
			Severity:      "critical",
			Description:   "Pool is not in ACTIVE state",
		})
	}

	// Check if disabled
	if existing.Disabled {
		differences = append(differences, ResourceDifference{
			Field:         "disabled",
			ExistingValue: true,
			ProposedValue: false,
			Severity:      "critical",
			Description:   "Pool is disabled",
		})
	}

	return differences
}

// analyzeWorkloadIdentityProviderDifferences analyzes differences between existing and proposed providers
func (c *Client) analyzeWorkloadIdentityProviderDifferences(existing *WorkloadIdentityProviderInfo, proposed *WorkloadIdentityConfig) []ResourceDifference {
	var differences []ResourceDifference

	// Check display name
	if proposed.ProviderName != "" && existing.DisplayName != proposed.ProviderName {
		differences = append(differences, ResourceDifference{
			Field:         "display_name",
			ExistingValue: existing.DisplayName,
			ProposedValue: proposed.ProviderName,
			Severity:      "warning",
			Description:   "Provider display name differs (cannot be updated)",
		})
	}

	// Check issuer URI
	expectedIssuer := "https://token.actions.githubusercontent.com"
	if proposed.GitHubOIDC != nil {
		expectedIssuer = proposed.GitHubOIDC.IssuerURI
	}
	if existing.IssuerURI != expectedIssuer {
		differences = append(differences, ResourceDifference{
			Field:         "issuer_uri",
			ExistingValue: existing.IssuerURI,
			ProposedValue: expectedIssuer,
			Severity:      "critical",
			Description:   "Provider issuer URI differs (cannot be updated)",
		})
	}

	// Check repository in attribute condition
	if proposed.Repository != "" && !strings.Contains(existing.AttributeCondition, proposed.Repository) {
		differences = append(differences, ResourceDifference{
			Field:         "repository_condition",
			ExistingValue: existing.AttributeCondition,
			ProposedValue: fmt.Sprintf("condition for %s", proposed.Repository),
			Severity:      "critical",
			Description:   "Provider is configured for a different repository",
		})
	}

	return differences
}

// determineConflictSeverity determines the overall severity of a conflict based on its differences
func (c *Client) determineConflictSeverity(differences []ResourceDifference) ConflictSeverity {
	hasCritical := false
	hasWarning := false

	for _, diff := range differences {
		switch diff.Severity {
		case "critical":
			hasCritical = true
		case "warning":
			hasWarning = true
		}
	}

	if hasCritical {
		return ConflictSeverityCritical
	} else if hasWarning {
		return ConflictSeverityMedium
	} else if len(differences) > 0 {
		return ConflictSeverityLow
	}

	return ConflictSeverityLow
}

// generateServiceAccountResolutionSuggestions generates resolution suggestions for service account conflicts
func (c *Client) generateServiceAccountResolutionSuggestions(conflict *ResourceConflict, existing *ServiceAccountInfo, proposed *ServiceAccountConfig) []ConflictResolutionSuggestion {
	var suggestions []ConflictResolutionSuggestion

	// Skip/reuse existing
	suggestions = append(suggestions, ConflictResolutionSuggestion{
		Resolution:  ConflictResolutionSkip,
		Title:       "Use Existing Service Account",
		Description: "Continue with the existing service account without changes",
		Pros:        []string{"No disruption to existing resources", "Faster setup"},
		Cons:        []string{"May not have all required roles", "Configuration differences remain"},
		Automated:   true,
		Recommended: conflict.Severity == ConflictSeverityLow,
	})

	// Update existing
	if len(conflict.Differences) > 0 {
		hasUpdatableFields := false
		for _, diff := range conflict.Differences {
			if diff.Field == "missing_roles" || diff.Field == "display_name" || diff.Field == "description" {
				hasUpdatableFields = true
				break
			}
		}

		if hasUpdatableFields {
			suggestions = append(suggestions, ConflictResolutionSuggestion{
				Resolution:  ConflictResolutionOverwrite,
				Title:       "Update Existing Service Account",
				Description: "Update the existing service account with new configuration",
				Pros:        []string{"Keeps existing service account", "Applies new configuration"},
				Cons:        []string{"May affect other resources using this service account"},
				Commands:    []string{fmt.Sprintf("gcp-wif test-sa --project PROJECT --name %s --update", existing.Name)},
				Automated:   true,
				Recommended: conflict.Severity <= ConflictSeverityMedium,
			})
		}
	}

	// Rename/create new
	suggestions = append(suggestions, ConflictResolutionSuggestion{
		Resolution:  ConflictResolutionRename,
		Title:       "Create New Service Account",
		Description: "Create a new service account with a different name",
		Pros:        []string{"Clean new configuration", "No impact on existing resources"},
		Cons:        []string{"Additional service account to manage", "Need to update names"},
		Commands:    []string{fmt.Sprintf("Use --name %s-new or similar", proposed.Name)},
		Automated:   false,
		Recommended: conflict.Severity == ConflictSeverityCritical,
	})

	return suggestions
}

// generateWorkloadIdentityPoolResolutionSuggestions generates resolution suggestions for pool conflicts
func (c *Client) generateWorkloadIdentityPoolResolutionSuggestions(conflict *ResourceConflict, existing *WorkloadIdentityPoolInfo, proposed *WorkloadIdentityConfig) []ConflictResolutionSuggestion {
	var suggestions []ConflictResolutionSuggestion

	// Use existing if compatible
	if existing.State == "ACTIVE" && !existing.Disabled {
		suggestions = append(suggestions, ConflictResolutionSuggestion{
			Resolution:  ConflictResolutionSkip,
			Title:       "Use Existing Pool",
			Description: "Continue with the existing workload identity pool",
			Pros:        []string{"Pool is active and ready to use", "No configuration changes needed"},
			Cons:        []string{"Display name and description differences remain"},
			Automated:   true,
			Recommended: true,
		})
	}

	// Rename suggestion
	suggestions = append(suggestions, ConflictResolutionSuggestion{
		Resolution:  ConflictResolutionRename,
		Title:       "Create New Pool",
		Description: "Create a new workload identity pool with a different ID",
		Pros:        []string{"Clean new configuration", "Exact specification match"},
		Cons:        []string{"Additional resource to manage", "Need to update pool ID references"},
		Commands:    []string{fmt.Sprintf("Use --pool-id %s-new or similar", proposed.PoolID)},
		Automated:   false,
		Recommended: existing.State != "ACTIVE" || existing.Disabled,
	})

	return suggestions
}

// generateWorkloadIdentityProviderResolutionSuggestions generates resolution suggestions for provider conflicts
func (c *Client) generateWorkloadIdentityProviderResolutionSuggestions(conflict *ResourceConflict, existing *WorkloadIdentityProviderInfo, proposed *WorkloadIdentityConfig) []ConflictResolutionSuggestion {
	var suggestions []ConflictResolutionSuggestion

	// Check if provider is compatible
	isCompatible := existing.IssuerURI == "https://token.actions.githubusercontent.com" &&
		strings.Contains(existing.AttributeCondition, proposed.Repository)

	if isCompatible {
		suggestions = append(suggestions, ConflictResolutionSuggestion{
			Resolution:  ConflictResolutionSkip,
			Title:       "Use Existing Provider",
			Description: "Continue with the existing workload identity provider",
			Pros:        []string{"Provider is configured for GitHub OIDC", "Repository matches"},
			Cons:        []string{"May have different security conditions", "Display name differences"},
			Automated:   true,
			Recommended: true,
		})
	}

	// Always offer rename option
	suggestions = append(suggestions, ConflictResolutionSuggestion{
		Resolution:  ConflictResolutionRename,
		Title:       "Create New Provider",
		Description: "Create a new workload identity provider with a different ID",
		Pros:        []string{"Exact specification match", "Custom security conditions"},
		Cons:        []string{"Additional resource to manage", "Need to update provider ID references"},
		Commands:    []string{fmt.Sprintf("Use --provider-id %s-new or similar", proposed.ProviderID)},
		Automated:   false,
		Recommended: !isCompatible,
	})

	return suggestions
}

// determineRecommendedAction determines the recommended action based on conflict analysis
func (c *Client) determineRecommendedAction(result *ConflictDetectionResult) string {
	if !result.HasConflicts {
		return "Proceed with resource creation"
	}

	if result.CriticalCount > 0 {
		return "Review critical conflicts before proceeding"
	}

	if result.HighCount > 0 {
		return "Review high-priority conflicts and consider alternative names"
	}

	if result.MediumCount > 0 {
		return "Review configuration differences and proceed with caution"
	}

	return "Minor conflicts detected, safe to proceed"
}

// generateConflictSummary generates a human-readable summary of conflicts
func (c *Client) generateConflictSummary(result *ConflictDetectionResult) string {
	if !result.HasConflicts {
		return "No resource conflicts detected"
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Found %d resource conflict(s): ", result.TotalConflicts))

	var parts []string
	if result.CriticalCount > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", result.CriticalCount))
	}
	if result.HighCount > 0 {
		parts = append(parts, fmt.Sprintf("%d high", result.HighCount))
	}
	if result.MediumCount > 0 {
		parts = append(parts, fmt.Sprintf("%d medium", result.MediumCount))
	}
	if result.LowCount > 0 {
		parts = append(parts, fmt.Sprintf("%d low", result.LowCount))
	}

	summary.WriteString(strings.Join(parts, ", "))
	return summary.String()
}

// extractServiceAccountName extracts service account name from email
func extractServiceAccountName(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return email
}

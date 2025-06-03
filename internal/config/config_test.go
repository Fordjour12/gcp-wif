package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", config.Version)
	}

	if config.Project.Region != "us-central1" {
		t.Errorf("Expected default region us-central1, got %s", config.Project.Region)
	}

	if len(config.ServiceAccount.Roles) == 0 {
		t.Error("Expected default roles to be set")
	}

	if config.CloudRun.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", config.CloudRun.Port)
	}
}

func TestNewConfig(t *testing.T) {
	config := NewConfig("test-project-123", "testowner", "test-repo")

	if config.Project.ID != "test-project-123" {
		t.Errorf("Expected project ID test-project-123, got %s", config.Project.ID)
	}

	if config.Repository.Owner != "testowner" {
		t.Errorf("Expected repository owner testowner, got %s", config.Repository.Owner)
	}

	if config.Repository.Name != "test-repo" {
		t.Errorf("Expected repository name test-repo, got %s", config.Repository.Name)
	}

	// Check generated service account name
	expectedSAName := "github-testowner-test-repo"
	if config.ServiceAccount.Name != expectedSAName {
		t.Errorf("Expected service account name %s, got %s", expectedSAName, config.ServiceAccount.Name)
	}

	// Check that workload identity IDs are properly shortened
	if len(config.WorkloadIdentity.PoolID) > 32 {
		t.Errorf("Pool ID too long: %s (%d chars)", config.WorkloadIdentity.PoolID, len(config.WorkloadIdentity.PoolID))
	}
	if len(config.WorkloadIdentity.ProviderID) > 32 {
		t.Errorf("Provider ID too long: %s (%d chars)", config.WorkloadIdentity.ProviderID, len(config.WorkloadIdentity.ProviderID))
	}
}

func TestValidateSchema_ValidConfig(t *testing.T) {
	config := NewConfig("test-project-123", "testowner", "test-repo")

	result := config.ValidateSchema()

	if !result.Valid {
		t.Errorf("Expected valid configuration, but got errors: %v", result.Errors)
	}

	if len(result.Errors) > 0 {
		t.Errorf("Expected no validation errors, got %d", len(result.Errors))
	}
}

func TestValidateSchema_InvalidProjectID(t *testing.T) {
	config := DefaultConfig()
	config.Project.ID = "INVALID-PROJECT" // Uppercase not allowed
	config.Repository.Owner = "testowner"
	config.Repository.Name = "test-repo"
	config.ServiceAccount.Name = "test-service-account"
	config.WorkloadIdentity.PoolID = "test-pool"
	config.WorkloadIdentity.ProviderID = "test-provider"

	result := config.ValidateSchema()

	if result.Valid {
		t.Error("Expected invalid configuration due to invalid project ID")
	}

	found := false
	for _, err := range result.Errors {
		if err.Field == "project.id" && err.Code == "INVALID_FORMAT" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected validation error for invalid project ID format")
	}
}

func TestValidateSchema_InvalidRepositoryOwner(t *testing.T) {
	config := DefaultConfig()
	config.Project.ID = "test-project-123"
	config.Repository.Owner = "-invalid" // Cannot start with hyphen
	config.Repository.Name = "test-repo"
	config.ServiceAccount.Name = "test-service-account"
	config.WorkloadIdentity.PoolID = "test-pool"
	config.WorkloadIdentity.ProviderID = "test-provider"

	result := config.ValidateSchema()

	if result.Valid {
		t.Error("Expected invalid configuration due to invalid repository owner")
	}

	found := false
	for _, err := range result.Errors {
		if err.Field == "repository.owner" && err.Code == "INVALID_FORMAT" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected validation error for invalid repository owner format")
	}
}

func TestValidateSchema_MissingRequiredFields(t *testing.T) {
	config := &Config{
		Version: "1.0",
		// Missing all required fields
	}

	result := config.ValidateSchema()

	if result.Valid {
		t.Error("Expected invalid configuration due to missing required fields")
	}

	// Should have errors for missing project ID, repository owner, repository name, service account name, etc.
	if len(result.Errors) < 5 {
		t.Errorf("Expected at least 5 validation errors for missing required fields, got %d", len(result.Errors))
	}
}

func TestValidateSchema_CloudRunPortValidation(t *testing.T) {
	config := NewConfig("test-project-123", "testowner", "test-repo")
	config.CloudRun.ServiceName = "test-service"
	config.CloudRun.Port = 99999 // Invalid port

	result := config.ValidateSchema()

	if result.Valid {
		t.Error("Expected invalid configuration due to invalid port")
	}

	found := false
	for _, err := range result.Errors {
		if err.Field == "cloud_run.port" && err.Code == "INVALID_RANGE" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected validation error for invalid port range")
	}
}

func TestValidateSchema_WorkflowFilenameWarning(t *testing.T) {
	config := NewConfig("test-project-123", "testowner", "test-repo")
	config.Workflow.Filename = "deploy.txt" // Should warn about extension

	result := config.ValidateSchema()

	if !result.Valid {
		t.Errorf("Expected valid configuration, but got errors: %v", result.Errors)
	}

	found := false
	for _, warning := range result.Warnings {
		if warning.Field == "workflow.filename" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning for workflow filename without .yml/.yaml extension")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	// Create and save config
	originalConfig := NewConfig("test-project-123", "testowner", "test-repo")
	originalConfig.CloudRun.ServiceName = "test-service"
	originalConfig.Workflow.Environment = "production"

	err := originalConfig.SaveToFile(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config back
	loadedConfig, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded config matches original
	if loadedConfig.Project.ID != originalConfig.Project.ID {
		t.Errorf("Project ID mismatch: expected %s, got %s", originalConfig.Project.ID, loadedConfig.Project.ID)
	}

	if loadedConfig.Repository.Owner != originalConfig.Repository.Owner {
		t.Errorf("Repository owner mismatch: expected %s, got %s", originalConfig.Repository.Owner, loadedConfig.Repository.Owner)
	}

	if loadedConfig.CloudRun.ServiceName != originalConfig.CloudRun.ServiceName {
		t.Errorf("Cloud Run service name mismatch: expected %s, got %s", originalConfig.CloudRun.ServiceName, loadedConfig.CloudRun.ServiceName)
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	_, err := LoadFromFile("non-existent-config.json")

	if err == nil {
		t.Error("Expected error when loading non-existent config file")
	}

	// Check that it's a configuration error
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestGetHelperMethods(t *testing.T) {
	config := NewConfig("test-project-123", "testowner", "test-repo")
	config.CloudRun.ServiceName = "test-service"
	config.CloudRun.Region = "us-west1"

	// Test GetRepoFullName
	expected := "testowner/test-repo"
	if config.GetRepoFullName() != expected {
		t.Errorf("Expected repo full name %s, got %s", expected, config.GetRepoFullName())
	}

	// Test GetServiceAccountEmail
	expectedEmail := "github-testowner-test-repo@test-project-123.iam.gserviceaccount.com"
	if config.GetServiceAccountEmail() != expectedEmail {
		t.Errorf("Expected service account email %s, got %s", expectedEmail, config.GetServiceAccountEmail())
	}

	// Test GetCloudRunURL
	expectedURL := "https://test-service-us-west1.a.run.app"
	if config.GetCloudRunURL() != expectedURL {
		t.Errorf("Expected Cloud Run URL %s, got %s", expectedURL, config.GetCloudRunURL())
	}

	// Test GetWorkflowFilePath
	expectedPath := ".github/workflows/deploy.yml"
	if config.GetWorkflowFilePath() != expectedPath {
		t.Errorf("Expected workflow file path %s, got %s", expectedPath, config.GetWorkflowFilePath())
	}
}

func TestToJSONAndFromJSON(t *testing.T) {
	originalConfig := NewConfig("test-project-123", "testowner", "test-repo")
	originalConfig.CloudRun.ServiceName = "test-service"

	// Convert to JSON
	jsonStr, err := originalConfig.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert config to JSON: %v", err)
	}

	if jsonStr == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Convert back from JSON
	loadedConfig, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("Failed to create config from JSON: %v", err)
	}

	// Verify configs match
	if loadedConfig.Project.ID != originalConfig.Project.ID {
		t.Errorf("Project ID mismatch after JSON round-trip: expected %s, got %s",
			originalConfig.Project.ID, loadedConfig.Project.ID)
	}

	if loadedConfig.CloudRun.ServiceName != originalConfig.CloudRun.ServiceName {
		t.Errorf("Cloud Run service name mismatch after JSON round-trip: expected %s, got %s",
			originalConfig.CloudRun.ServiceName, loadedConfig.CloudRun.ServiceName)
	}
}

func TestDefaultRoles(t *testing.T) {
	roles := DefaultRoles()

	expectedRoles := []string{
		"roles/run.admin",
		"roles/storage.admin",
		"roles/artifactregistry.admin",
	}

	if len(roles) != len(expectedRoles) {
		t.Errorf("Expected %d default roles, got %d", len(expectedRoles), len(roles))
	}

	for i, expected := range expectedRoles {
		if roles[i] != expected {
			t.Errorf("Expected role %s at index %d, got %s", expected, i, roles[i])
		}
	}
}

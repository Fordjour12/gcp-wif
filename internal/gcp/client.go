// Package gcp provides GCP API client wrappers and authentication
// for the GCP Workload Identity Federation CLI tool.
package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/logging"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/iamcredentials/v1"
	"google.golang.org/api/option"
)

// AuthInfo contains information about the current authentication
type AuthInfo struct {
	Account      string    `json:"account"`
	Status       string    `json:"status"`
	Type         string    `json:"type"`
	ProjectID    string    `json:"project_id"`
	QuotaProject string    `json:"quota_project_id"`
	HasADC       bool      `json:"has_adc"`
	LastRefresh  time.Time `json:"last_refresh"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// ProjectInfo contains detailed project information
type ProjectInfo struct {
	ProjectID      string            `json:"projectId"`
	ProjectNumber  string            `json:"projectNumber"`
	Name           string            `json:"name"`
	LifecycleState string            `json:"lifecycleState"`
	Labels         map[string]string `json:"labels,omitempty"`
	CreateTime     string            `json:"createTime"`
}

// Client wraps the GCP API clients with authentication
type Client struct {
	ctx         context.Context
	logger      *logging.Logger
	authInfo    *AuthInfo
	projectInfo *ProjectInfo

	// GCP Service clients
	IAMService      *iam.Service
	ResourceManager *cloudresourcemanager.Service
	IAMCredentials  *iamcredentials.Service

	ProjectID string
}

// ClientConfig holds configuration for creating a GCP client
type ClientConfig struct {
	ProjectID  string
	RequireADC bool     // Require Application Default Credentials
	Scopes     []string // Custom OAuth scopes
	UserAgent  string   // Custom user agent
}

// NewClient creates a new GCP client using gcloud CLI authentication
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	config := &ClientConfig{
		ProjectID: projectID,
		UserAgent: "gcp-wif-cli/1.0",
	}
	return NewClientWithConfig(ctx, config)
}

// NewClientWithConfig creates a new GCP client with custom configuration
func NewClientWithConfig(ctx context.Context, config *ClientConfig) (*Client, error) {
	logger := logging.WithField("component", "gcp_client")
	logger.Info("Initializing GCP client", "project_id", config.ProjectID)

	if config.ProjectID == "" {
		return nil, errors.NewValidationError("Project ID is required")
	}

	// Verify gcloud CLI is installed and authenticated
	authInfo, err := checkGCloudAuth()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeAuthentication, "GCLOUD_AUTH_FAILED",
			"gcloud CLI authentication verification failed")
	}

	logger.Info("gcloud authentication verified", "account", authInfo.Account, "type", authInfo.Type)

	// Validate project access
	projectInfo, err := validateProjectAccess(ctx, config.ProjectID)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "PROJECT_ACCESS_FAILED",
			fmt.Sprintf("Failed to validate access to project %s", config.ProjectID))
	}

	logger.Info("Project access validated", "project_id", projectInfo.ProjectID, "name", projectInfo.Name)

	// Set up OAuth scopes
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			iam.CloudPlatformScope,
			cloudresourcemanager.CloudPlatformScope,
		}
	}

	// Create client options
	clientOptions := []option.ClientOption{
		option.WithScopes(scopes...),
	}

	if config.UserAgent != "" {
		clientOptions = append(clientOptions, option.WithUserAgent(config.UserAgent))
	}

	// Create service clients
	iamService, err := iam.NewService(ctx, clientOptions...)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "IAM_CLIENT_FAILED",
			"Failed to create IAM service client")
	}

	resourceManager, err := cloudresourcemanager.NewService(ctx, clientOptions...)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "RESOURCE_MANAGER_CLIENT_FAILED",
			"Failed to create Resource Manager service client")
	}

	iamCredentials, err := iamcredentials.NewService(ctx, clientOptions...)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "IAM_CREDENTIALS_CLIENT_FAILED",
			"Failed to create IAM Credentials service client")
	}

	client := &Client{
		ctx:             ctx,
		logger:          logger,
		authInfo:        authInfo,
		projectInfo:     projectInfo,
		IAMService:      iamService,
		ResourceManager: resourceManager,
		IAMCredentials:  iamCredentials,
		ProjectID:       config.ProjectID,
	}

	logger.Info("GCP client initialized successfully")
	return client, nil
}

// checkGCloudAuth verifies that gcloud CLI is installed and authenticated
func checkGCloudAuth() (*AuthInfo, error) {
	logger := logging.WithField("function", "checkGCloudAuth")

	// Check if gcloud is installed
	gcloudPath, err := exec.LookPath("gcloud")
	if err != nil {
		return nil, errors.NewAuthenticationError(
			"gcloud CLI is not installed or not in PATH",
			"Install the Google Cloud CLI: https://cloud.google.com/sdk/docs/install",
			"Ensure gcloud is available in your system PATH",
			"Restart your terminal after installation")
	}

	logger.Debug("gcloud CLI found", "path", gcloudPath)

	// Check gcloud version
	versionCmd := exec.Command("gcloud", "version", "--format=json")
	versionOutput, err := versionCmd.Output()
	if err != nil {
		logger.Warn("Could not get gcloud version", "error", err)
	} else {
		logger.Debug("gcloud version retrieved", "output_length", len(versionOutput))
	}

	// Get current active account
	cmd := exec.Command("gcloud", "auth", "list", "--filter=status:ACTIVE", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.NewAuthenticationError(
			"Failed to check gcloud authentication status",
			"Run 'gcloud auth login' to authenticate",
			"Ensure you have the necessary permissions",
			"Check your network connectivity")
	}

	var accounts []map[string]interface{}
	if err := json.Unmarshal(output, &accounts); err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeAuthentication, "AUTH_PARSE_FAILED",
			"Failed to parse gcloud auth output")
	}

	if len(accounts) == 0 {
		return nil, errors.NewAuthenticationError(
			"No active gcloud authentication found",
			"Run 'gcloud auth login' to authenticate with your Google account",
			"For service accounts, use 'gcloud auth activate-service-account'",
			"Verify your authentication with 'gcloud auth list'")
	}

	account := accounts[0]
	authInfo := &AuthInfo{
		Account:     fmt.Sprintf("%v", account["account"]),
		Status:      fmt.Sprintf("%v", account["status"]),
		Type:        fmt.Sprintf("%v", account["type"]),
		LastRefresh: time.Now(),
	}

	// Check for Application Default Credentials
	authInfo.HasADC = checkApplicationDefaultCredentials()

	// Get current project
	projectCmd := exec.Command("gcloud", "config", "get-value", "project")
	projectOutput, err := projectCmd.Output()
	if err == nil {
		authInfo.ProjectID = strings.TrimSpace(string(projectOutput))
	}

	// Get quota project
	quotaCmd := exec.Command("gcloud", "auth", "application-default", "print-access-token", "--verbosity=none")
	if quotaCmd.Run() == nil {
		authInfo.QuotaProject = authInfo.ProjectID
	}

	logger.Info("Authentication verified",
		"account", authInfo.Account,
		"type", authInfo.Type,
		"has_adc", authInfo.HasADC,
		"project", authInfo.ProjectID)

	return authInfo, nil
}

// checkApplicationDefaultCredentials checks if ADC is properly configured
func checkApplicationDefaultCredentials() bool {
	// Check for ADC environment variable
	if credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credPath != "" {
		if _, err := os.Stat(credPath); err == nil {
			return true
		}
	}

	// Try to run ADC command
	cmd := exec.Command("gcloud", "auth", "application-default", "print-access-token", "--verbosity=none")
	return cmd.Run() == nil
}

// validateProjectAccess validates that the user has access to the specified project
func validateProjectAccess(ctx context.Context, projectID string) (*ProjectInfo, error) {
	logger := logging.WithField("function", "validateProjectAccess")
	logger.Debug("Validating project access", "project_id", projectID)

	// Create a temporary Resource Manager client for validation
	resourceManager, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "could not find default credentials") {
			return nil, errors.NewAuthenticationError(
				"Failed to create temporary Resource Manager client: Application Default Credentials (ADC) not found.",
				"Please run 'gcloud auth application-default login' to set up ADC and try again.",
				"For more information, visit: https://cloud.google.com/docs/authentication/external/set-up-adc",
			)
		}
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "TEMP_CLIENT_FAILED",
			"Failed to create temporary Resource Manager client")
	}

	// Try to get project information
	project, err := resourceManager.Projects.Get(projectID).Context(ctx).Do()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "PROJECT_GET_FAILED",
			fmt.Sprintf("Failed to get project %s - check project ID and permissions", projectID))
	}

	projectInfo := &ProjectInfo{
		ProjectID:      project.ProjectId,
		ProjectNumber:  fmt.Sprintf("%d", project.ProjectNumber),
		Name:           project.Name,
		LifecycleState: project.LifecycleState,
		Labels:         project.Labels,
		CreateTime:     project.CreateTime,
	}

	// Verify project is active
	if project.LifecycleState != "ACTIVE" {
		return nil, errors.NewGCPError(
			fmt.Sprintf("Project %s is not active (state: %s)", projectID, project.LifecycleState),
			"Ensure the project is active and not deleted",
			"Check the project status in the Google Cloud Console")
	}

	logger.Info("Project validation successful",
		"project_id", projectInfo.ProjectID,
		"project_number", projectInfo.ProjectNumber,
		"name", projectInfo.Name,
		"state", projectInfo.LifecycleState)

	return projectInfo, nil
}

// GetAuthInfo returns the current authentication information
func (c *Client) GetAuthInfo() *AuthInfo {
	return c.authInfo
}

// GetProjectInfo returns the current project information
func (c *Client) GetProjectInfo() *ProjectInfo {
	return c.projectInfo
}

// GetProject retrieves project information using the client
func (c *Client) GetProject() (*cloudresourcemanager.Project, error) {
	c.logger.Debug("Getting project information", "project_id", c.ProjectID)

	project, err := c.ResourceManager.Projects.Get(c.ProjectID).Context(c.ctx).Do()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "PROJECT_GET_FAILED",
			fmt.Sprintf("Failed to get project %s", c.ProjectID))
	}

	c.logger.Debug("Project information retrieved successfully",
		"project_id", project.ProjectId,
		"name", project.Name,
		"state", project.LifecycleState)

	return project, nil
}

// TestConnection tests the connection to GCP services
func (c *Client) TestConnection() error {
	c.logger.Info("Testing GCP service connections")

	// Test Resource Manager
	_, err := c.GetProject()
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "CONNECTION_TEST_FAILED",
			"Resource Manager connection test failed")
	}

	// Test IAM Service by trying to list service accounts
	request := c.IAMService.Projects.ServiceAccounts.List(fmt.Sprintf("projects/%s", c.ProjectID))
	_, err = request.Context(c.ctx).Do()
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "IAM_CONNECTION_TEST_FAILED",
			"IAM service connection test failed")
	}

	c.logger.Info("All GCP service connections verified successfully")
	return nil
}

// CheckPermissions checks if the authenticated user has specific permissions
func (c *Client) CheckPermissions(permissions []string) (map[string]bool, error) {
	c.logger.Debug("Checking IAM permissions", "permissions", permissions, "project_id", c.ProjectID)

	testRequest := &cloudresourcemanager.TestIamPermissionsRequest{
		Permissions: permissions,
	}

	response, err := c.ResourceManager.Projects.TestIamPermissions(c.ProjectID, testRequest).Context(c.ctx).Do()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeGCP, "PERMISSION_CHECK_FAILED",
			"Failed to check IAM permissions")
	}

	// Create result map
	result := make(map[string]bool)
	permissionSet := make(map[string]bool)
	for _, perm := range response.Permissions {
		permissionSet[perm] = true
	}

	for _, perm := range permissions {
		result[perm] = permissionSet[perm]
	}

	c.logger.Debug("Permission check completed", "result", result)
	return result, nil
}

// RefreshAuth refreshes the authentication token
func (c *Client) RefreshAuth() error {
	c.logger.Info("Refreshing gcloud authentication")

	// Run gcloud auth application-default print-access-token to refresh
	cmd := exec.Command("gcloud", "auth", "application-default", "print-access-token")
	if err := cmd.Run(); err != nil {
		return errors.WrapError(err, errors.ErrorTypeAuthentication, "AUTH_REFRESH_FAILED",
			"Failed to refresh authentication token")
	}

	// Update auth info
	authInfo, err := checkGCloudAuth()
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeAuthentication, "AUTH_RECHECK_FAILED",
			"Failed to verify authentication after refresh")
	}

	c.authInfo = authInfo
	c.authInfo.LastRefresh = time.Now()

	c.logger.Info("Authentication refreshed successfully")
	return nil
}

// Close cleans up the client resources
func (c *Client) Close() error {
	c.logger.Debug("Closing GCP client")
	// Currently no cleanup needed for Google API clients
	return nil
}

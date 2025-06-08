package wif

import (
	"fmt"
	"os/exec"
	"strings"
)

// Client handles GCP Workload Identity Federation operations
type Client struct {
	projectID string
}

// NewClient creates a new WIF client
func NewClient(projectID string) *Client {
	return &Client{
		projectID: projectID,
	}
}

// Setup orchestrates the complete WIF setup process
func (c *Client) Setup(config *Config) error {
	fmt.Println("🔧 Starting Workload Identity Federation setup...")

	// Step 1: Check if pool exists, create if not
	if err := c.ensurePool(config); err != nil {
		return fmt.Errorf("failed to setup workload identity pool: %w", err)
	}

	// Step 2: Check if provider exists, create if not
	if err := c.ensureProvider(config); err != nil {
		return fmt.Errorf("failed to setup workload identity provider: %w", err)
	}

	// Step 3: Bind service account
	if err := c.bindServiceAccount(config); err != nil {
		return fmt.Errorf("failed to bind service account: %w", err)
	}

	// Success!
	c.printSuccess(config)
	return nil
}

func (c *Client) ensurePool(config *Config) error {
	fmt.Printf("   🏊 Checking workload identity pool: %s\n", config.PoolID)

	// Check if pool exists
	exists, err := c.poolExists(config.PoolID)
	if err != nil {
		return err
	}

	if exists {
		fmt.Printf("   ✅ Pool already exists: %s\n", config.PoolID)
		return nil
	}

	// Create pool
	fmt.Printf("   🆕 Creating workload identity pool: %s\n", config.PoolID)
	return c.createPool(config)
}

func (c *Client) ensureProvider(config *Config) error {
	fmt.Printf("   🔌 Checking workload identity provider: %s\n", config.ProviderID)

	// Check if provider exists
	exists, err := c.providerExists(config.PoolID, config.ProviderID)
	if err != nil {
		return err
	}

	if exists {
		fmt.Printf("   ✅ Provider already exists: %s\n", config.ProviderID)
		return nil
	}

	// Create provider
	fmt.Printf("   🆕 Creating workload identity provider: %s\n", config.ProviderID)
	return c.createProvider(config)
}

func (c *Client) poolExists(poolID string) (bool, error) {
	cmd := exec.Command("gcloud", "iam", "workload-identity-pools", "describe", poolID,
		"--project", c.projectID,
		"--location", "global",
		"--format", "value(name)")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a NOT_FOUND error
		if strings.Contains(string(output), "NOT_FOUND") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check pool existence: %s", string(output))
	}

	return strings.TrimSpace(string(output)) != "", nil
}

func (c *Client) providerExists(poolID, providerID string) (bool, error) {
	cmd := exec.Command("gcloud", "iam", "workload-identity-pools", "providers", "describe", providerID,
		"--project", c.projectID,
		"--location", "global",
		"--workload-identity-pool", poolID,
		"--format", "value(name)")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a NOT_FOUND error
		if strings.Contains(string(output), "NOT_FOUND") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check provider existence: %s", string(output))
	}

	return strings.TrimSpace(string(output)) != "", nil
}

func (c *Client) createPool(config *Config) error {
	displayName := fmt.Sprintf("GitHub Actions Pool for %s", config.Repository)
	if len(displayName) > 32 {
		displayName = fmt.Sprintf("GitHub Pool: %s", config.GetRepositoryOwner())
		if len(displayName) > 32 {
			displayName = displayName[:32]
		}
	}

	cmd := exec.Command("gcloud", "iam", "workload-identity-pools", "create", config.PoolID,
		"--project", c.projectID,
		"--location", "global",
		"--display-name", displayName,
		"--description", fmt.Sprintf("Workload identity pool for GitHub repository %s", config.Repository))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create pool: %s", string(output))
	}

	return nil
}

func (c *Client) createProvider(config *Config) error {
	displayName := fmt.Sprintf("GitHub OIDC for %s", config.Repository)
	if len(displayName) > 32 {
		displayName = fmt.Sprintf("GitHub: %s", config.GetRepositoryOwner())
		if len(displayName) > 32 {
			displayName = displayName[:32]
		}
	}

	// Build attribute mapping (essential claims only)
	attributeMapping := strings.Join([]string{
		"google.subject=assertion.sub",
		"attribute.actor=assertion.actor",
		"attribute.repository=assertion.repository",
		"attribute.repository_owner=assertion.repository_owner",
		"attribute.ref=assertion.ref",
		"attribute.pull_request=assertion.pull_request",
	}, ",")

	// Build security condition
	condition := fmt.Sprintf("assertion.repository=='%s'", config.Repository)

	// Get audiences (GitHub-specific + STS fallback)
	audiences := fmt.Sprintf("%s,sts.googleapis.com", config.GetGitHubAudience())

	cmd := exec.Command("gcloud", "iam", "workload-identity-pools", "providers", "create-oidc", config.ProviderID,
		"--project", c.projectID,
		"--location", "global",
		"--workload-identity-pool", config.PoolID,
		"--display-name", displayName,
		"--description", fmt.Sprintf("GitHub OIDC provider for repository %s", config.Repository),
		"--issuer-uri", "https://token.actions.githubusercontent.com",
		"--allowed-audiences", audiences,
		"--attribute-mapping", attributeMapping,
		"--attribute-condition", condition)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create provider: %s", string(output))
	}

	return nil
}

func (c *Client) bindServiceAccount(config *Config) error {
	fmt.Printf("   🔗 Binding service account: %s\n", config.SAEmail)

	// Build principal set member
	member := fmt.Sprintf("principalSet://iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/%s/providers/%s/*",
		c.projectID, config.PoolID, config.ProviderID)

	// Create IAM binding with condition
	condition := fmt.Sprintf("attribute.repository=='%s'", config.Repository)

	cmd := exec.Command("gcloud", "iam", "service-accounts", "add-iam-policy-binding", config.SAEmail,
		"--project", c.projectID,
		"--role", "roles/iam.workloadIdentityUser",
		"--member", member,
		"--condition", fmt.Sprintf("expression=%s,title=GitHub Actions Access,description=Allow GitHub Actions from %s to impersonate this service account", condition, config.Repository))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to bind service account: %s", string(output))
	}

	return nil
}

func (c *Client) printSuccess(config *Config) {
	fmt.Println()
	fmt.Println("🎉 Workload Identity Federation setup completed successfully!")
	fmt.Println()
	fmt.Println("📋 Summary:")
	fmt.Printf("   Project: %s\n", config.Project)
	fmt.Printf("   Repository: %s\n", config.Repository)
	fmt.Printf("   Pool: %s\n", config.PoolID)
	fmt.Printf("   Provider: %s\n", config.ProviderID)
	fmt.Printf("   Service Account: %s\n", config.SAEmail)
	fmt.Println()
	fmt.Println("🚀 Next Steps:")
	fmt.Println("   Add this to your GitHub Actions workflow:")
	fmt.Println()
	fmt.Printf("   - name: Authenticate to Google Cloud\n")
	fmt.Printf("     uses: google-github-actions/auth@v1\n")
	fmt.Printf("     with:\n")
	fmt.Printf("       workload_identity_provider: projects/%s/locations/global/workloadIdentityPools/%s/providers/%s\n", c.projectID, config.PoolID, config.ProviderID)
	fmt.Printf("       service_account: %s\n", config.SAEmail)
	fmt.Println()
}

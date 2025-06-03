package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/gcp"
	"github.com/Fordjour12/gcp-wif/internal/logging"
	"github.com/spf13/cobra"
)

var (
	testWIFProjectID      string
	testWIFPoolID         string
	testWIFPoolName       string
	testWIFPoolDesc       string
	testWIFProviderID     string
	testWIFProviderName   string
	testWIFProviderDesc   string
	testWIFRepository     string
	testWIFServiceAccount string
	testWIFBranches       []string
	testWIFTags           []string
	testWIFAllowPR        bool
	testWIFCreatePool     bool
	testWIFCreateProvider bool
	testWIFBind           bool
	testWIFDelete         bool
	testWIFList           bool
	testWIFInfo           bool
	testWIFAudiences      []string
	testWIFBlockForked    bool
	testWIFRequireActor   bool
	testWIFValidateToken  bool
	testWIFTrustedRepos   []string
	testWIFValidateOIDC   bool
)

// testWIFCmd represents the test-wif command
var testWIFCmd = &cobra.Command{
	Use:   "test-wif",
	Short: "Test Workload Identity Federation setup and management",
	Long: `Test Workload Identity Federation pools, providers, and bindings for GitHub Actions.

This command allows you to test various Workload Identity Federation operations:
1. Create workload identity pools and providers
2. List existing pools and providers
3. Get detailed information about pools and providers
4. Bind service accounts to workload identity providers
5. Configure security conditions (branches, tags, pull requests)
6. Delete pools and providers

Examples:
  # List all workload identity pools
  gcp-wif test-wif --project my-project --list

  # Create a pool and provider for a GitHub repository
  gcp-wif test-wif --project my-project --pool-id my-pool --provider-id github-provider \
    --repository owner/repo --create-pool --create-provider

  # Create with custom security conditions and GitHub OIDC features
  gcp-wif test-wif --project my-project --pool-id my-pool --provider-id github-provider \
    --repository owner/repo --branches main,develop --tags "v*" --allow-pr \
    --audiences sts.googleapis.com --block-forked --require-actor \
    --create-pool --create-provider

  # Create with enhanced security for multiple repositories
  gcp-wif test-wif --project my-project --pool-id my-pool --provider-id github-provider \
    --repository owner/repo --trusted-repos owner/repo1,owner/repo2 \
    --validate-token-path --create-pool --create-provider

  # Get detailed pool and provider information with OIDC validation
  gcp-wif test-wif --project my-project --pool-id my-pool --provider-id github-provider \
    --info --validate-oidc

  # Bind service account to workload identity
  gcp-wif test-wif --project my-project --pool-id my-pool --provider-id github-provider \
    --repository owner/repo --service-account sa@project.iam.gserviceaccount.com --bind

  # Delete a provider
  gcp-wif test-wif --project my-project --pool-id my-pool --provider-id github-provider --delete`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTestWorkloadIdentity(cmd, args); err != nil {
			HandleError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(testWIFCmd)

	testWIFCmd.Flags().StringVarP(&testWIFProjectID, "project", "p", "", "Google Cloud Project ID (required)")
	testWIFCmd.Flags().StringVar(&testWIFPoolID, "pool-id", "", "Workload identity pool ID")
	testWIFCmd.Flags().StringVar(&testWIFPoolName, "pool-name", "", "Workload identity pool display name")
	testWIFCmd.Flags().StringVar(&testWIFPoolDesc, "pool-description", "", "Workload identity pool description")
	testWIFCmd.Flags().StringVar(&testWIFProviderID, "provider-id", "", "Workload identity provider ID")
	testWIFCmd.Flags().StringVar(&testWIFProviderName, "provider-name", "", "Workload identity provider display name")
	testWIFCmd.Flags().StringVar(&testWIFProviderDesc, "provider-description", "", "Workload identity provider description")
	testWIFCmd.Flags().StringVarP(&testWIFRepository, "repository", "r", "", "GitHub repository (owner/name)")
	testWIFCmd.Flags().StringVar(&testWIFServiceAccount, "service-account", "", "Service account email for binding")
	testWIFCmd.Flags().StringSliceVar(&testWIFBranches, "branches", nil, "Allowed branches (comma-separated)")
	testWIFCmd.Flags().StringSliceVar(&testWIFTags, "tags", nil, "Allowed tags (comma-separated)")
	testWIFCmd.Flags().BoolVar(&testWIFAllowPR, "allow-pr", false, "Allow pull request workflows")
	testWIFCmd.Flags().BoolVar(&testWIFCreatePool, "create-pool", false, "Create workload identity pool")
	testWIFCmd.Flags().BoolVar(&testWIFCreateProvider, "create-provider", false, "Create workload identity provider")
	testWIFCmd.Flags().BoolVar(&testWIFBind, "bind", false, "Bind service account to workload identity")
	testWIFCmd.Flags().BoolVar(&testWIFDelete, "delete", false, "Delete pool or provider")
	testWIFCmd.Flags().BoolVar(&testWIFList, "list", false, "List workload identity pools")
	testWIFCmd.Flags().BoolVar(&testWIFInfo, "info", false, "Get detailed pool/provider information")
	testWIFCmd.Flags().StringSliceVar(&testWIFAudiences, "audiences", []string{"sts.googleapis.com"}, "Allowed OIDC audiences")
	testWIFCmd.Flags().BoolVar(&testWIFBlockForked, "block-forked", true, "Block forked repository access")
	testWIFCmd.Flags().BoolVar(&testWIFRequireActor, "require-actor", true, "Require actor claim in tokens")
	testWIFCmd.Flags().BoolVar(&testWIFValidateToken, "validate-token-path", true, "Validate workflow token path")
	testWIFCmd.Flags().StringSliceVar(&testWIFTrustedRepos, "trusted-repos", nil, "Additional trusted repositories")
	testWIFCmd.Flags().BoolVar(&testWIFValidateOIDC, "validate-oidc", false, "Validate OIDC configuration and token format")

	testWIFCmd.MarkFlagRequired("project")
}

func runTestWorkloadIdentity(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "test_workload_identity")
	logger.Info("Starting workload identity test", "project_id", testWIFProjectID)

	ctx := context.Background()

	fmt.Println("🔧 Testing Workload Identity Federation")
	fmt.Println("======================================")
	fmt.Printf("📋 Project ID: %s\n\n", testWIFProjectID)

	// Create GCP client
	client, err := gcp.NewClient(ctx, testWIFProjectID)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "CLIENT_CREATION_FAILED",
			"Failed to create GCP client")
	}
	defer client.Close()

	// Handle list operation
	if testWIFList {
		return listWorkloadIdentityPools(client)
	}

	// Handle info operation
	if testWIFInfo {
		return getWorkloadIdentityInfo(client, testWIFPoolID, testWIFProviderID)
	}

	// Handle delete operation
	if testWIFDelete {
		return deleteWorkloadIdentity(client, testWIFPoolID, testWIFProviderID)
	}

	// Handle bind operation
	if testWIFBind {
		return bindServiceAccountToWorkloadIdentity(client)
	}

	// Handle create operations
	if testWIFCreatePool || testWIFCreateProvider {
		return createWorkloadIdentityResources(client)
	}

	// Default: Show help
	return errors.NewValidationError(
		"No operation specified",
		"Use --list to see all workload identity pools",
		"Use --create-pool or --create-provider to create resources",
		"Use --info to get detailed information",
		"Use --help to see all available options")
}

func listWorkloadIdentityPools(client *gcp.Client) error {
	fmt.Println("📋 Listing Workload Identity Pools...")

	pools, err := client.ListWorkloadIdentityPools()
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "WI_POOLS_LIST_FAILED",
			"Failed to list workload identity pools")
	}

	if len(pools) == 0 {
		fmt.Println("✅ No workload identity pools found in project")
		fmt.Println("\n💡 Create a new pool with:")
		fmt.Println("   gcp-wif test-wif --project PROJECT_ID --pool-id my-pool --create-pool")
		return nil
	}

	fmt.Printf("✅ Found %d workload identity pools:\n\n", len(pools))

	for i, pool := range pools {
		fmt.Printf("%d. %s\n", i+1, pool.DisplayName)
		fmt.Printf("   Pool ID: %s\n", extractResourceID(pool.Name))
		fmt.Printf("   Full Name: %s\n", pool.Name)
		fmt.Printf("   Description: %s\n", pool.Description)
		fmt.Printf("   State: %s\n", pool.State)
		fmt.Printf("   Disabled: %t\n", pool.Disabled)
		if !pool.CreateTime.IsZero() {
			fmt.Printf("   Created: %s\n", pool.CreateTime.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}

	fmt.Println("💡 Next steps:")
	fmt.Println("   • Use --pool-id POOL_ID --info to get detailed pool information")
	fmt.Println("   • Create a provider with --pool-id POOL_ID --provider-id PROVIDER_ID --create-provider")

	return nil
}

func getWorkloadIdentityInfo(client *gcp.Client, poolID, providerID string) error {
	if poolID == "" {
		return errors.NewValidationError(
			"Pool ID is required for info operation",
			"Use --pool-id to specify the pool ID")
	}

	fmt.Printf("🔍 Getting Workload Identity Information\n\n")

	// Get pool information
	fmt.Printf("📋 Pool Information: %s\n", poolID)
	poolInfo, err := client.GetWorkloadIdentityPoolInfo(poolID)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "WI_POOL_INFO_GET_FAILED",
			fmt.Sprintf("Failed to get pool info for %s", poolID))
	}

	if !poolInfo.Exists {
		fmt.Printf("❌ Workload identity pool '%s' does not exist\n", poolID)
		fmt.Println("\n💡 Suggestions:")
		fmt.Println("   • Use --list to see all available pools")
		fmt.Println("   • Use --create-pool to create a new pool")
		return nil
	}

	fmt.Printf("   Name: %s\n", poolInfo.Name)
	fmt.Printf("   Display Name: %s\n", poolInfo.DisplayName)
	fmt.Printf("   Description: %s\n", poolInfo.Description)
	fmt.Printf("   State: %s\n", poolInfo.State)
	fmt.Printf("   Disabled: %t\n", poolInfo.Disabled)
	if !poolInfo.CreateTime.IsZero() {
		fmt.Printf("   Created: %s\n", poolInfo.CreateTime.Format("2006-01-02 15:04:05"))
	}

	// Get provider information if specified
	if providerID != "" {
		fmt.Printf("\n🔗 Provider Information: %s\n", providerID)
		providerInfo, err := client.GetWorkloadIdentityProviderInfo(poolID, providerID)
		if err != nil {
			return errors.WrapError(err, errors.ErrorTypeGCP, "WI_PROVIDER_INFO_GET_FAILED",
				fmt.Sprintf("Failed to get provider info for %s", providerID))
		}

		if !providerInfo.Exists {
			fmt.Printf("❌ Workload identity provider '%s' does not exist\n", providerID)
			fmt.Println("\n💡 Suggestions:")
			fmt.Println("   • Create the provider with --create-provider")
			return nil
		}

		fmt.Printf("   Name: %s\n", providerInfo.Name)
		fmt.Printf("   Display Name: %s\n", providerInfo.DisplayName)
		fmt.Printf("   Description: %s\n", providerInfo.Description)
		fmt.Printf("   State: %s\n", providerInfo.State)
		fmt.Printf("   Disabled: %t\n", providerInfo.Disabled)
		fmt.Printf("   Issuer URI: %s\n", providerInfo.IssuerURI)
		fmt.Printf("   Allowed Audiences: %s\n", strings.Join(providerInfo.AllowedAudiences, ", "))
		fmt.Printf("   Attribute Condition: %s\n", providerInfo.AttributeCondition)

		if len(providerInfo.AttributeMapping) > 0 {
			fmt.Printf("   Attribute Mapping:\n")
			for k, v := range providerInfo.AttributeMapping {
				fmt.Printf("     • %s = %s\n", k, v)
			}
		}

		if !providerInfo.CreateTime.IsZero() {
			fmt.Printf("   Created: %s\n", providerInfo.CreateTime.Format("2006-01-02 15:04:05"))
		}

		// Show provider name for GitHub Actions
		providerName := client.GetWorkloadIdentityProviderName(poolID, providerID)
		fmt.Printf("\n🎯 Use this provider in GitHub Actions:\n")
		fmt.Printf("   Provider: %s\n", providerName)

		// Validate OIDC configuration if requested
		if testWIFValidateOIDC {
			fmt.Printf("\n🔍 Validating GitHub OIDC Configuration:\n")
			oidcConfig, err := client.GetGitHubOIDCConfiguration(poolID, providerID)
			if err != nil {
				fmt.Printf("❌ Failed to get OIDC configuration: %v\n", err)
			} else {
				fmt.Printf("   ✅ Issuer URI: %s\n", oidcConfig.IssuerURI)
				fmt.Printf("   ✅ Allowed Audiences: %s\n", strings.Join(oidcConfig.AllowedAudiences, ", "))

				// Validate token format (example)
				exampleToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJyZXBvOm93bmVyL3JlcG8iLCJhdWQiOiJzdHMuZ29vZ2xlYXBpcy5jb20ifQ.signature"
				if err := client.ValidateGitHubOIDCToken(exampleToken, testWIFRepository); err != nil {
					fmt.Printf("   ❌ Token validation: %v\n", err)
				} else {
					fmt.Printf("   ✅ Token format validation passed\n")
				}
			}
		}
	}

	return nil
}

func createWorkloadIdentityResources(client *gcp.Client) error {
	fmt.Println("🔨 Creating Workload Identity Resources...")

	if testWIFRepository == "" {
		return errors.NewValidationError(
			"Repository is required for creation",
			"Use --repository to specify the GitHub repository (owner/name)")
	}

	config := &gcp.WorkloadIdentityConfig{
		PoolID:              testWIFPoolID,
		PoolName:            testWIFPoolName,
		PoolDescription:     testWIFPoolDesc,
		ProviderID:          testWIFProviderID,
		ProviderName:        testWIFProviderName,
		ProviderDescription: testWIFProviderDesc,
		Repository:          testWIFRepository,
		AllowedBranches:     testWIFBranches,
		AllowedTags:         testWIFTags,
		AllowPullRequests:   testWIFAllowPR,
		CreateNew:           true,
		GitHubOIDC: &gcp.GitHubOIDCConfig{
			IssuerURI:         "https://token.actions.githubusercontent.com",
			AllowedAudiences:  testWIFAudiences,
			DefaultAudience:   "sts.googleapis.com",
			ValidateTokenPath: testWIFValidateToken,
			RequireActor:      testWIFRequireActor,
			TrustedRepos:      testWIFTrustedRepos,
			BlockForkedRepos:  testWIFBlockForked,
		},
		ClaimsMapping: gcp.GetDefaultGitHubClaimsMapping(),
	}

	// Create pool if requested
	if testWIFCreatePool {
		if testWIFPoolID == "" {
			return errors.NewValidationError(
				"Pool ID is required for pool creation",
				"Use --pool-id to specify the pool ID")
		}

		fmt.Printf("📋 Creating Workload Identity Pool: %s\n", testWIFPoolID)
		fmt.Printf("   Repository: %s\n", testWIFRepository)
		if testWIFPoolName != "" {
			fmt.Printf("   Display Name: %s\n", testWIFPoolName)
		}
		if testWIFPoolDesc != "" {
			fmt.Printf("   Description: %s\n", testWIFPoolDesc)
		}
		fmt.Println()

		poolInfo, err := client.CreateWorkloadIdentityPool(config)
		if err != nil {
			return errors.WrapError(err, errors.ErrorTypeGCP, "WI_POOL_CREATION_FAILED",
				fmt.Sprintf("Failed to create workload identity pool %s", testWIFPoolID))
		}

		fmt.Printf("✅ Workload Identity Pool Created Successfully!\n")
		fmt.Printf("   Name: %s\n", poolInfo.Name)
		fmt.Printf("   Display Name: %s\n", poolInfo.DisplayName)
		fmt.Printf("   State: %s\n", poolInfo.State)
		fmt.Println()
	}

	// Create provider if requested
	if testWIFCreateProvider {
		if testWIFPoolID == "" || testWIFProviderID == "" {
			return errors.NewValidationError(
				"Pool ID and Provider ID are required for provider creation",
				"Use --pool-id and --provider-id to specify the IDs")
		}

		fmt.Printf("🔗 Creating Workload Identity Provider: %s\n", testWIFProviderID)
		fmt.Printf("   Pool: %s\n", testWIFPoolID)
		fmt.Printf("   Repository: %s\n", testWIFRepository)
		fmt.Printf("   OIDC Issuer: https://token.actions.githubusercontent.com\n")
		fmt.Printf("   Audiences: %s\n", strings.Join(testWIFAudiences, ", "))
		if len(testWIFBranches) > 0 {
			fmt.Printf("   Allowed Branches: %s\n", strings.Join(testWIFBranches, ", "))
		}
		if len(testWIFTags) > 0 {
			fmt.Printf("   Allowed Tags: %s\n", strings.Join(testWIFTags, ", "))
		}
		if testWIFAllowPR {
			fmt.Printf("   Allow Pull Requests: Yes\n")
		}
		if testWIFBlockForked {
			fmt.Printf("   Block Forked Repos: Yes\n")
		}
		if testWIFRequireActor {
			fmt.Printf("   Require Actor: Yes\n")
		}
		if testWIFValidateToken {
			fmt.Printf("   Validate Token Path: Yes\n")
		}
		if len(testWIFTrustedRepos) > 0 {
			fmt.Printf("   Trusted Repos: %s\n", strings.Join(testWIFTrustedRepos, ", "))
		}
		fmt.Println()

		providerInfo, err := client.CreateWorkloadIdentityProvider(config)
		if err != nil {
			return errors.WrapError(err, errors.ErrorTypeGCP, "WI_PROVIDER_CREATION_FAILED",
				fmt.Sprintf("Failed to create workload identity provider %s", testWIFProviderID))
		}

		fmt.Printf("✅ Workload Identity Provider Created Successfully!\n")
		fmt.Printf("   Name: %s\n", providerInfo.Name)
		fmt.Printf("   Display Name: %s\n", providerInfo.DisplayName)
		fmt.Printf("   State: %s\n", providerInfo.State)
		fmt.Printf("   Issuer URI: %s\n", providerInfo.IssuerURI)
		fmt.Printf("   Attribute Condition: %s\n", providerInfo.AttributeCondition)

		// Show provider name for GitHub Actions
		providerName := client.GetWorkloadIdentityProviderName(testWIFPoolID, testWIFProviderID)
		fmt.Printf("\n🎯 Use this provider in GitHub Actions:\n")
		fmt.Printf("   Provider: %s\n", providerName)
	}

	fmt.Println("\n💡 Next steps:")
	fmt.Println("   • Bind a service account with --bind --service-account SA_EMAIL")
	fmt.Println("   • Use the provider in your GitHub Actions workflow")
	fmt.Println("   • Test authentication from GitHub Actions")

	return nil
}

func bindServiceAccountToWorkloadIdentity(client *gcp.Client) error {
	fmt.Println("🔗 Binding Service Account to Workload Identity...")

	if testWIFPoolID == "" || testWIFProviderID == "" || testWIFRepository == "" || testWIFServiceAccount == "" {
		return errors.NewValidationError(
			"Pool ID, Provider ID, Repository, and Service Account are required for binding",
			"Use --pool-id, --provider-id, --repository, and --service-account flags")
	}

	config := &gcp.WorkloadIdentityConfig{
		PoolID:              testWIFPoolID,
		ProviderID:          testWIFProviderID,
		Repository:          testWIFRepository,
		ServiceAccountEmail: testWIFServiceAccount,
	}

	fmt.Printf("   Pool: %s\n", testWIFPoolID)
	fmt.Printf("   Provider: %s\n", testWIFProviderID)
	fmt.Printf("   Repository: %s\n", testWIFRepository)
	fmt.Printf("   Service Account: %s\n", testWIFServiceAccount)
	fmt.Println()

	if err := client.BindServiceAccountToWorkloadIdentity(config); err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "WI_BINDING_FAILED",
			"Failed to bind service account to workload identity")
	}

	fmt.Printf("✅ Service Account Bound Successfully!\n\n")
	fmt.Println("💡 Your GitHub Actions can now impersonate this service account using:")
	fmt.Printf("   Provider: %s\n", client.GetWorkloadIdentityProviderName(testWIFPoolID, testWIFProviderID))
	fmt.Printf("   Service Account: %s\n", testWIFServiceAccount)

	return nil
}

func deleteWorkloadIdentity(client *gcp.Client, poolID, providerID string) error {
	if poolID == "" {
		return errors.NewValidationError(
			"Pool ID is required for deletion",
			"Use --pool-id to specify the pool ID")
	}

	// Delete provider if specified
	if providerID != "" {
		fmt.Printf("🗑️ Deleting Workload Identity Provider: %s\n", providerID)
		fmt.Printf("   Pool: %s\n", poolID)
		fmt.Println()

		if err := client.DeleteWorkloadIdentityProvider(poolID, providerID); err != nil {
			return errors.WrapError(err, errors.ErrorTypeGCP, "WI_PROVIDER_DELETE_FAILED",
				fmt.Sprintf("Failed to delete workload identity provider %s", providerID))
		}

		fmt.Printf("✅ Workload Identity Provider Deleted Successfully!\n")
		return nil
	}

	// Delete pool
	fmt.Printf("🗑️ Deleting Workload Identity Pool: %s\n", poolID)
	fmt.Println("⚠️  This will also delete all providers in the pool")
	fmt.Println()

	if err := client.DeleteWorkloadIdentityPool(poolID); err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "WI_POOL_DELETE_FAILED",
			fmt.Sprintf("Failed to delete workload identity pool %s", poolID))
	}

	fmt.Printf("✅ Workload Identity Pool Deleted Successfully!\n")
	fmt.Println("💡 All providers in the pool have also been deleted")

	return nil
}

// extractResourceID extracts the resource ID from a full resource name
func extractResourceID(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

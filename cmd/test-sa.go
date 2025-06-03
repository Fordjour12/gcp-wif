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
	testSAProjectID string
	testSAName      string
	testSADisplay   string
	testSADesc      string
	testSARoles     []string
	testSACreateNew bool
	testSADelete    bool
	testSAList      bool
	testSAUpdate    bool
)

// testSACmd represents the test-sa command
var testSACmd = &cobra.Command{
	Use:   "test-sa",
	Short: "Test service account creation and management",
	Long: `Test service account creation, role assignment, and management operations.

This command allows you to test various service account operations:
1. Create new service accounts with IAM roles
2. List existing service accounts
3. Get detailed service account information
4. Grant and revoke project-level IAM roles
5. Update service account metadata
6. Delete service accounts

Examples:
  # Create a new service account with default WIF roles
  gcp-wif test-sa --project my-project --name my-sa --create

  # Create with custom roles
  gcp-wif test-sa --project my-project --name my-sa --create --roles roles/iam.serviceAccountUser,roles/run.admin

  # List all service accounts
  gcp-wif test-sa --project my-project --list

  # Get info about specific service account
  gcp-wif test-sa --project my-project --name my-sa

  # Update service account metadata
  gcp-wif test-sa --project my-project --name my-sa --update --display "Updated Display Name"

  # Delete service account
  gcp-wif test-sa --project my-project --name my-sa --delete`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTestServiceAccount(cmd, args); err != nil {
			HandleError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(testSACmd)

	testSACmd.Flags().StringVarP(&testSAProjectID, "project", "p", "", "Google Cloud Project ID (required)")
	testSACmd.Flags().StringVarP(&testSAName, "name", "n", "", "Service account name (required for most operations)")
	testSACmd.Flags().StringVar(&testSADisplay, "display", "", "Service account display name")
	testSACmd.Flags().StringVar(&testSADesc, "description", "", "Service account description")
	testSACmd.Flags().StringSliceVar(&testSARoles, "roles", nil, "IAM roles to grant (comma-separated)")
	testSACmd.Flags().BoolVar(&testSACreateNew, "create", false, "Create a new service account")
	testSACmd.Flags().BoolVar(&testSADelete, "delete", false, "Delete the service account")
	testSACmd.Flags().BoolVar(&testSAList, "list", false, "List all service accounts in the project")
	testSACmd.Flags().BoolVar(&testSAUpdate, "update", false, "Update service account metadata")

	testSACmd.MarkFlagRequired("project")
}

func runTestServiceAccount(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "test_service_account")
	logger.Info("Starting service account test", "project_id", testSAProjectID)

	ctx := context.Background()

	fmt.Println("🔧 Testing Service Account Operations")
	fmt.Println("====================================")
	fmt.Printf("📋 Project ID: %s\n\n", testSAProjectID)

	// Create GCP client
	client, err := gcp.NewClient(ctx, testSAProjectID)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "CLIENT_CREATION_FAILED",
			"Failed to create GCP client")
	}
	defer client.Close()

	// Handle list operation
	if testSAList {
		return listServiceAccounts(client)
	}

	// Handle delete operation
	if testSADelete {
		return deleteServiceAccount(client, testSAName)
	}

	// Handle update operation
	if testSAUpdate {
		return updateServiceAccount(client, testSAName, testSADisplay, testSADesc)
	}

	// Handle create operation
	if testSACreateNew {
		return createServiceAccount(client, testSAName, testSADisplay, testSADesc, testSARoles)
	}

	// Default: Get service account info
	if testSAName == "" {
		return errors.NewValidationError(
			"Service account name is required",
			"Use --name to specify the service account name",
			"Use --list to see all service accounts")
	}

	return getServiceAccountInfo(client, testSAName)
}

func listServiceAccounts(client *gcp.Client) error {
	fmt.Println("📋 Listing Service Accounts...")

	accounts, err := client.ListServiceAccounts()
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "SA_LIST_FAILED",
			"Failed to list service accounts")
	}

	if len(accounts) == 0 {
		fmt.Println("✅ No service accounts found in project")
		return nil
	}

	fmt.Printf("✅ Found %d service accounts:\n\n", len(accounts))

	for i, account := range accounts {
		fmt.Printf("%d. %s\n", i+1, account.DisplayName)
		fmt.Printf("   Email: %s\n", account.Email)
		fmt.Printf("   Description: %s\n", account.Description)
		fmt.Printf("   Unique ID: %s\n", account.UniqueId)
		fmt.Printf("   Disabled: %t\n", account.Disabled)

		// Get roles for this service account
		roles, err := client.GetServiceAccountProjectRoles(account.Email)
		if err != nil {
			fmt.Printf("   Roles: Failed to get roles (%v)\n", err)
		} else if len(roles) > 0 {
			fmt.Printf("   Roles: %s\n", strings.Join(roles, ", "))
		} else {
			fmt.Printf("   Roles: None\n")
		}
		fmt.Println()
	}

	return nil
}

func getServiceAccountInfo(client *gcp.Client, name string) error {
	fmt.Printf("🔍 Getting Service Account Info: %s\n", name)

	info, err := client.GetServiceAccountInfo(name)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "SA_INFO_GET_FAILED",
			fmt.Sprintf("Failed to get service account info for %s", name))
	}

	if !info.Exists {
		fmt.Printf("❌ Service account '%s' does not exist\n", name)
		fmt.Println("\n💡 Suggestions:")
		fmt.Println("   • Use --list to see all service accounts")
		fmt.Println("   • Use --create to create a new service account")
		return nil
	}

	fmt.Printf("✅ Service Account Found:\n\n")
	fmt.Printf("   Name: %s\n", info.Name)
	fmt.Printf("   Email: %s\n", info.Email)
	fmt.Printf("   Display Name: %s\n", info.DisplayName)
	fmt.Printf("   Description: %s\n", info.Description)
	fmt.Printf("   Unique ID: %s\n", info.UniqueId)
	fmt.Printf("   Disabled: %t\n", info.Disabled)
	fmt.Printf("   Created: %s\n", info.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Last Modified: %s\n", info.LastModified.Format("2006-01-02 15:04:05"))

	if len(info.ProjectRoles) > 0 {
		fmt.Printf("   Project Roles (%d):\n", len(info.ProjectRoles))
		for _, role := range info.ProjectRoles {
			fmt.Printf("     • %s\n", role)
		}
	} else {
		fmt.Printf("   Project Roles: None\n")
	}

	return nil
}

func createServiceAccount(client *gcp.Client, name, displayName, description string, roles []string) error {
	fmt.Printf("🔨 Creating Service Account: %s\n", name)

	if name == "" {
		return errors.NewValidationError(
			"Service account name is required for creation",
			"Use --name to specify the service account name")
	}

	// Use default roles if none specified
	if len(roles) == 0 {
		roles = gcp.DefaultWorkloadIdentityRoles()
		fmt.Printf("🔧 Using default Workload Identity roles (%d roles)\n", len(roles))
	}

	config := &gcp.ServiceAccountConfig{
		Name:        name,
		DisplayName: displayName,
		Description: description,
		Roles:       roles,
		CreateNew:   true,
	}

	fmt.Printf("   • Name: %s\n", config.Name)
	fmt.Printf("   • Display Name: %s\n", config.DisplayName)
	fmt.Printf("   • Description: %s\n", config.Description)
	fmt.Printf("   • Roles: %s\n", strings.Join(config.Roles, ", "))
	fmt.Println()

	fmt.Println("Creating service account...")

	info, err := client.CreateServiceAccount(config)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "SA_CREATION_FAILED",
			fmt.Sprintf("Failed to create service account %s", name))
	}

	fmt.Printf("✅ Service Account Created Successfully!\n\n")
	fmt.Printf("   Email: %s\n", info.Email)
	fmt.Printf("   Unique ID: %s\n", info.UniqueId)

	if len(info.ProjectRoles) > 0 {
		fmt.Printf("   Granted Roles (%d):\n", len(info.ProjectRoles))
		for _, role := range info.ProjectRoles {
			fmt.Printf("     • %s\n", role)
		}
	}

	fmt.Println("\n💡 Next steps:")
	fmt.Println("   • Configure Workload Identity Federation for this service account")
	fmt.Println("   • Add any additional IAM roles as needed")
	fmt.Printf("   • Use this email in your GitHub Actions: %s\n", info.Email)

	return nil
}

func updateServiceAccount(client *gcp.Client, name, displayName, description string) error {
	fmt.Printf("🔄 Updating Service Account: %s\n", name)

	if name == "" {
		return errors.NewValidationError(
			"Service account name is required for update",
			"Use --name to specify the service account name")
	}

	// Check if service account exists
	existing, err := client.GetServiceAccount(name)
	if err != nil {
		return err
	}

	if existing == nil {
		return errors.NewGCPError(
			fmt.Sprintf("Service account '%s' does not exist", name),
			"Use --list to see existing service accounts",
			"Use --create to create a new service account")
	}

	// Use existing values if not provided
	if displayName == "" {
		displayName = existing.DisplayName
	}
	if description == "" {
		description = existing.Description
	}

	fmt.Printf("   • Current Display Name: %s\n", existing.DisplayName)
	fmt.Printf("   • New Display Name: %s\n", displayName)
	fmt.Printf("   • Current Description: %s\n", existing.Description)
	fmt.Printf("   • New Description: %s\n", description)
	fmt.Println()

	fmt.Println("Updating service account...")

	updated, err := client.UpdateServiceAccount(name, displayName, description)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "SA_UPDATE_FAILED",
			fmt.Sprintf("Failed to update service account %s", name))
	}

	fmt.Printf("✅ Service Account Updated Successfully!\n\n")
	fmt.Printf("   Email: %s\n", updated.Email)
	fmt.Printf("   Display Name: %s\n", updated.DisplayName)
	fmt.Printf("   Description: %s\n", updated.Description)

	return nil
}

func deleteServiceAccount(client *gcp.Client, name string) error {
	fmt.Printf("🗑️ Deleting Service Account: %s\n", name)

	if name == "" {
		return errors.NewValidationError(
			"Service account name is required for deletion",
			"Use --name to specify the service account name")
	}

	// Check if service account exists
	existing, err := client.GetServiceAccount(name)
	if err != nil {
		return err
	}

	if existing == nil {
		fmt.Printf("ℹ️ Service account '%s' does not exist, nothing to delete\n", name)
		return nil
	}

	fmt.Printf("   • Email: %s\n", existing.Email)
	fmt.Printf("   • Display Name: %s\n", existing.DisplayName)
	fmt.Printf("   • Unique ID: %s\n", existing.UniqueId)
	fmt.Println()

	// Get roles that will be revoked
	roles, err := client.GetServiceAccountProjectRoles(existing.Email)
	if err != nil {
		fmt.Printf("⚠️ Warning: Could not get service account roles: %v\n", err)
	} else if len(roles) > 0 {
		fmt.Printf("   Will revoke %d project roles:\n", len(roles))
		for _, role := range roles {
			fmt.Printf("     • %s\n", role)
		}
		fmt.Println()
	}

	fmt.Println("Deleting service account...")

	if err := client.DeleteServiceAccount(name); err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "SA_DELETE_FAILED",
			fmt.Sprintf("Failed to delete service account %s", name))
	}

	fmt.Printf("✅ Service Account Deleted Successfully!\n\n")
	fmt.Println("💡 The service account and all its project-level IAM bindings have been removed")

	return nil
}

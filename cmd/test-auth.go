package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/gcp"
	"github.com/Fordjour12/gcp-wif/internal/logging"
	"github.com/spf13/cobra"
)

var (
	testProjectID string
	checkPerms    bool
	refreshAuth   bool
)

// testAuthCmd represents the test-auth command
var testAuthCmd = &cobra.Command{
	Use:   "test-auth",
	Short: "Test GCP authentication and client connectivity",
	Long: `Test GCP authentication using gcloud CLI and verify client connectivity.

This command verifies that:
1. gcloud CLI is installed and properly authenticated
2. You have access to the specified GCP project
3. GCP API clients can be created and are functional
4. Required permissions are available for Workload Identity Federation

Examples:
  gcp-wif test-auth --project my-project-id
  gcp-wif test-auth --project my-project-id --check-permissions
  gcp-wif test-auth --project my-project-id --refresh`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTestAuth(cmd, args); err != nil {
			HandleError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(testAuthCmd)

	testAuthCmd.Flags().StringVarP(&testProjectID, "project", "p", "", "Google Cloud Project ID to test (required)")
	testAuthCmd.Flags().BoolVar(&checkPerms, "check-permissions", false, "Check required permissions for Workload Identity Federation")
	testAuthCmd.Flags().BoolVar(&refreshAuth, "refresh", false, "Refresh authentication tokens before testing")

	testAuthCmd.MarkFlagRequired("project")
}

func runTestAuth(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "test_auth")
	logger.Info("Starting GCP authentication test", "project_id", testProjectID)

	ctx := context.Background()

	fmt.Println("🔍 Testing GCP Authentication and Connectivity")
	fmt.Println("=============================================")
	fmt.Printf("📋 Project ID: %s\n\n", testProjectID)

	// Step 1: Create GCP client
	fmt.Println("Step 1: Creating GCP client...")
	client, err := gcp.NewClient(ctx, testProjectID)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "CLIENT_CREATION_FAILED",
			"Failed to create GCP client")
	}
	defer client.Close()

	fmt.Println("✅ GCP client created successfully")

	// Step 2: Display authentication information
	fmt.Println("\nStep 2: Authentication Information")
	authInfo := client.GetAuthInfo()
	fmt.Printf("   Account: %s\n", authInfo.Account)
	fmt.Printf("   Type: %s\n", authInfo.Type)
	fmt.Printf("   Status: %s\n", authInfo.Status)
	fmt.Printf("   Has ADC: %t\n", authInfo.HasADC)
	if authInfo.ProjectID != "" {
		fmt.Printf("   Default Project: %s\n", authInfo.ProjectID)
	}
	fmt.Printf("   Last Refresh: %s\n", authInfo.LastRefresh.Format(time.RFC3339))

	// Step 3: Display project information
	fmt.Println("\nStep 3: Project Information")
	projectInfo := client.GetProjectInfo()
	fmt.Printf("   Project ID: %s\n", projectInfo.ProjectID)
	fmt.Printf("   Project Number: %s\n", projectInfo.ProjectNumber)
	fmt.Printf("   Name: %s\n", projectInfo.Name)
	fmt.Printf("   State: %s\n", projectInfo.LifecycleState)
	fmt.Printf("   Created: %s\n", projectInfo.CreateTime)

	// Step 4: Test API connectivity
	fmt.Println("\nStep 4: Testing API connectivity...")
	if err := client.TestConnection(); err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "CONNECTION_TEST_FAILED",
			"GCP API connectivity test failed")
	}
	fmt.Println("✅ All GCP API connections verified")

	// Step 5: Refresh authentication if requested
	if refreshAuth {
		fmt.Println("\nStep 5: Refreshing authentication...")
		if err := client.RefreshAuth(); err != nil {
			logger.Warn("Authentication refresh failed", "error", err)
			fmt.Printf("⚠️  Warning: Authentication refresh failed: %v\n", err)
		} else {
			fmt.Println("✅ Authentication refreshed successfully")
		}
	}

	// Step 6: Check permissions if requested
	if checkPerms {
		fmt.Println("\nStep 6: Checking required permissions...")
		requiredPermissions := []string{
			"iam.serviceAccounts.create",
			"iam.serviceAccounts.get",
			"iam.serviceAccounts.setIamPolicy",
			"iam.workloadIdentityPools.create",
			"iam.workloadIdentityPools.get",
			"iam.workloadIdentityProviders.create",
			"iam.workloadIdentityProviders.get",
			"resourcemanager.projects.get",
			"resourcemanager.projects.setIamPolicy",
		}

		permissionResults, err := client.CheckPermissions(requiredPermissions)
		if err != nil {
			logger.Warn("Permission check failed", "error", err)
			fmt.Printf("⚠️  Warning: Permission check failed: %v\n", err)
		} else {
			fmt.Println("   Permission Check Results:")
			hasAllPermissions := true
			for _, perm := range requiredPermissions {
				if has, exists := permissionResults[perm]; exists && has {
					fmt.Printf("   ✅ %s\n", perm)
				} else {
					fmt.Printf("   ❌ %s\n", perm)
					hasAllPermissions = false
				}
			}

			if hasAllPermissions {
				fmt.Println("\n✅ All required permissions are available!")
			} else {
				fmt.Println("\n⚠️  Some required permissions are missing.")
				fmt.Println("   You may need additional IAM roles to set up Workload Identity Federation.")
				fmt.Println("   Required roles: Owner, Editor, or custom role with the missing permissions.")
			}
		}
	}

	fmt.Println("\n🎉 GCP authentication test completed successfully!")
	fmt.Println("\n💡 Next steps:")
	fmt.Println("   • Run 'gcp-wif setup' to configure Workload Identity Federation")
	fmt.Println("   • Use '--help' to see all available commands and options")

	logger.Info("GCP authentication test completed successfully")
	return nil
}

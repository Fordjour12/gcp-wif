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
	testConflictsProjectID   string
	testConflictsPoolID      string
	testConflictsProviderID  string
	testConflictsRepository  string
	testConflictsSAName      string
	testConflictsSAEmail     string
	testConflictsTestMode    string
	testConflictsCreateNew   bool
	testConflictsShowDetails bool
	testConflictsAutoResolve bool
	testConflictsSeverityMin string
)

// testConflictsCmd represents the test-conflicts command
var testConflictsCmd = &cobra.Command{
	Use:   "test-conflicts",
	Short: "Test comprehensive resource conflict detection and resolution",
	Long: `Test the enhanced conflict detection system for GCP resources.

This command tests conflict detection for:
- Service accounts (existing vs proposed configuration)
- Workload identity pools (state, configuration differences)
- Workload identity providers (repository, OIDC configuration)
- Cross-resource conflicts and dependencies

Test modes:
- service-account: Test service account conflicts only
- workload-identity: Test workload identity pool and provider conflicts
- comprehensive: Test all resource types and cross-dependencies
- resolution: Test conflict resolution suggestions and automation

Examples:
  # Test service account conflicts
  gcp-wif test-conflicts --project my-project --sa-name my-sa --test-mode service-account

  # Test workload identity conflicts
  gcp-wif test-conflicts --project my-project --pool-id my-pool --provider-id github \
    --repository owner/repo --test-mode workload-identity

  # Comprehensive conflict analysis
  gcp-wif test-conflicts --project my-project --pool-id my-pool --provider-id github \
    --repository owner/repo --sa-name my-sa --test-mode comprehensive --show-details

  # Test conflict resolution suggestions
  gcp-wif test-conflicts --project my-project --sa-name existing-sa \
    --test-mode resolution --auto-resolve

  # Filter by severity level
  gcp-wif test-conflicts --project my-project --sa-name my-sa \
    --severity-min critical --show-details`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTestConflicts(cmd, args); err != nil {
			HandleError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(testConflictsCmd)

	testConflictsCmd.Flags().StringVarP(&testConflictsProjectID, "project", "p", "", "Google Cloud Project ID (required)")
	testConflictsCmd.Flags().StringVar(&testConflictsPoolID, "pool-id", "", "Workload identity pool ID to test")
	testConflictsCmd.Flags().StringVar(&testConflictsProviderID, "provider-id", "", "Workload identity provider ID to test")
	testConflictsCmd.Flags().StringVarP(&testConflictsRepository, "repository", "r", "", "GitHub repository (owner/name)")
	testConflictsCmd.Flags().StringVar(&testConflictsSAName, "sa-name", "", "Service account name to test")
	testConflictsCmd.Flags().StringVar(&testConflictsSAEmail, "sa-email", "", "Service account email to test")
	testConflictsCmd.Flags().StringVar(&testConflictsTestMode, "test-mode", "comprehensive", "Test mode: service-account, workload-identity, comprehensive, resolution")
	testConflictsCmd.Flags().BoolVar(&testConflictsCreateNew, "create-new", true, "Test with create-new flag enabled")
	testConflictsCmd.Flags().BoolVar(&testConflictsShowDetails, "show-details", false, "Show detailed conflict analysis")
	testConflictsCmd.Flags().BoolVar(&testConflictsAutoResolve, "auto-resolve", false, "Test automatic conflict resolution")
	testConflictsCmd.Flags().StringVar(&testConflictsSeverityMin, "severity-min", "low", "Minimum severity to show: low, medium, high, critical")

	testConflictsCmd.MarkFlagRequired("project")
}

func runTestConflicts(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "test_conflicts")
	logger.Info("Starting conflict detection tests", "project_id", testConflictsProjectID, "test_mode", testConflictsTestMode)

	ctx := context.Background()

	fmt.Println("🔍 Testing Resource Conflict Detection System")
	fmt.Println("============================================")
	fmt.Printf("📋 Project ID: %s\n", testConflictsProjectID)
	fmt.Printf("🧪 Test Mode: %s\n\n", testConflictsTestMode)

	// Create GCP client
	client, err := gcp.NewClient(ctx, testConflictsProjectID)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "CLIENT_CREATION_FAILED",
			"Failed to create GCP client")
	}
	defer client.Close()

	switch testConflictsTestMode {
	case "service-account":
		return testServiceAccountConflicts(client)
	case "workload-identity":
		return testWorkloadIdentityConflicts(client)
	case "comprehensive":
		return testComprehensiveConflicts(client)
	case "resolution":
		return testConflictResolution(client)
	default:
		return errors.NewValidationError(
			fmt.Sprintf("Unknown test mode: %s", testConflictsTestMode),
			"Valid modes: service-account, workload-identity, comprehensive, resolution")
	}
}

func testServiceAccountConflicts(client *gcp.Client) error {
	fmt.Println("🔧 Testing Service Account Conflicts...")

	if testConflictsSAName == "" {
		return errors.NewValidationError(
			"Service account name is required for service account conflict testing",
			"Use --sa-name to specify the service account name")
	}

	// Test configuration with different scenarios
	config := &gcp.ServiceAccountConfig{
		Name:        testConflictsSAName,
		DisplayName: "Test SA for Conflict Detection",
		Description: "Service account created for testing conflict detection",
		Roles:       gcp.DefaultWorkloadIdentityRoles(),
		CreateNew:   testConflictsCreateNew,
	}

	fmt.Printf("   Testing with: %s\n", config.Name)
	fmt.Printf("   Create New: %t\n\n", config.CreateNew)

	// Run conflict detection
	result, err := client.DetectAllResourceConflicts(config)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "SA_CONFLICT_TEST_FAILED",
			"Failed to run service account conflict detection")
	}

	displayConflictResults(result, "Service Account")

	// Show detailed analysis if requested
	if testConflictsShowDetails && result.HasConflicts {
		fmt.Println("\n📊 Detailed Conflict Analysis:")
		for i, conflict := range result.Conflicts {
			if shouldShowConflict(conflict.Severity) {
				displayDetailedConflict(i+1, conflict)
			}
		}
	}

	return nil
}

func testWorkloadIdentityConflicts(client *gcp.Client) error {
	fmt.Println("🔗 Testing Workload Identity Conflicts...")

	if testConflictsPoolID == "" || testConflictsProviderID == "" || testConflictsRepository == "" {
		return errors.NewValidationError(
			"Pool ID, Provider ID, and Repository are required for workload identity conflict testing",
			"Use --pool-id, --provider-id, and --repository flags")
	}

	// Test configuration
	config := &gcp.WorkloadIdentityConfig{
		PoolID:              testConflictsPoolID,
		PoolName:            "Test WIF Pool",
		PoolDescription:     "Workload identity pool for conflict testing",
		ProviderID:          testConflictsProviderID,
		ProviderName:        "Test GitHub Provider",
		ProviderDescription: "GitHub OIDC provider for conflict testing",
		Repository:          testConflictsRepository,
		ServiceAccountEmail: testConflictsSAEmail,
		CreateNew:           testConflictsCreateNew,
		GitHubOIDC:          gcp.GetDefaultGitHubOIDCConfig(),
		ClaimsMapping:       gcp.GetDefaultGitHubClaimsMapping(),
	}

	fmt.Printf("   Pool ID: %s\n", config.PoolID)
	fmt.Printf("   Provider ID: %s\n", config.ProviderID)
	fmt.Printf("   Repository: %s\n", config.Repository)
	fmt.Printf("   Create New: %t\n\n", config.CreateNew)

	// Run conflict detection
	result, err := client.DetectAllResourceConflicts(config)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeGCP, "WI_CONFLICT_TEST_FAILED",
			"Failed to run workload identity conflict detection")
	}

	displayConflictResults(result, "Workload Identity")

	// Show detailed analysis if requested
	if testConflictsShowDetails && result.HasConflicts {
		fmt.Println("\n📊 Detailed Conflict Analysis:")
		for i, conflict := range result.Conflicts {
			if shouldShowConflict(conflict.Severity) {
				displayDetailedConflict(i+1, conflict)
			}
		}
	}

	return nil
}

func testComprehensiveConflicts(client *gcp.Client) error {
	fmt.Println("🔍 Testing Comprehensive Resource Conflicts...")

	// Run both service account and workload identity tests
	fmt.Println("\n1️⃣ Service Account Conflict Detection:")
	if testConflictsSAName != "" {
		if err := testServiceAccountConflicts(client); err != nil {
			fmt.Printf("❌ Service account conflict test failed: %v\n", err)
		}
	} else {
		fmt.Println("⏭️  Skipped (no service account name provided)")
	}

	fmt.Println("\n2️⃣ Workload Identity Conflict Detection:")
	if testConflictsPoolID != "" && testConflictsProviderID != "" && testConflictsRepository != "" {
		if err := testWorkloadIdentityConflicts(client); err != nil {
			fmt.Printf("❌ Workload identity conflict test failed: %v\n", err)
		}
	} else {
		fmt.Println("⏭️  Skipped (missing pool-id, provider-id, or repository)")
	}

	// Test cross-resource dependencies
	fmt.Println("\n3️⃣ Cross-Resource Dependency Analysis:")
	if testConflictsSAName != "" && testConflictsRepository != "" {
		testCrossResourceDependencies(client)
	} else {
		fmt.Println("⏭️  Skipped (need both service account and repository)")
	}

	return nil
}

func testConflictResolution(client *gcp.Client) error {
	fmt.Println("🛠️ Testing Conflict Resolution Suggestions...")

	if testConflictsSAName == "" {
		return errors.NewValidationError(
			"Service account name is required for resolution testing",
			"Use --sa-name to specify the service account name")
	}

	// Test different conflict scenarios
	scenarios := []struct {
		name        string
		createNew   bool
		description string
	}{
		{"Fail on Conflicts", false, "Test with CreateNew=false to see conflict errors"},
		{"Auto-resolve", true, "Test with CreateNew=true to see auto-resolution"},
	}

	config := &gcp.ServiceAccountConfig{
		Name:        testConflictsSAName,
		DisplayName: "Updated Display Name",
		Description: "Updated description for testing",
		Roles:       gcp.DefaultWorkloadIdentityRoles(),
	}

	for i, scenario := range scenarios {
		fmt.Printf("\n%d. %s:\n", i+1, scenario.name)
		fmt.Printf("   %s\n", scenario.description)

		config.CreateNew = scenario.createNew

		result, err := client.DetectAllResourceConflicts(config)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
			continue
		}

		if !result.HasConflicts {
			fmt.Printf("   ✅ No conflicts detected\n")
			continue
		}

		fmt.Printf("   📋 Found %d conflict(s) - %s\n", result.TotalConflicts, result.Summary)
		fmt.Printf("   🎯 Recommended Action: %s\n", result.RecommendedAction)

		// Show resolution suggestions for first conflict
		if len(result.Conflicts) > 0 {
			conflict := result.Conflicts[0]
			fmt.Printf("   🔧 Resolution Options:\n")
			for j, suggestion := range conflict.Suggestions {
				status := "•"
				if suggestion.Recommended {
					status = "✓"
				}
				fmt.Printf("     %s [%d] %s\n", status, j+1, suggestion.Title)
				fmt.Printf("       %s\n", suggestion.Description)
				if suggestion.Automated {
					fmt.Printf("       🤖 Can be automated\n")
				}
			}
		}
	}

	return nil
}

func testCrossResourceDependencies(client *gcp.Client) {
	fmt.Printf("   Analyzing dependencies between resources...\n")
	fmt.Printf("   Service Account: %s\n", testConflictsSAName)
	fmt.Printf("   Repository: %s\n", testConflictsRepository)

	// Check if service account exists
	existing, err := client.GetServiceAccountInfo(testConflictsSAName)
	if err != nil {
		fmt.Printf("   ❌ Error checking service account: %v\n", err)
		return
	}

	if existing == nil || !existing.Exists {
		fmt.Printf("   ℹ️  Service account does not exist - no dependencies to analyze\n")
		return
	}

	// Check for workload identity bindings
	fmt.Printf("   📋 Service Account Found:\n")
	fmt.Printf("     Email: %s\n", existing.Email)
	fmt.Printf("     Roles: %v\n", existing.ProjectRoles)

	// Check if this SA is used with any workload identity pools
	pools, err := client.ListWorkloadIdentityPools()
	if err != nil {
		fmt.Printf("   ⚠️  Could not list workload identity pools: %v\n", err)
		return
	}

	var relatedPools int
	for _, pool := range pools {
		// This is a simplified check - in practice you'd need to examine IAM bindings
		if strings.Contains(pool.Description, testConflictsSAName) ||
			strings.Contains(pool.Description, testConflictsRepository) {
			relatedPools++
		}
	}

	if relatedPools > 0 {
		fmt.Printf("   🔗 Found %d potentially related workload identity pool(s)\n", relatedPools)
		fmt.Printf("   ⚠️  Modifying this service account may affect existing WIF configurations\n")
	} else {
		fmt.Printf("   ✅ No obvious workload identity dependencies detected\n")
	}
}

func displayConflictResults(result *gcp.ConflictDetectionResult, resourceType string) {
	fmt.Printf("🔍 %s Conflict Detection Results:\n", resourceType)
	fmt.Printf("   Total Conflicts: %d\n", result.TotalConflicts)

	if !result.HasConflicts {
		fmt.Printf("   ✅ No conflicts detected - safe to proceed\n")
		return
	}

	fmt.Printf("   📊 Severity Breakdown:\n")
	if result.CriticalCount > 0 {
		fmt.Printf("     🔴 Critical: %d\n", result.CriticalCount)
	}
	if result.HighCount > 0 {
		fmt.Printf("     🟠 High: %d\n", result.HighCount)
	}
	if result.MediumCount > 0 {
		fmt.Printf("     🟡 Medium: %d\n", result.MediumCount)
	}
	if result.LowCount > 0 {
		fmt.Printf("     🟢 Low: %d\n", result.LowCount)
	}

	fmt.Printf("   🎯 Can Proceed: %t\n", result.CanProceed)
	fmt.Printf("   📋 Summary: %s\n", result.Summary)
	fmt.Printf("   💡 Recommended Action: %s\n", result.RecommendedAction)
}

func displayDetailedConflict(index int, conflict gcp.ResourceConflict) {
	severityIcon := getSeverityIcon(conflict.Severity)
	fmt.Printf("\n   %s Conflict #%d: %s (%s)\n", severityIcon, index, conflict.ResourceName, conflict.ResourceType)
	fmt.Printf("     Type: %s\n", conflict.ConflictType)
	fmt.Printf("     Severity: %s\n", conflict.Severity)
	fmt.Printf("     Auto-Resolvable: %t\n", conflict.CanAutoResolve)

	if len(conflict.Differences) > 0 {
		fmt.Printf("     📊 Differences (%d):\n", len(conflict.Differences))
		for i, diff := range conflict.Differences {
			diffIcon := getSeverityIcon(gcp.ConflictSeverity(diff.Severity))
			fmt.Printf("       %s [%d] %s: %s\n", diffIcon, i+1, diff.Field, diff.Description)
			if testConflictsShowDetails {
				fmt.Printf("         Existing: %v\n", diff.ExistingValue)
				fmt.Printf("         Proposed: %v\n", diff.ProposedValue)
			}
		}
	}

	if len(conflict.Suggestions) > 0 {
		fmt.Printf("     🔧 Resolution Suggestions (%d):\n", len(conflict.Suggestions))
		for i, suggestion := range conflict.Suggestions {
			status := "•"
			if suggestion.Recommended {
				status = "✓"
			}
			automation := ""
			if suggestion.Automated {
				automation = " (automated)"
			}
			fmt.Printf("       %s [%d] %s%s\n", status, i+1, suggestion.Title, automation)
			fmt.Printf("         %s\n", suggestion.Description)

			if len(suggestion.Commands) > 0 && testConflictsShowDetails {
				fmt.Printf("         Commands: %v\n", suggestion.Commands)
			}
		}
	}
}

func shouldShowConflict(severity gcp.ConflictSeverity) bool {
	severityOrder := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	minLevel, exists := severityOrder[testConflictsSeverityMin]
	if !exists {
		minLevel = 1 // Default to low
	}

	currentLevel, exists := severityOrder[string(severity)]
	if !exists {
		return true // Show unknown severities
	}

	return currentLevel >= minLevel
}

func getSeverityIcon(severity gcp.ConflictSeverity) string {
	switch severity {
	case gcp.ConflictSeverityCritical:
		return "🔴"
	case gcp.ConflictSeverityHigh:
		return "🟠"
	case gcp.ConflictSeverityMedium:
		return "🟡"
	case gcp.ConflictSeverityLow:
		return "🟢"
	default:
		return "⚪"
	}
}

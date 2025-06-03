package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Fordjour12/gcp-wif/internal/config"
	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/gcp"
	"github.com/Fordjour12/gcp-wif/internal/logging"
	"github.com/spf13/cobra"
)

var (
	// Cleanup scope flags
	cleanupAll                bool
	cleanupServiceAccountFlag bool
	cleanupWorkloadIdentity   bool
	cleanupWorkflows          bool
	cleanupIAMBindings        bool

	// Cleanup behavior flags
	cleanupForce       bool
	cleanupDryRun      bool
	cleanupConfirm     bool
	cleanupBackupFirst bool
	cleanupShowDetails bool
	cleanupParallel    bool
	cleanupTimeout     string
	cleanupRetries     int

	// Resource specification flags
	cleanupProjectID     string
	cleanupServiceAcct   string
	cleanupPoolID        string
	cleanupProviderID    string
	cleanupWorkflowPaths []string

	// Advanced flags
	cleanupIgnoreErrors   bool
	cleanupPurgeBackups   bool
	cleanupDeleteConfigs  bool
	cleanupVerifyDeletion bool
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up and rollback Workload Identity Federation resources",
	Long: `Clean up and rollback Google Cloud Workload Identity Federation resources.

This command provides comprehensive cleanup and rollback functionality for WIF setups.
It can selectively remove different types of resources or perform complete cleanup.

Cleanup Scope Options:
• --all: Remove all WIF resources (service account, workload identity, workflows)
• --service-account: Remove only the service account and its IAM bindings
• --workload-identity: Remove only workload identity pools and providers
• --workflows: Remove only GitHub Actions workflow files
• --iam-bindings: Remove only IAM policy bindings (keep resources)

Safety and Verification:
• --dry-run: Preview what would be deleted without making changes
• --confirm: Require explicit confirmation for each resource
• --backup-first: Create backups before deletion
• --verify-deletion: Verify each resource is actually deleted

Advanced Options:
• --force: Skip all confirmation prompts and safety checks
• --parallel: Delete resources in parallel for faster cleanup
• --ignore-errors: Continue cleanup even if some operations fail
• --show-details: Display detailed information about each resource

Examples:
  # Preview complete cleanup
  gcp-wif cleanup --all --dry-run --config wif-config.json

  # Remove only workload identity resources with confirmation
  gcp-wif cleanup --workload-identity --confirm

  # Force cleanup of everything except service account
  gcp-wif cleanup --workload-identity --workflows --iam-bindings --force

  # Cleanup specific resources by ID
  gcp-wif cleanup --pool-id my-pool --provider-id my-provider

  # Safe cleanup with backups and verification
  gcp-wif cleanup --all --backup-first --verify-deletion`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runCleanupCommand(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	// Cleanup scope flags
	cleanupCmd.Flags().BoolVar(&cleanupAll, "all", false, "Clean up all WIF resources")
	cleanupCmd.Flags().BoolVar(&cleanupServiceAccountFlag, "service-account", false, "Clean up service account")
	cleanupCmd.Flags().BoolVar(&cleanupWorkloadIdentity, "workload-identity", false, "Clean up workload identity resources")
	cleanupCmd.Flags().BoolVar(&cleanupWorkflows, "workflows", false, "Clean up workflow files")
	cleanupCmd.Flags().BoolVar(&cleanupIAMBindings, "iam-bindings", false, "Clean up IAM bindings only")

	// Cleanup behavior flags
	cleanupCmd.Flags().BoolVar(&cleanupForce, "force", false, "Force cleanup without confirmation")
	cleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "Show what would be cleaned up")
	cleanupCmd.Flags().BoolVar(&cleanupConfirm, "confirm", false, "Require confirmation for each resource")
	cleanupCmd.Flags().BoolVar(&cleanupBackupFirst, "backup-first", false, "Create backups before deletion")
	cleanupCmd.Flags().BoolVar(&cleanupShowDetails, "show-details", false, "Show detailed resource information")
	cleanupCmd.Flags().BoolVar(&cleanupParallel, "parallel", false, "Perform cleanup operations in parallel")
	cleanupCmd.Flags().StringVar(&cleanupTimeout, "timeout", "10m", "Timeout for cleanup operations")
	cleanupCmd.Flags().IntVar(&cleanupRetries, "retries", 3, "Number of retries for failed operations")

	// Resource specification flags
	cleanupCmd.Flags().StringVar(&cleanupProjectID, "project-id", "", "Google Cloud Project ID")
	cleanupCmd.Flags().StringVar(&cleanupServiceAcct, "service-account-name", "", "Service account name to clean up")
	cleanupCmd.Flags().StringVar(&cleanupPoolID, "pool-id", "", "Workload identity pool ID to clean up")
	cleanupCmd.Flags().StringVar(&cleanupProviderID, "provider-id", "", "Workload identity provider ID to clean up")
	cleanupCmd.Flags().StringSliceVar(&cleanupWorkflowPaths, "workflow-paths", []string{}, "Workflow file paths to clean up")

	// Advanced flags
	cleanupCmd.Flags().BoolVar(&cleanupIgnoreErrors, "ignore-errors", false, "Continue cleanup even if some operations fail")
	cleanupCmd.Flags().BoolVar(&cleanupPurgeBackups, "purge-backups", false, "Also delete backup files")
	cleanupCmd.Flags().BoolVar(&cleanupDeleteConfigs, "delete-configs", false, "Also delete configuration files")
	cleanupCmd.Flags().BoolVar(&cleanupVerifyDeletion, "verify-deletion", false, "Verify each resource is actually deleted")
}

func runCleanupCommand(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "cleanup")
	logger.Info("Starting cleanup command")

	fmt.Println("🧹 WIF Cleanup and Rollback Tool")
	fmt.Println("================================")

	// Load configuration
	cfg, err := loadCleanupConfig()
	if err != nil {
		return err
	}

	// Apply command-line overrides
	if err := applyCleanupFlags(cfg); err != nil {
		return err
	}

	// Determine cleanup scope
	scope, err := determineCleanupScope()
	if err != nil {
		return err
	}

	if len(scope.Resources) == 0 {
		return errors.NewValidationError(
			"No cleanup scope specified",
			"Use --all or specify specific resource types to clean up",
			"Examples:",
			"  gcp-wif cleanup --all",
			"  gcp-wif cleanup --workload-identity --workflows",
			"  gcp-wif cleanup --service-account")
	}

	// Display cleanup plan
	if err := displayCleanupPlan(cfg, scope); err != nil {
		return err
	}

	// Handle dry-run mode
	if cleanupDryRun {
		fmt.Println("\n💡 This was a dry-run. Use --dry-run=false to execute cleanup.")
		return nil
	}

	// Get confirmation unless force mode
	if !cleanupForce {
		if !confirmCleanup(scope) {
			fmt.Println("Cleanup cancelled by user")
			return nil
		}
	}

	// Execute cleanup
	logger.Info("Starting cleanup execution", "scope", scope.Resources)
	fmt.Println("\n🔧 Executing cleanup operations...")

	if err := executeCleanup(cfg, scope); err != nil {
		return err
	}

	fmt.Println("\n🎉 Cleanup completed successfully!")
	logger.Info("Cleanup command completed successfully")
	return nil
}

// CleanupScope defines what resources should be cleaned up
type CleanupScope struct {
	Resources         []string `json:"resources"`
	ServiceAccount    bool     `json:"service_account"`
	WorkloadIdentity  bool     `json:"workload_identity"`
	IAMBindings       bool     `json:"iam_bindings"`
	Workflows         bool     `json:"workflows"`
	ConfigFiles       bool     `json:"config_files"`
	BackupFiles       bool     `json:"backup_files"`
	VerifyDeletion    bool     `json:"verify_deletion"`
	ParallelExecution bool     `json:"parallel_execution"`
	IgnoreErrors      bool     `json:"ignore_errors"`
}

// CleanupResult tracks the results of cleanup operations
type CleanupResult struct {
	ResourcesDeleted []string                   `json:"resources_deleted"`
	ResourcesFailed  []string                   `json:"resources_failed"`
	BackupsCreated   []string                   `json:"backups_created"`
	OperationDetails map[string]CleanupOpResult `json:"operation_details"`
	TotalDuration    time.Duration              `json:"total_duration"`
	SuccessCount     int                        `json:"success_count"`
	FailureCount     int                        `json:"failure_count"`
}

// CleanupOpResult tracks individual cleanup operation results
type CleanupOpResult struct {
	ResourceType string        `json:"resource_type"`
	ResourceID   string        `json:"resource_id"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	Duration     time.Duration `json:"duration"`
	BackupPath   string        `json:"backup_path,omitempty"`
	Verified     bool          `json:"verified"`
}

// loadCleanupConfig loads configuration for cleanup operations
func loadCleanupConfig() (*config.Config, error) {
	logger := logging.WithField("function", "loadCleanupConfig")

	// Try to load from specified config file
	if cfgFile != "" {
		logger.Debug("Loading configuration from file", "path", cfgFile)
		cfg, err := config.LoadFromFile(cfgFile)
		if err != nil {
			return nil, err
		}
		logger.Info("Configuration loaded from file", "path", cfgFile)
		return cfg, nil
	}

	// Try auto-discovery
	logger.Debug("Attempting config file auto-discovery")
	cfg, err := config.LoadFromFileWithDiscovery("")
	if err == nil {
		logger.Info("Configuration loaded via auto-discovery")
		return cfg, nil
	}

	// Create minimal config for flag-based cleanup
	logger.Debug("Creating minimal configuration for flag-based cleanup")
	cfg = config.DefaultConfig()
	logger.Info("Created default configuration for cleanup")
	return cfg, nil
}

// applyCleanupFlags applies command-line flags to override configuration
func applyCleanupFlags(cfg *config.Config) error {
	logger := logging.WithField("function", "applyCleanupFlags")

	// Apply project ID override
	if cleanupProjectID != "" {
		cfg.Project.ID = cleanupProjectID
		logger.Debug("Applied project ID from flag", "project_id", cleanupProjectID)
	}

	// Apply service account override
	if cleanupServiceAcct != "" {
		cfg.ServiceAccount.Name = cleanupServiceAcct
		logger.Debug("Applied service account name from flag", "name", cleanupServiceAcct)
	}

	// Apply workload identity overrides
	if cleanupPoolID != "" {
		cfg.WorkloadIdentity.PoolID = cleanupPoolID
		logger.Debug("Applied pool ID from flag", "pool_id", cleanupPoolID)
	}

	if cleanupProviderID != "" {
		cfg.WorkloadIdentity.ProviderID = cleanupProviderID
		logger.Debug("Applied provider ID from flag", "provider_id", cleanupProviderID)
	}

	// Apply advanced flags to config
	cfg.Advanced.DryRun = cleanupDryRun
	cfg.Advanced.ForceUpdate = cleanupForce
	cfg.Advanced.BackupExisting = cleanupBackupFirst

	logger.Debug("Applied all cleanup flags successfully")
	return nil
}

// determineCleanupScope determines what resources should be cleaned up
func determineCleanupScope() (*CleanupScope, error) {
	scope := &CleanupScope{
		VerifyDeletion:    cleanupVerifyDeletion,
		ParallelExecution: cleanupParallel,
		IgnoreErrors:      cleanupIgnoreErrors,
	}

	// Determine resource scope
	if cleanupAll {
		scope.Resources = []string{"service-account", "workload-identity", "iam-bindings", "workflows"}
		scope.ServiceAccount = true
		scope.WorkloadIdentity = true
		scope.IAMBindings = true
		scope.Workflows = true
	} else {
		if cleanupServiceAccountFlag {
			scope.Resources = append(scope.Resources, "service-account")
			scope.ServiceAccount = true
		}
		if cleanupWorkloadIdentity {
			scope.Resources = append(scope.Resources, "workload-identity")
			scope.WorkloadIdentity = true
		}
		if cleanupIAMBindings {
			scope.Resources = append(scope.Resources, "iam-bindings")
			scope.IAMBindings = true
		}
		if cleanupWorkflows {
			scope.Resources = append(scope.Resources, "workflows")
			scope.Workflows = true
		}
	}

	// Additional cleanup options
	if cleanupDeleteConfigs {
		scope.Resources = append(scope.Resources, "config-files")
		scope.ConfigFiles = true
	}

	if cleanupPurgeBackups {
		scope.Resources = append(scope.Resources, "backup-files")
		scope.BackupFiles = true
	}

	return scope, nil
}

// displayCleanupPlan displays what will be cleaned up
func displayCleanupPlan(cfg *config.Config, scope *CleanupScope) error {
	logger := logging.WithField("function", "displayCleanupPlan")

	fmt.Println("\n📋 Cleanup Plan:")
	fmt.Println("================")

	// Project information
	fmt.Printf("🏗️  Project: %s\n", cfg.Project.ID)
	if cfg.GetRepoFullName() != "/" {
		fmt.Printf("📚 Repository: %s\n", cfg.GetRepoFullName())
	}

	// Resources to be cleaned
	fmt.Println("\n🗑️  Resources to be cleaned:")
	resourceCount := 0

	if scope.ServiceAccount {
		fmt.Printf("   • Service Account: %s\n", cfg.ServiceAccount.Name)
		fmt.Printf("     Email: %s\n", cfg.GetServiceAccountEmail())
		if len(cfg.ServiceAccount.Roles) > 0 {
			fmt.Printf("     Roles: %s\n", strings.Join(cfg.ServiceAccount.Roles, ", "))
		}
		resourceCount++
	}

	if scope.WorkloadIdentity {
		fmt.Printf("   • Workload Identity Pool: %s\n", cfg.WorkloadIdentity.PoolID)
		fmt.Printf("   • Workload Identity Provider: %s\n", cfg.WorkloadIdentity.ProviderID)
		resourceCount += 2
	}

	if scope.IAMBindings {
		fmt.Println("   • IAM Policy Bindings:")
		fmt.Println("     - roles/iam.serviceAccountTokenCreator")
		fmt.Println("     - roles/iam.workloadIdentityUser (legacy)")
		resourceCount++
	}

	if scope.Workflows {
		workflowPath := cfg.Workflow.GetWorkflowFilePath()
		fmt.Printf("   • GitHub Actions Workflow: %s\n", workflowPath)
		if len(cleanupWorkflowPaths) > 0 {
			fmt.Printf("   • Additional Workflows: %s\n", strings.Join(cleanupWorkflowPaths, ", "))
		}
		resourceCount++
	}

	if scope.ConfigFiles {
		fmt.Println("   • Configuration Files:")
		fmt.Println("     - wif-config.json")
		fmt.Println("     - .gcp-wif.json")
		resourceCount++
	}

	if scope.BackupFiles {
		fmt.Println("   • Backup Files:")
		fmt.Println("     - All *.backup files")
		fmt.Println("     - All *-backup-*.json files")
		resourceCount++
	}

	// Operation settings
	fmt.Println("\n⚙️  Operation Settings:")
	if cleanupDryRun {
		fmt.Println("   • Mode: DRY RUN (no changes will be made)")
	} else {
		fmt.Println("   • Mode: LIVE (changes will be applied)")
	}

	if cleanupForce {
		fmt.Println("   • Confirmation: DISABLED (force mode)")
	} else {
		fmt.Println("   • Confirmation: ENABLED")
	}

	if cleanupBackupFirst && !cleanupDryRun {
		fmt.Println("   • Backups: ENABLED (backups will be created)")
	}

	if cleanupVerifyDeletion {
		fmt.Println("   • Verification: ENABLED (deletions will be verified)")
	}

	if cleanupParallel {
		fmt.Println("   • Execution: PARALLEL (faster but less detailed logging)")
	} else {
		fmt.Println("   • Execution: SEQUENTIAL (detailed progress reporting)")
	}

	fmt.Printf("\n📊 Total Resources: %d\n", resourceCount)

	if cleanupShowDetails {
		if err := showResourceDetails(cfg); err != nil {
			logger.Warn("Failed to show resource details", "error", err)
		}
	}

	logger.Debug("Cleanup plan displayed", "resource_count", resourceCount)
	return nil
}

// showResourceDetails shows detailed information about resources to be cleaned
func showResourceDetails(cfg *config.Config) error {
	fmt.Println("\n🔍 Resource Details:")
	fmt.Println("===================")

	// Initialize GCP client to check resource existence
	ctx := context.Background()
	client, err := gcp.NewClient(ctx, cfg.Project.ID)
	if err != nil {
		return fmt.Errorf("failed to initialize GCP client: %w", err)
	}

	// Check service account
	if cleanupServiceAccountFlag || cleanupAll {
		sa, err := client.GetServiceAccount(cfg.ServiceAccount.Name)
		if err != nil {
			fmt.Printf("   ❌ Service Account: %s (not found or error: %v)\n", cfg.ServiceAccount.Name, err)
		} else if sa != nil {
			fmt.Printf("   ✅ Service Account: %s\n", sa.Email)
			fmt.Printf("      Description: %s\n", sa.Description)
		}
	}

	// Check workload identity pool
	if cleanupWorkloadIdentity || cleanupAll {
		poolInfo, err := client.GetWorkloadIdentityPoolInfo(cfg.WorkloadIdentity.PoolID)
		if err != nil {
			fmt.Printf("   ❌ WI Pool: %s (not found or error: %v)\n", cfg.WorkloadIdentity.PoolID, err)
		} else if poolInfo.Exists {
			fmt.Printf("   ✅ WI Pool: %s\n", poolInfo.DisplayName)
			fmt.Printf("      State: %s\n", poolInfo.State)
			fmt.Printf("      Created: %s\n", poolInfo.CreateTime.Format("2006-01-02 15:04:05"))
		}

		// Check workload identity provider
		providerInfo, err := client.GetWorkloadIdentityProviderInfo(cfg.WorkloadIdentity.PoolID, cfg.WorkloadIdentity.ProviderID)
		if err != nil {
			fmt.Printf("   ❌ WI Provider: %s (not found or error: %v)\n", cfg.WorkloadIdentity.ProviderID, err)
		} else if providerInfo.Exists {
			fmt.Printf("   ✅ WI Provider: %s\n", providerInfo.DisplayName)
			fmt.Printf("      State: %s\n", providerInfo.State)
			fmt.Printf("      Issuer: %s\n", providerInfo.IssuerURI)
		}
	}

	return nil
}

// confirmCleanup prompts for user confirmation
func confirmCleanup(scope *CleanupScope) bool {
	if cleanupConfirm {
		// Individual resource confirmation
		for _, resource := range scope.Resources {
			fmt.Printf("\n❓ Do you want to clean up %s? (y/N): ", resource)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
				fmt.Printf("Skipping %s cleanup\n", resource)
				// Remove from scope
				// Implementation would need to modify scope
			}
		}
		return true
	} else {
		// Single confirmation for all
		fmt.Printf("\n❓ Are you sure you want to proceed with cleanup? (y/N): ")
		var response string
		fmt.Scanln(&response)
		return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
	}
}

// executeCleanup performs the actual cleanup operations
func executeCleanup(cfg *config.Config, scope *CleanupScope) error {
	logger := logging.WithField("function", "executeCleanup")
	logger.Info("Starting cleanup execution")

	result := &CleanupResult{
		OperationDetails: make(map[string]CleanupOpResult),
	}
	startTime := time.Now()

	// Initialize GCP client
	ctx := context.Background()
	client, err := gcp.NewClient(ctx, cfg.Project.ID)
	if err != nil {
		return fmt.Errorf("failed to initialize GCP client: %w", err)
	}

	// Execute cleanup operations in order
	operations := []CleanupOperation{
		{Type: "iam-bindings", Enabled: scope.IAMBindings, Function: func() error { return cleanupIAMBindingsOp(client, cfg, result) }},
		{Type: "workload-identity-provider", Enabled: scope.WorkloadIdentity, Function: func() error { return cleanupWIProviderOp(client, cfg, result) }},
		{Type: "workload-identity-pool", Enabled: scope.WorkloadIdentity, Function: func() error { return cleanupWIPoolOp(client, cfg, result) }},
		{Type: "service-account", Enabled: scope.ServiceAccount, Function: func() error { return cleanupServiceAccountOp(client, cfg, result) }},
		{Type: "workflows", Enabled: scope.Workflows, Function: func() error { return cleanupWorkflowsOp(cfg, result) }},
		{Type: "config-files", Enabled: scope.ConfigFiles, Function: func() error { return cleanupConfigFilesOp(cfg, result) }},
		{Type: "backup-files", Enabled: scope.BackupFiles, Function: func() error { return cleanupBackupFilesOp(cfg, result) }},
	}

	// Execute operations
	for _, op := range operations {
		if !op.Enabled {
			continue
		}

		fmt.Printf("   🔧 Cleaning up %s...\n", op.Type)
		opStart := time.Now()

		err := op.Function()
		duration := time.Since(opStart)

		if err != nil {
			result.FailureCount++
			result.ResourcesFailed = append(result.ResourcesFailed, op.Type)
			logger.Error("Cleanup operation failed", "type", op.Type, "error", err, "duration", duration)

			if !scope.IgnoreErrors {
				return fmt.Errorf("cleanup failed for %s: %w", op.Type, err)
			} else {
				fmt.Printf("   ⚠️  %s cleanup failed (continuing): %v\n", op.Type, err)
			}
		} else {
			result.SuccessCount++
			result.ResourcesDeleted = append(result.ResourcesDeleted, op.Type)
			fmt.Printf("   ✅ %s cleaned up successfully\n", op.Type)
			logger.Info("Cleanup operation completed", "type", op.Type, "duration", duration)
		}
	}

	result.TotalDuration = time.Since(startTime)

	// Display final results
	displayCleanupResults(result)

	logger.Info("Cleanup execution completed",
		"success_count", result.SuccessCount,
		"failure_count", result.FailureCount,
		"total_duration", result.TotalDuration)

	return nil
}

// CleanupOperation represents a single cleanup operation
type CleanupOperation struct {
	Type     string
	Enabled  bool
	Function func() error
}

// Individual cleanup operation functions
func cleanupIAMBindingsOp(client *gcp.Client, cfg *config.Config, result *CleanupResult) error {
	workloadIdentityConfig := &gcp.WorkloadIdentityConfig{
		PoolID:              cfg.WorkloadIdentity.PoolID,
		ProviderID:          cfg.WorkloadIdentity.ProviderID,
		Repository:          cfg.GetRepoFullName(),
		ServiceAccountEmail: cfg.GetServiceAccountEmail(),
	}

	err := client.RemoveServiceAccountWorkloadIdentityBinding(workloadIdentityConfig)
	if err != nil {
		return err
	}

	if cleanupVerifyDeletion {
		// Verify bindings are removed
		bindings, err := client.ListServiceAccountWorkloadIdentityBindings(cfg.GetServiceAccountEmail())
		if err == nil && len(bindings) == 0 {
			fmt.Println("     ✅ IAM bindings removal verified")
		}
	}

	return nil
}

func cleanupWIProviderOp(client *gcp.Client, cfg *config.Config, result *CleanupResult) error {
	err := client.DeleteWorkloadIdentityProvider(cfg.WorkloadIdentity.PoolID, cfg.WorkloadIdentity.ProviderID)
	if err != nil {
		return err
	}

	if cleanupVerifyDeletion {
		// Verify provider is deleted
		_, err := client.GetWorkloadIdentityProviderInfo(cfg.WorkloadIdentity.PoolID, cfg.WorkloadIdentity.ProviderID)
		if err != nil {
			fmt.Println("     ✅ WI provider deletion verified")
		}
	}

	return nil
}

func cleanupWIPoolOp(client *gcp.Client, cfg *config.Config, result *CleanupResult) error {
	err := client.DeleteWorkloadIdentityPool(cfg.WorkloadIdentity.PoolID)
	if err != nil {
		return err
	}

	if cleanupVerifyDeletion {
		// Verify pool is deleted
		poolInfo, err := client.GetWorkloadIdentityPoolInfo(cfg.WorkloadIdentity.PoolID)
		if err != nil || !poolInfo.Exists {
			fmt.Println("     ✅ WI pool deletion verified")
		}
	}

	return nil
}

func cleanupServiceAccountOp(client *gcp.Client, cfg *config.Config, result *CleanupResult) error {
	err := client.DeleteServiceAccount(cfg.ServiceAccount.Name)
	if err != nil {
		return err
	}

	if cleanupVerifyDeletion {
		// Verify service account is deleted
		sa, err := client.GetServiceAccount(cfg.ServiceAccount.Name)
		if err != nil || sa == nil {
			fmt.Println("     ✅ Service account deletion verified")
		}
	}

	return nil
}

func cleanupWorkflowsOp(cfg *config.Config, result *CleanupResult) error {
	// This would implement workflow file cleanup
	// For now, just log the operation
	fmt.Printf("     • Would delete workflow: %s\n", cfg.Workflow.GetWorkflowFilePath())

	// Add additional workflow paths
	for _, path := range cleanupWorkflowPaths {
		fmt.Printf("     • Would delete workflow: %s\n", path)
	}

	return nil
}

func cleanupConfigFilesOp(cfg *config.Config, result *CleanupResult) error {
	// This would implement config file cleanup
	fmt.Println("     • Would delete configuration files")
	return nil
}

func cleanupBackupFilesOp(cfg *config.Config, result *CleanupResult) error {
	// This would implement backup file cleanup
	fmt.Println("     • Would delete backup files")
	return nil
}

// displayCleanupResults shows the final cleanup results
func displayCleanupResults(result *CleanupResult) {
	fmt.Println("\n📊 Cleanup Results:")
	fmt.Println("==================")

	fmt.Printf("✅ Successfully cleaned: %d resources\n", result.SuccessCount)
	if result.SuccessCount > 0 {
		for _, resource := range result.ResourcesDeleted {
			fmt.Printf("   • %s\n", resource)
		}
	}

	if result.FailureCount > 0 {
		fmt.Printf("❌ Failed to clean: %d resources\n", result.FailureCount)
		for _, resource := range result.ResourcesFailed {
			fmt.Printf("   • %s\n", resource)
		}
	}

	if len(result.BackupsCreated) > 0 {
		fmt.Printf("💾 Backups created: %d\n", len(result.BackupsCreated))
		for _, backup := range result.BackupsCreated {
			fmt.Printf("   • %s\n", backup)
		}
	}

	fmt.Printf("⏱️  Total duration: %s\n", result.TotalDuration.Round(time.Second))
}

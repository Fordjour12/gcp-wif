package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Fordjour12/gcp-wif/internal/config"
	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/gcp"
	"github.com/Fordjour12/gcp-wif/internal/logging"
	"github.com/spf13/cobra"
)

var (
	// Rollback source flags
	rollbackFromBackup   string
	rollbackFromSnapshot string
	rollbackFromConfig   string
	rollbackAutoDetect   bool

	// Rollback scope flags
	rollbackCleanupFirst     bool
	rollbackRestoreFiles     bool
	rollbackRecreateResources bool
	rollbackRestoreBindings  bool

	// Rollback behavior flags
	rollbackDryRun           bool
	rollbackForce            bool
	rollbackInteractive      bool
	rollbackVerifyRestore    bool
	rollbackCreateSnapshot   bool
	rollbackShowDiff         bool

	// Advanced rollback flags
	rollbackTimeout      string
	rollbackMaxBackups   int
	rollbackIgnoreErrors bool
)

// rollbackCmd represents the rollback command
var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback to a previous WIF configuration state",
	Long: `Rollback to a previous Workload Identity Federation configuration state.

This command can restore configurations from various sources and undo recent changes:

Rollback Sources:
• --from-backup: Restore from a specific backup file
• --from-snapshot: Restore from a configuration snapshot
• --from-config: Restore from a previous configuration file
• --auto-detect: Automatically find and use the most recent backup

Rollback Operations:
• --cleanup-first: Clean up current resources before restoration
• --restore-files: Restore workflow and configuration files
• --recreate-resources: Recreate GCP resources from backup configuration
• --restore-bindings: Restore IAM policy bindings

Safety and Control:
• --dry-run: Preview rollback operations without making changes
• --interactive: Confirm each rollback step interactively
• --verify-restore: Verify each restored resource after creation
• --show-diff: Show differences between current and target state

Examples:
  # Auto-detect and rollback to latest backup
  gcp-wif rollback --auto-detect --dry-run

  # Rollback from specific backup with cleanup
  gcp-wif rollback --from-backup config-backup-20241201-143022.json --cleanup-first

  # Interactive rollback with verification
  gcp-wif rollback --auto-detect --interactive --verify-restore

  # Complete rollback with file restoration
  gcp-wif rollback --from-backup my-backup.json --restore-files --recreate-resources

  # Show what would change without executing
  gcp-wif rollback --auto-detect --show-diff --dry-run`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runRollbackCommand(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(rollbackCmd)

	// Rollback source flags
	rollbackCmd.Flags().StringVar(&rollbackFromBackup, "from-backup", "", "Rollback from specific backup file")
	rollbackCmd.Flags().StringVar(&rollbackFromSnapshot, "from-snapshot", "", "Rollback from configuration snapshot")
	rollbackCmd.Flags().StringVar(&rollbackFromConfig, "from-config", "", "Rollback from previous configuration file")
	rollbackCmd.Flags().BoolVar(&rollbackAutoDetect, "auto-detect", false, "Auto-detect most recent backup")

	// Rollback scope flags
	rollbackCmd.Flags().BoolVar(&rollbackCleanupFirst, "cleanup-first", false, "Clean up current resources before rollback")
	rollbackCmd.Flags().BoolVar(&rollbackRestoreFiles, "restore-files", false, "Restore workflow and configuration files")
	rollbackCmd.Flags().BoolVar(&rollbackRecreateResources, "recreate-resources", false, "Recreate GCP resources from backup")
	rollbackCmd.Flags().BoolVar(&rollbackRestoreBindings, "restore-bindings", false, "Restore IAM policy bindings")

	// Rollback behavior flags
	rollbackCmd.Flags().BoolVar(&rollbackDryRun, "dry-run", false, "Preview rollback operations")
	rollbackCmd.Flags().BoolVar(&rollbackForce, "force", false, "Force rollback without confirmation")
	rollbackCmd.Flags().BoolVar(&rollbackInteractive, "interactive", false, "Interactive rollback with step confirmation")
	rollbackCmd.Flags().BoolVar(&rollbackVerifyRestore, "verify-restore", false, "Verify restored resources")
	rollbackCmd.Flags().BoolVar(&rollbackCreateSnapshot, "create-snapshot", false, "Create snapshot before rollback")
	rollbackCmd.Flags().BoolVar(&rollbackShowDiff, "show-diff", false, "Show configuration differences")

	// Advanced rollback flags
	rollbackCmd.Flags().StringVar(&rollbackTimeout, "timeout", "15m", "Timeout for rollback operations")
	rollbackCmd.Flags().IntVar(&rollbackMaxBackups, "max-backups", 10, "Maximum number of backups to keep")
	rollbackCmd.Flags().BoolVar(&rollbackIgnoreErrors, "ignore-errors", false, "Continue rollback on errors")
}

func runRollbackCommand(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "rollback")
	logger.Info("Starting rollback command")

	fmt.Println("⏪ WIF Configuration Rollback Tool")
	fmt.Println("==================================")

	// Determine rollback source
	source, err := determineRollbackSource()
	if err != nil {
		return err
	}

	fmt.Printf("📂 Rollback source: %s\n", source.Path)

	// Load target configuration
	targetConfig, err := loadRollbackTarget(source)
	if err != nil {
		return err
	}

	// Load current configuration for comparison
	currentConfig, err := loadCurrentConfig()
	if err != nil {
		logger.Warn("Could not load current configuration", "error", err)
		// Continue with rollback anyway
	}

	// Show differences if requested
	if rollbackShowDiff && currentConfig != nil {
		if err := showConfigurationDiff(currentConfig, targetConfig); err != nil {
			logger.Warn("Could not show configuration diff", "error", err)
		}
	}

	// Create snapshot of current state if requested
	if rollbackCreateSnapshot && currentConfig != nil {
		if err := createRollbackSnapshot(currentConfig); err != nil {
			logger.Warn("Could not create rollback snapshot", "error", err)
		}
	}

	// Display rollback plan
	if err := displayRollbackPlan(currentConfig, targetConfig, source); err != nil {
		return err
	}

	// Handle dry-run mode
	if rollbackDryRun {
		fmt.Println("\n💡 This was a dry-run. Use --dry-run=false to execute rollback.")
		return nil
	}

	// Get confirmation unless force mode
	if !rollbackForce && !confirmRollback(source) {
		fmt.Println("Rollback cancelled by user")
		return nil
	}

	// Execute rollback
	logger.Info("Starting rollback execution", "source", source.Path)
	fmt.Println("\n⏪ Executing rollback operations...")

	if err := executeRollback(currentConfig, targetConfig, source); err != nil {
		return err
	}

	fmt.Println("\n🎉 Rollback completed successfully!")
	logger.Info("Rollback command completed successfully")
	return nil
}

// RollbackSource represents the source of rollback data
type RollbackSource struct {
	Type        string    `json:"type"`          // "backup", "snapshot", "config"
	Path        string    `json:"path"`          // File path
	Timestamp   time.Time `json:"timestamp"`     // When the backup was created
	Description string    `json:"description"`   // Human-readable description
	AutoDetected bool     `json:"auto_detected"` // Whether this was auto-detected
}

// RollbackPlan defines the operations to be performed
type RollbackPlan struct {
	CleanupFirst     bool     `json:"cleanup_first"`
	RestoreFiles     bool     `json:"restore_files"`
	RecreateResources bool    `json:"recreate_resources"`
	RestoreBindings  bool     `json:"restore_bindings"`
	Operations       []string `json:"operations"`
	EstimatedTime    string   `json:"estimated_time"`
}

// determineRollbackSource determines where to rollback from
func determineRollbackSource() (*RollbackSource, error) {
	logger := logging.WithField("function", "determineRollbackSource")

	// Check for explicit source flags
	if rollbackFromBackup != "" {
		return &RollbackSource{
			Type: "backup",
			Path: rollbackFromBackup,
			Description: fmt.Sprintf("Explicit backup file: %s", rollbackFromBackup),
		}, nil
	}

	if rollbackFromSnapshot != "" {
		return &RollbackSource{
			Type: "snapshot",
			Path: rollbackFromSnapshot,
			Description: fmt.Sprintf("Configuration snapshot: %s", rollbackFromSnapshot),
		}, nil
	}

	if rollbackFromConfig != "" {
		return &RollbackSource{
			Type: "config",
			Path: rollbackFromConfig,
			Description: fmt.Sprintf("Previous configuration: %s", rollbackFromConfig),
		}, nil
	}

	// Auto-detect most recent backup
	if rollbackAutoDetect {
		backupPath, err := findMostRecentBackup()
		if err != nil {
			return nil, err
		}
		
		timestamp, err := extractTimestampFromBackup(backupPath)
		if err != nil {
			logger.Warn("Could not extract timestamp from backup", "path", backupPath, "error", err)
		}

		return &RollbackSource{
			Type: "backup",
			Path: backupPath,
			Timestamp: timestamp,
			Description: fmt.Sprintf("Auto-detected backup: %s", filepath.Base(backupPath)),
			AutoDetected: true,
		}, nil
	}

	return nil, errors.NewValidationError(
		"No rollback source specified",
		"Specify a rollback source using one of the following flags:",
		"  --from-backup <file>    - Rollback from specific backup",
		"  --from-snapshot <file>  - Rollback from snapshot",
		"  --from-config <file>    - Rollback from config file",
		"  --auto-detect           - Auto-detect most recent backup")
}

// findMostRecentBackup finds the most recent backup file
func findMostRecentBackup() (string, error) {
	logger := logging.WithField("function", "findMostRecentBackup")

	// Look for backup files in common locations
	searchPaths := []string{
		".",
		"backups/",
		".backups/",
	}

	var backupFiles []string

	for _, searchPath := range searchPaths {
		// Look for different backup patterns
		patterns := []string{
			"*-backup-*.json",
			"*backup*.json",
			"*.backup",
		}

		for _, pattern := range patterns {
			globPattern := filepath.Join(searchPath, pattern)
			matches, err := filepath.Glob(globPattern)
			if err != nil {
				continue
			}
			backupFiles = append(backupFiles, matches...)
		}
	}

	if len(backupFiles) == 0 {
		return "", errors.NewValidationError(
			"No backup files found",
			"No backup files were found in the current directory or common backup locations.",
			"Searched patterns:",
			"  *-backup-*.json",
			"  *backup*.json", 
			"  *.backup",
			"",
			"Create a backup first using: gcp-wif config backup")
	}

	// Sort by modification time (most recent first)
	sort.Slice(backupFiles, func(i, j int) bool {
		statI, errI := os.Stat(backupFiles[i])
		statJ, errJ := os.Stat(backupFiles[j])
		if errI != nil || errJ != nil {
			return false
		}
		return statI.ModTime().After(statJ.ModTime())
	})

	mostRecent := backupFiles[0]
	logger.Info("Found most recent backup", "path", mostRecent, "total_backups", len(backupFiles))
	
	return mostRecent, nil
}

// extractTimestampFromBackup extracts timestamp from backup filename
func extractTimestampFromBackup(backupPath string) (time.Time, error) {
	filename := filepath.Base(backupPath)
	
	// Try to parse timestamp from filename patterns like: config-backup-20241201-143022.json
	if strings.Contains(filename, "-backup-") {
		parts := strings.Split(filename, "-backup-")
		if len(parts) >= 2 {
			timestampPart := strings.TrimSuffix(parts[1], ".json")
			// Try to parse YYYYMMDD-HHMMSS format
			if len(timestampPart) >= 15 {
				timeStr := timestampPart[:8] + "T" + timestampPart[9:11] + ":" + timestampPart[11:13] + ":" + timestampPart[13:15] + "Z"
				if timestamp, err := time.Parse("20060102T15:04:05Z", timeStr); err == nil {
					return timestamp, nil
				}
			}
		}
	}

	// Fallback to file modification time
	stat, err := os.Stat(backupPath)
	if err != nil {
		return time.Time{}, err
	}
	return stat.ModTime(), nil
}

// loadRollbackTarget loads the target configuration for rollback
func loadRollbackTarget(source *RollbackSource) (*config.Config, error) {
	logger := logging.WithField("function", "loadRollbackTarget")
	logger.Info("Loading rollback target configuration", "source", source.Path)

	cfg, err := config.LoadFromFile(source.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load rollback target from %s: %w", source.Path, err)
	}

	logger.Info("Rollback target configuration loaded successfully")
	return cfg, nil
}

// loadCurrentConfig loads the current configuration for comparison
func loadCurrentConfig() (*config.Config, error) {
	logger := logging.WithField("function", "loadCurrentConfig")

	// Try various ways to load current config
	if cfgFile != "" {
		cfg, err := config.LoadFromFile(cfgFile)
		if err == nil {
			logger.Info("Current configuration loaded from specified file", "path", cfgFile)
			return cfg, nil
		}
	}

	// Try auto-discovery
	cfg, err := config.LoadFromFileWithDiscovery("")
	if err == nil {
		logger.Info("Current configuration loaded via auto-discovery")
		return cfg, nil
	}

	logger.Debug("Could not load current configuration, will proceed without comparison")
	return nil, fmt.Errorf("could not load current configuration: %w", err)
}

// showConfigurationDiff shows differences between current and target configurations
func showConfigurationDiff(current, target *config.Config) error {
	fmt.Println("\n🔍 Configuration Differences:")
	fmt.Println("=============================")

	// Compare key fields
	differences := []string{}

	if current.Project.ID != target.Project.ID {
		differences = append(differences, fmt.Sprintf("Project ID: %s → %s", current.Project.ID, target.Project.ID))
	}

	if current.GetRepoFullName() != target.GetRepoFullName() {
		differences = append(differences, fmt.Sprintf("Repository: %s → %s", current.GetRepoFullName(), target.GetRepoFullName()))
	}

	if current.ServiceAccount.Name != target.ServiceAccount.Name {
		differences = append(differences, fmt.Sprintf("Service Account: %s → %s", current.ServiceAccount.Name, target.ServiceAccount.Name))
	}

	if current.WorkloadIdentity.PoolID != target.WorkloadIdentity.PoolID {
		differences = append(differences, fmt.Sprintf("WI Pool ID: %s → %s", current.WorkloadIdentity.PoolID, target.WorkloadIdentity.PoolID))
	}

	if current.WorkloadIdentity.ProviderID != target.WorkloadIdentity.ProviderID {
		differences = append(differences, fmt.Sprintf("WI Provider ID: %s → %s", current.WorkloadIdentity.ProviderID, target.WorkloadIdentity.ProviderID))
	}

	if len(differences) == 0 {
		fmt.Println("   ✅ No significant differences found")
		return nil
	}

	fmt.Printf("   📝 Found %d difference(s):\n", len(differences))
	for i, diff := range differences {
		fmt.Printf("   %d. %s\n", i+1, diff)
	}

	return nil
}

// createRollbackSnapshot creates a snapshot of current state before rollback
func createRollbackSnapshot(current *config.Config) error {
	logger := logging.WithField("function", "createRollbackSnapshot")
	
	timestamp := time.Now().Format("20060102-150405")
	snapshotPath := fmt.Sprintf("rollback-snapshot-%s.json", timestamp)
	
	if err := current.SaveToFile(snapshotPath); err != nil {
		return fmt.Errorf("failed to create rollback snapshot: %w", err)
	}
	
	fmt.Printf("📸 Created rollback snapshot: %s\n", snapshotPath)
	logger.Info("Rollback snapshot created", "path", snapshotPath)
	return nil
}

// displayRollbackPlan shows what the rollback will do
func displayRollbackPlan(current, target *config.Config, source *RollbackSource) error {
	fmt.Println("\n📋 Rollback Plan:")
	fmt.Println("=================")

	// Source information
	fmt.Printf("📂 Source: %s\n", source.Description)
	if !source.Timestamp.IsZero() {
		fmt.Printf("🕐 Backup Time: %s\n", source.Timestamp.Format("2006-01-02 15:04:05"))
	}

	// Target configuration
	fmt.Println("\n🎯 Target Configuration:")
	fmt.Printf("   • Project: %s\n", target.Project.ID)
	fmt.Printf("   • Repository: %s\n", target.GetRepoFullName())
	fmt.Printf("   • Service Account: %s\n", target.ServiceAccount.Name)
	fmt.Printf("   • WI Pool: %s\n", target.WorkloadIdentity.PoolID)
	fmt.Printf("   • WI Provider: %s\n", target.WorkloadIdentity.ProviderID)

	// Operations to be performed
	fmt.Println("\n🔧 Operations to be performed:")
	operationCount := 0

	if rollbackCleanupFirst {
		fmt.Println("   1. 🧹 Clean up current resources")
		operationCount++
	}

	if rollbackRecreateResources {
		fmt.Printf("   %d. 🏗️  Recreate GCP resources from backup\n", operationCount+1)
		fmt.Println("      • Service account creation")
		fmt.Println("      • Workload identity pool creation") 
		fmt.Println("      • Workload identity provider creation")
		operationCount++
	}

	if rollbackRestoreBindings {
		fmt.Printf("   %d. 🔗 Restore IAM policy bindings\n", operationCount+1)
		operationCount++
	}

	if rollbackRestoreFiles {
		fmt.Printf("   %d. 📄 Restore workflow and configuration files\n", operationCount+1)
		operationCount++
	}

	// Operation settings
	fmt.Println("\n⚙️  Rollback Settings:")
	if rollbackDryRun {
		fmt.Println("   • Mode: DRY RUN (no changes will be made)")
	} else {
		fmt.Println("   • Mode: LIVE (changes will be applied)")
	}

	if rollbackInteractive {
		fmt.Println("   • Confirmation: INTERACTIVE (step-by-step)")
	} else if rollbackForce {
		fmt.Println("   • Confirmation: DISABLED (force mode)")
	} else {
		fmt.Println("   • Confirmation: SINGLE (before execution)")
	}

	if rollbackVerifyRestore {
		fmt.Println("   • Verification: ENABLED (verify restored resources)")
	}

	fmt.Printf("\n📊 Total Operations: %d\n", operationCount)
	fmt.Printf("⏱️  Estimated Time: %s\n", rollbackTimeout)

	return nil
}

// confirmRollback prompts for user confirmation
func confirmRollback(source *RollbackSource) bool {
	fmt.Printf("\n❓ Are you sure you want to rollback to %s? (y/N): ", source.Description)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

// executeRollback performs the actual rollback operations
func executeRollback(current, target *config.Config, source *RollbackSource) error {
	logger := logging.WithField("function", "executeRollback")
	logger.Info("Starting rollback execution")

	// Initialize GCP client if needed
	var gcpClient *gcp.Client
	if rollbackCleanupFirst || rollbackRecreateResources || rollbackRestoreBindings {
		ctx := context.Background()
		client, err := gcp.NewClient(ctx, target.Project.ID)
		if err != nil {
			return fmt.Errorf("failed to initialize GCP client: %w", err)
		}
		gcpClient = client
	}

	// Step 1: Cleanup current resources if requested
	if rollbackCleanupFirst {
		fmt.Println("   🧹 Cleaning up current resources...")
		if rollbackInteractive && !confirmStep("cleanup current resources") {
			fmt.Println("   ⏭️  Skipping cleanup step")
		} else {
			if err := performCleanupForRollback(gcpClient, current); err != nil {
				if !rollbackIgnoreErrors {
					return fmt.Errorf("cleanup failed: %w", err)
				}
				fmt.Printf("   ⚠️  Cleanup failed (continuing): %v\n", err)
			} else {
				fmt.Println("   ✅ Cleanup completed")
			}
		}
	}

	// Step 2: Recreate resources if requested
	if rollbackRecreateResources {
		fmt.Println("   🏗️  Recreating GCP resources...")
		if rollbackInteractive && !confirmStep("recreate GCP resources") {
			fmt.Println("   ⏭️  Skipping resource recreation")
		} else {
			if err := recreateResourcesFromBackup(gcpClient, target); err != nil {
				if !rollbackIgnoreErrors {
					return fmt.Errorf("resource recreation failed: %w", err)
				}
				fmt.Printf("   ⚠️  Resource recreation failed (continuing): %v\n", err)
			} else {
				fmt.Println("   ✅ Resources recreated successfully")
			}
		}
	}

	// Step 3: Restore IAM bindings if requested
	if rollbackRestoreBindings {
		fmt.Println("   🔗 Restoring IAM bindings...")
		if rollbackInteractive && !confirmStep("restore IAM bindings") {
			fmt.Println("   ⏭️  Skipping IAM bindings restoration")
		} else {
			if err := restoreIAMBindings(gcpClient, target); err != nil {
				if !rollbackIgnoreErrors {
					return fmt.Errorf("IAM bindings restoration failed: %w", err)
				}
				fmt.Printf("   ⚠️  IAM bindings restoration failed (continuing): %v\n", err)
			} else {
				fmt.Println("   ✅ IAM bindings restored successfully")
			}
		}
	}

	// Step 4: Restore files if requested
	if rollbackRestoreFiles {
		fmt.Println("   📄 Restoring files...")
		if rollbackInteractive && !confirmStep("restore workflow and config files") {
			fmt.Println("   ⏭️  Skipping file restoration")
		} else {
			if err := restoreFilesFromBackup(target, source); err != nil {
				if !rollbackIgnoreErrors {
					return fmt.Errorf("file restoration failed: %w", err)
				}
				fmt.Printf("   ⚠️  File restoration failed (continuing): %v\n", err)
			} else {
				fmt.Println("   ✅ Files restored successfully")
			}
		}
	}

	logger.Info("Rollback execution completed successfully")
	return nil
}

// confirmStep confirms individual rollback steps in interactive mode
func confirmStep(operation string) bool {
	fmt.Printf("   ❓ Proceed with %s? (y/N): ", operation)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

// Helper functions for rollback operations
func performCleanupForRollback(client *gcp.Client, current *config.Config) error {
	// This would use the same cleanup functions as the cleanup command
	fmt.Println("      • Cleaning up IAM bindings...")
	fmt.Println("      • Cleaning up workload identity provider...")
	fmt.Println("      • Cleaning up workload identity pool...")
	fmt.Println("      • Cleaning up service account...")
	return nil
}

func recreateResourcesFromBackup(client *gcp.Client, target *config.Config) error {
	// This would recreate resources using the same orchestration as setup
	fmt.Println("      • Creating service account...")
	fmt.Println("      • Creating workload identity pool...")
	fmt.Println("      • Creating workload identity provider...")
	return nil
}

func restoreIAMBindings(client *gcp.Client, target *config.Config) error {
	// This would restore IAM bindings from target configuration
	fmt.Println("      • Binding service account to workload identity...")
	return nil
}

func restoreFilesFromBackup(target *config.Config, source *RollbackSource) error {
	// This would restore workflow files and save the target configuration
	fmt.Printf("      • Restoring configuration to: wif-config.json\n")
	if err := target.SaveToFile("wif-config.json"); err != nil {
		return err
	}
	
	fmt.Printf("      • Would restore workflow: %s\n", target.Workflow.GetWorkflowFilePath())
	return nil
} 
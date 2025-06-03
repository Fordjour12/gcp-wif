package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Fordjour12/gcp-wif/internal/config"
	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/logging"
	"github.com/Fordjour12/gcp-wif/internal/ui"
	"github.com/spf13/cobra"
)

var (
	// Flags for config subcommands
	outputFormat string
	templateFile string
	backupDir    string
	force        bool
	validate     bool
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration files",
	Long: `Manage GCP Workload Identity Federation configuration files.

This command provides subcommands for creating, validating, displaying,
and managing configuration files for the WIF setup process.

Available subcommands:
- init: Create a new configuration file interactively
- validate: Validate an existing configuration file
- show: Display current configuration settings
- backup: Create backup copies of configuration files`,
}

// configInitCmd creates a new configuration file
var configInitCmd = &cobra.Command{
	Use:   "init [config-file]",
	Short: "Create a new configuration file",
	Long: `Create a new configuration file interactively or from a template.

This command will guide you through creating a configuration file with
all the necessary settings for Workload Identity Federation.

Examples:
  gcp-wif config init                    # Interactive creation with default name
  gcp-wif config init my-config.json     # Interactive creation with custom name
  gcp-wif config init --template app.json  # Create from template file`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runConfigInit(cmd, args); err != nil {
			HandleError(err)
		}
	},
}

// configValidateCmd validates a configuration file
var configValidateCmd = &cobra.Command{
	Use:   "validate [config-file]",
	Short: "Validate a configuration file",
	Long: `Validate the structure and contents of a configuration file.

This command checks that the configuration file has valid JSON syntax,
contains all required fields, and passes validation rules.

Examples:
  gcp-wif config validate                 # Validate default config file
  gcp-wif config validate my-config.json  # Validate specific file`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runConfigValidate(cmd, args); err != nil {
			HandleError(err)
		}
	},
}

// configShowCmd displays current configuration
var configShowCmd = &cobra.Command{
	Use:   "show [config-file]",
	Short: "Display configuration settings",
	Long: `Display the current configuration settings in a readable format.

This command loads and displays the configuration file with syntax
highlighting and organized sections.

Examples:
  gcp-wif config show                     # Show default config file
  gcp-wif config show my-config.json      # Show specific file
  gcp-wif config show --format json       # Show as formatted JSON
  gcp-wif config show --format summary    # Show as summary`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runConfigShow(cmd, args); err != nil {
			HandleError(err)
		}
	},
}

// configBackupCmd creates backup copies
var configBackupCmd = &cobra.Command{
	Use:   "backup [config-file]",
	Short: "Create backup copies of configuration files",
	Long: `Create timestamped backup copies of configuration files.

This command helps preserve configuration versions before making changes.

Examples:
  gcp-wif config backup                   # Backup default config file
  gcp-wif config backup my-config.json    # Backup specific file
  gcp-wif config backup --dir ./backups   # Backup to specific directory`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runConfigBackup(cmd, args); err != nil {
			HandleError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configBackupCmd)

	// Flags for config init
	configInitCmd.Flags().StringVar(&templateFile, "template", "", "Template file to use for initialization")
	configInitCmd.Flags().BoolVar(&force, "force", false, "Overwrite existing configuration file")

	// Flags for config show
	configShowCmd.Flags().StringVar(&outputFormat, "format", "summary", "Output format (json, yaml, summary)")

	// Flags for config backup
	configBackupCmd.Flags().StringVar(&backupDir, "dir", "./backups", "Directory for backup files")

	// Flags for config validate
	configValidateCmd.Flags().BoolVar(&validate, "strict", false, "Enable strict validation mode")
}

// runConfigInit handles the config init command
func runConfigInit(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "config_init")
	logger.Info("Starting configuration initialization")

	// Determine config file path
	configFile := getConfigFilePath(args)

	// Check if file exists and handle overwrite
	if !force {
		if _, err := os.Stat(configFile); err == nil {
			return errors.NewConfigurationError(
				fmt.Sprintf("Configuration file already exists: %s", configFile),
				"Use --force to overwrite the existing file",
				"Choose a different filename",
				"Use 'gcp-wif config backup' to save the current version first")
		}
	}

	var cfg *config.Config

	// Load from template if specified
	if templateFile != "" {
		logger.Info("Loading template file", "template", templateFile)
		templateCfg, err := config.LoadFromFile(templateFile)
		if err != nil {
			return errors.WrapError(err, errors.ErrorTypeConfiguration, "TEMPLATE_LOAD_FAILED",
				fmt.Sprintf("Failed to load template file: %s", templateFile))
		}
		cfg = templateCfg
		logger.Info("Template loaded successfully")
	} else {
		// Start with default configuration
		cfg = config.DefaultConfig()
		logger.Info("Using default configuration template")
	}

	// Run interactive configuration
	fmt.Println("🚀 Initializing new configuration file...")
	fmt.Printf("📁 Target file: %s\n\n", configFile)

	interactiveCfg, err := ui.RunInteractiveConfig(cfg)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeInternal, "INTERACTIVE_CONFIG_FAILED",
			"Failed to run interactive configuration")
	}

	// Save the configuration
	if err := interactiveCfg.SaveToFile(configFile); err != nil {
		return err
	}

	fmt.Printf("✅ Configuration file created successfully: %s\n", configFile)
	logger.Info("Configuration initialization completed", "file", configFile)

	return nil
}

// runConfigValidate handles the config validate command
func runConfigValidate(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "config_validate")
	logger.Info("Starting configuration validation")

	configFile := getConfigFilePath(args)

	// Load and validate configuration
	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		return err
	}

	// Perform validation
	result := cfg.ValidateSchema()

	// Display results
	fmt.Printf("📋 Validating configuration: %s\n\n", configFile)

	if result.Valid {
		fmt.Println("✅ Configuration is valid!")

		// Show warnings if any
		if len(result.Warnings) > 0 {
			fmt.Printf("\n⚠️  Warnings (%d):\n", len(result.Warnings))
			for _, warning := range result.Warnings {
				fmt.Printf("   • %s: %s\n", warning.Field, warning.Message)
			}
		}

		// Show info if any
		if len(result.Info) > 0 {
			fmt.Printf("\n💡 Information (%d):\n", len(result.Info))
			for _, info := range result.Info {
				fmt.Printf("   • %s: %s\n", info.Field, info.Message)
			}
		}
	} else {
		fmt.Printf("❌ Configuration validation failed (%d errors):\n\n", len(result.Errors))
		for i, valErr := range result.Errors {
			fmt.Printf("%d. %s: %s\n", i+1, valErr.Field, valErr.Message)
			if valErr.Value != "" {
				fmt.Printf("   Current value: %s\n", valErr.Value)
			}
		}

		return errors.NewValidationError(
			"Configuration validation failed",
			fmt.Sprintf("Found %d validation errors", len(result.Errors)),
			"Check the errors above and fix the configuration file",
			"Use 'gcp-wif config show' to view current settings")
	}

	logger.Info("Configuration validation completed", "valid", result.Valid, "errors", len(result.Errors))
	return nil
}

// runConfigShow handles the config show command
func runConfigShow(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "config_show")
	logger.Info("Displaying configuration")

	configFile := getConfigFilePath(args)

	// Load configuration
	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		return err
	}

	fmt.Printf("📋 Configuration: %s\n\n", configFile)

	switch outputFormat {
	case "json":
		jsonStr, err := cfg.ToJSON()
		if err != nil {
			return errors.WrapError(err, errors.ErrorTypeInternal, "JSON_FORMAT_FAILED",
				"Failed to format configuration as JSON")
		}
		fmt.Println(jsonStr)

	case "summary":
		displayDetailedConfigSummary(cfg)

	default:
		return errors.NewValidationError(
			fmt.Sprintf("Unsupported output format: %s", outputFormat),
			"Supported formats: json, summary")
	}

	logger.Info("Configuration displayed successfully", "format", outputFormat)
	return nil
}

// runConfigBackup handles the config backup command
func runConfigBackup(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "config_backup")
	logger.Info("Creating configuration backup")

	configFile := getConfigFilePath(args)

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return errors.NewConfigurationError(
			fmt.Sprintf("Configuration file not found: %s", configFile),
			"Create a configuration file first using 'gcp-wif config init'")
	}

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return errors.WrapError(err, errors.ErrorTypeFileSystem, "BACKUP_DIR_CREATE_FAILED",
			fmt.Sprintf("Failed to create backup directory: %s", backupDir))
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	basename := strings.TrimSuffix(filepath.Base(configFile), filepath.Ext(configFile))
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s-backup-%s.json", basename, timestamp))

	// Copy file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeFileSystem, "CONFIG_READ_FAILED",
			fmt.Sprintf("Failed to read configuration file: %s", configFile))
	}

	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		return errors.WrapError(err, errors.ErrorTypeFileSystem, "BACKUP_WRITE_FAILED",
			fmt.Sprintf("Failed to write backup file: %s", backupFile))
	}

	fmt.Printf("✅ Configuration backup created: %s\n", backupFile)
	logger.Info("Configuration backup created successfully", "original", configFile, "backup", backupFile)

	return nil
}

// getConfigFilePath determines the configuration file path from args or default
func getConfigFilePath(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	if cfgFile != "" {
		return cfgFile
	}
	return "./wif-config.json"
}

// displayDetailedConfigSummary displays a human-readable configuration summary
func displayDetailedConfigSummary(cfg *config.Config) {
	fmt.Println("📋 Configuration Summary:")
	fmt.Println("========================")

	// Project information
	fmt.Printf("🏗️  Project ID: %s\n", cfg.Project.ID)
	if cfg.Project.Number != "" {
		fmt.Printf("🔢 Project Number: %s\n", cfg.Project.Number)
	}
	if cfg.Project.Region != "" {
		fmt.Printf("🌍 Project Region: %s\n", cfg.Project.Region)
	}

	// Repository information
	fmt.Printf("📚 Repository: %s\n", cfg.GetRepoFullName())
	if cfg.Repository.Ref != "" {
		fmt.Printf("🌿 Reference: %s\n", cfg.Repository.Ref)
	}
	if len(cfg.Repository.Branches) > 0 {
		fmt.Printf("🌿 Branches: %s\n", strings.Join(cfg.Repository.Branches, ", "))
	}
	if len(cfg.Repository.Tags) > 0 {
		fmt.Printf("🏷️  Tags: %s\n", strings.Join(cfg.Repository.Tags, ", "))
	}
	if cfg.Repository.PullRequest {
		fmt.Println("🔄 Pull Request: Enabled")
	}

	// Service Account information
	fmt.Printf("👤 Service Account: %s\n", cfg.ServiceAccount.Name)
	fmt.Printf("📧 Service Account Email: %s\n", cfg.GetServiceAccountEmail())
	if cfg.ServiceAccount.DisplayName != "" {
		fmt.Printf("🏷️  Display Name: %s\n", cfg.ServiceAccount.DisplayName)
	}
	if len(cfg.ServiceAccount.Roles) > 0 {
		fmt.Printf("🔐 IAM Roles: %s\n", strings.Join(cfg.ServiceAccount.Roles, ", "))
	}

	// Workload Identity information
	fmt.Printf("🔗 Workload Identity Pool: %s\n", cfg.WorkloadIdentity.PoolID)
	fmt.Printf("🔌 Workload Identity Provider: %s\n", cfg.WorkloadIdentity.ProviderID)
	if len(cfg.WorkloadIdentity.Conditions) > 0 {
		fmt.Printf("🛡️  Security Conditions: %s\n", strings.Join(cfg.WorkloadIdentity.Conditions, ", "))
	}

	// Cloud Run information (if configured)
	if cfg.CloudRun.ServiceName != "" {
		fmt.Printf("☁️  Cloud Run Service: %s\n", cfg.CloudRun.ServiceName)
		fmt.Printf("🌍 Cloud Run Region: %s\n", cfg.CloudRun.Region)
		if cfg.CloudRun.Image != "" {
			fmt.Printf("🐳 Docker Image: %s\n", cfg.CloudRun.Image)
		}
		if cfg.CloudRun.Port != 0 {
			fmt.Printf("🔌 Port: %d\n", cfg.CloudRun.Port)
		}
		if cfg.GetCloudRunURL() != "" {
			fmt.Printf("🌐 Cloud Run URL: %s\n", cfg.GetCloudRunURL())
		}
	}

	// Workflow information
	fmt.Printf("⚡ Workflow File: %s\n", cfg.GetWorkflowFilePath())
	if len(cfg.Workflow.Triggers) > 0 {
		fmt.Printf("🎯 Triggers: %s\n", strings.Join(cfg.Workflow.Triggers, ", "))
	}
	if cfg.Workflow.Environment != "" {
		fmt.Printf("🌍 Environment: %s\n", cfg.Workflow.Environment)
	}

	// Advanced settings
	if cfg.Advanced.DryRun {
		fmt.Println("🧪 Dry Run: Enabled")
	}
	if cfg.Advanced.Timeout != "" {
		fmt.Printf("⏱️  Timeout: %s\n", cfg.Advanced.Timeout)
	}
	if cfg.Advanced.BackupExisting {
		fmt.Println("💾 Backup Existing: Enabled")
	}

	fmt.Printf("\n📊 Configuration Version: %s\n", cfg.Version)
}

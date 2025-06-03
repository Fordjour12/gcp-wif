package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Fordjour12/gcp-wif/internal/config"
	"github.com/Fordjour12/gcp-wif/internal/github"
	"github.com/Fordjour12/gcp-wif/internal/logging"
	"github.com/spf13/cobra"
)

var (
	// Workflow command flags
	workflowConfigFile string
	workflowOutputPath string
	workflowFilename   string
	workflowPreview    bool
	workflowValidate   bool
	workflowOverwrite  bool
	workflowBackup     bool
	workflowDryRun     bool
	workflowFormat     string
	workflowTemplate   string
)

// workflowCmd represents the workflow command
var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Generate and manage GitHub Actions workflows",
	Long: `Generate, preview, and manage GitHub Actions workflows with WIF authentication.

This command provides comprehensive workflow management capabilities:
- Generate workflows from configuration files
- Preview workflows before writing
- Validate workflow content
- Manage workflow files with backup and overwrite protection
- Support for multiple workflow templates

Available subcommands:
- generate: Generate workflow from configuration
- preview:  Preview workflow without writing
- validate: Validate workflow configuration and content
- info:     Show workflow file information

Examples:
  # Generate workflow from config file
  gcp-wif workflow generate --config config.json

  # Preview workflow without writing
  gcp-wif workflow preview --config config.json

  # Generate with custom output path
  gcp-wif workflow generate --config config.json --output-path .github/workflows --filename deploy-staging.yml

  # Validate workflow configuration
  gcp-wif workflow validate --config config.json

Use --help with any subcommand for detailed options.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logging for workflow commands
		config := logging.DefaultConfig()
		config.Level = logging.LevelInfo
		if verbose {
			config.Level = logging.LevelDebug
			config.Verbose = true
		}
		logging.InitGlobalLogger(config)
	},
}

// workflowGenerateCmd represents the workflow generate command
var workflowGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate GitHub Actions workflow file",
	Long: `Generate a GitHub Actions workflow file from configuration.

This command creates a complete GitHub Actions workflow file with:
- WIF authentication setup
- Docker build and push to Artifact Registry
- Cloud Run deployment
- Health checks and validation
- Environment-specific configuration

The workflow file will be written to the specified location with
comprehensive error handling and optional backup creation.

Examples:
  # Basic generation
  gcp-wif workflow generate --config config.json

  # Generate with overwrite protection
  gcp-wif workflow generate --config config.json --backup

  # Generate to custom location
  gcp-wif workflow generate --config config.json --output-path .github/workflows --filename custom-deploy.yml

  # Dry run to see what would be generated
  gcp-wif workflow generate --config config.json --dry-run`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runWorkflowGenerate(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

// workflowPreviewCmd represents the workflow preview command
var workflowPreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview GitHub Actions workflow without writing",
	Long: `Preview a GitHub Actions workflow without writing it to file.

This command generates and displays the workflow content, providing
detailed information about what would be created including:
- File size and line count
- Validation results
- File existence checks
- Content preview or summary

Perfect for reviewing workflows before committing to file creation.

Examples:
  # Basic preview
  gcp-wif workflow preview --config config.json

  # Show detailed preview with content
  gcp-wif workflow preview --config config.json --format full

  # Validate during preview
  gcp-wif workflow preview --config config.json --validate`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runWorkflowPreview(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

// workflowValidateCmd represents the workflow validate command
var workflowValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate workflow configuration and content",
	Long: `Validate workflow configuration and generated content.

This command performs comprehensive validation including:
- Configuration schema validation
- Required field checks
- Workflow content validation
- YAML structure verification
- WIF authentication setup validation

Useful for CI/CD pipelines and configuration validation.

Examples:
  # Validate configuration
  gcp-wif workflow validate --config config.json

  # Validate with detailed output
  gcp-wif workflow validate --config config.json --verbose`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runWorkflowValidate(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

// workflowInfoCmd represents the workflow info command
var workflowInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show workflow file information",
	Long: `Display information about workflow files and configuration.

This command shows detailed information about:
- Workflow file status and metadata
- Configuration summary
- File system information
- Template information

Examples:
  # Show workflow info
  gcp-wif workflow info --config config.json

  # Show info for specific workflow file
  gcp-wif workflow info --output-path .github/workflows --filename deploy.yml`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runWorkflowInfo(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(workflowCmd)

	// Add subcommands
	workflowCmd.AddCommand(workflowGenerateCmd)
	workflowCmd.AddCommand(workflowPreviewCmd)
	workflowCmd.AddCommand(workflowValidateCmd)
	workflowCmd.AddCommand(workflowInfoCmd)

	// Persistent flags for all workflow commands
	workflowCmd.PersistentFlags().StringVarP(&workflowConfigFile, "config", "c", "", "Configuration file path")
	workflowCmd.PersistentFlags().StringVar(&workflowOutputPath, "output-path", "", "Workflow output directory path (default: .github/workflows)")
	workflowCmd.PersistentFlags().StringVar(&workflowFilename, "filename", "", "Workflow filename (default: from config)")
	workflowCmd.PersistentFlags().StringVar(&workflowTemplate, "template", "", "Workflow template type (default, production, staging)")

	// Generate command flags
	workflowGenerateCmd.Flags().BoolVar(&workflowOverwrite, "overwrite", false, "Overwrite existing workflow file")
	workflowGenerateCmd.Flags().BoolVar(&workflowBackup, "backup", true, "Create backup of existing file")
	workflowGenerateCmd.Flags().BoolVar(&workflowDryRun, "dry-run", false, "Show what would be done without writing files")
	workflowGenerateCmd.Flags().BoolVar(&workflowValidate, "validate", true, "Validate workflow content before writing")

	// Preview command flags
	workflowPreviewCmd.Flags().StringVar(&workflowFormat, "format", "summary", "Preview format (summary, full, json)")
	workflowPreviewCmd.Flags().BoolVar(&workflowValidate, "validate", true, "Validate workflow content during preview")

	// Validate command flags
	workflowValidateCmd.Flags().BoolVar(&workflowPreview, "preview", false, "Include workflow content validation")

	// Info command flags
	workflowInfoCmd.Flags().StringVar(&workflowFormat, "format", "summary", "Info format (summary, json)")
}

// runWorkflowGenerate handles the workflow generate command
func runWorkflowGenerate(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "workflow.generate")
	logger.Info("Starting workflow generation")

	// Load configuration
	cfg, err := loadWorkflowConfig()
	if err != nil {
		return err
	}

	// Apply command-line overrides
	if err := applyWorkflowFlags(cfg); err != nil {
		return err
	}

	// Set up write options
	writeOptions := github.WriteWorkflowFileOptions{
		CreateBackup:      workflowBackup,
		OverwriteExisting: workflowOverwrite,
		DryRun:            workflowDryRun,
		Validate:          workflowValidate,
	}

	// Generate and write workflow
	fmt.Println("🔧 Generating GitHub Actions workflow...")
	if err := cfg.Workflow.GenerateAndWriteWorkflowWithOptions(writeOptions); err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Display success message
	if workflowDryRun {
		fmt.Printf("✅ Dry run: Workflow would be written to %s\n", cfg.Workflow.GetWorkflowFilePath())
	} else {
		fmt.Printf("✅ Workflow generated successfully: %s\n", cfg.Workflow.GetWorkflowFilePath())
	}

	return nil
}

// runWorkflowPreview handles the workflow preview command
func runWorkflowPreview(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "workflow.preview")
	logger.Info("Starting workflow preview")

	// Load configuration
	cfg, err := loadWorkflowConfig()
	if err != nil {
		return err
	}

	// Apply command-line overrides
	if err := applyWorkflowFlags(cfg); err != nil {
		return err
	}

	// Generate preview
	fmt.Println("👀 Generating workflow preview...")
	preview, err := cfg.Workflow.GenerateWorkflowPreview()
	if err != nil {
		return fmt.Errorf("failed to generate preview: %w", err)
	}

	// Display preview based on format
	switch workflowFormat {
	case "summary":
		fmt.Println(preview.GetPreviewSummary())
	case "full":
		fmt.Println(preview.GetPreviewSummary())
		fmt.Println("\n📄 Workflow Content:")
		fmt.Println("==================")
		fmt.Println(preview.Content)
	case "json":
		// Remove config from preview to avoid circular reference in JSON
		preview.Config = nil
		if jsonBytes, err := json.MarshalIndent(preview, "", "  "); err == nil {
			fmt.Println(string(jsonBytes))
		} else {
			return fmt.Errorf("failed to marshal preview to JSON: %w", err)
		}
	default:
		return fmt.Errorf("invalid format: %s (use: summary, full, json)", workflowFormat)
	}

	return nil
}

// runWorkflowValidate handles the workflow validate command
func runWorkflowValidate(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "workflow.validate")
	logger.Info("Starting workflow validation")

	// Load configuration
	cfg, err := loadWorkflowConfig()
	if err != nil {
		return err
	}

	// Apply command-line overrides
	if err := applyWorkflowFlags(cfg); err != nil {
		return err
	}

	fmt.Println("🔍 Validating workflow configuration...")

	// Validate configuration
	if err := cfg.Workflow.ValidateConfig(); err != nil {
		fmt.Printf("❌ Configuration validation failed: %v\n", err)
		return err
	}

	fmt.Println("✅ Configuration validation passed")

	// Validate content if requested
	if workflowPreview {
		fmt.Println("🔍 Validating workflow content...")
		content, err := cfg.Workflow.GenerateWorkflow()
		if err != nil {
			return fmt.Errorf("failed to generate workflow for validation: %w", err)
		}

		if err := cfg.Workflow.ValidateWorkflowContent(content); err != nil {
			fmt.Printf("❌ Content validation failed: %v\n", err)
			return err
		}

		fmt.Println("✅ Content validation passed")
	}

	fmt.Println("🎉 All validations passed successfully!")
	return nil
}

// runWorkflowInfo handles the workflow info command
func runWorkflowInfo(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "workflow.info")
	logger.Info("Starting workflow info")

	// Load configuration
	cfg, err := loadWorkflowConfig()
	if err != nil {
		return err
	}

	// Apply command-line overrides
	if err := applyWorkflowFlags(cfg); err != nil {
		return err
	}

	// Get workflow file info
	fileInfo, err := cfg.Workflow.GetWorkflowFileInfo()
	if err != nil {
		return fmt.Errorf("failed to get workflow file info: %w", err)
	}

	// Display info based on format
	switch workflowFormat {
	case "summary":
		displayWorkflowInfoSummary(cfg, fileInfo)
	case "json":
		if jsonBytes, err := json.MarshalIndent(fileInfo, "", "  "); err == nil {
			fmt.Println(string(jsonBytes))
		} else {
			return fmt.Errorf("failed to marshal file info to JSON: %w", err)
		}
	default:
		return fmt.Errorf("invalid format: %s (use: summary, json)", workflowFormat)
	}

	return nil
}

// loadWorkflowConfig loads workflow configuration from file or creates default
func loadWorkflowConfig() (*config.Config, error) {
	var cfg *config.Config
	var err error

	if workflowConfigFile != "" {
		cfg, err = config.LoadFromFile(workflowConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	} else {
		// Try auto-discovery
		cfg, err = config.LoadFromFileWithDiscovery("")
		if err != nil {
			return nil, fmt.Errorf("failed to load or discover config file: %w", err)
		}
	}

	return cfg, nil
}

// applyWorkflowFlags applies command-line flags to workflow configuration
func applyWorkflowFlags(cfg *config.Config) error {
	// Apply output path override
	if workflowOutputPath != "" {
		cfg.Workflow.Path = workflowOutputPath
	}

	// Apply filename override
	if workflowFilename != "" {
		cfg.Workflow.Filename = workflowFilename
	}

	// Apply template override
	if workflowTemplate != "" {
		switch strings.ToLower(workflowTemplate) {
		case "production":
			productionConfig := github.DefaultProductionWorkflowConfig()
			cfg.Workflow = *productionConfig
		case "staging":
			stagingConfig := github.DefaultStagingWorkflowConfig()
			cfg.Workflow = *stagingConfig
		case "default":
			defaultConfig := github.DefaultWorkflowConfig()
			cfg.Workflow = *defaultConfig
		default:
			return fmt.Errorf("invalid template: %s (use: default, production, staging)", workflowTemplate)
		}
	}

	// Ensure required fields are populated
	cfg.SetDefaults()

	return nil
}

// displayWorkflowInfoSummary displays a formatted summary of workflow information
func displayWorkflowInfoSummary(cfg *config.Config, fileInfo *github.WorkflowFileInfo) {
	fmt.Println("📊 Workflow Information:")
	fmt.Println("=======================")

	// Basic info
	fmt.Printf("📄 Name: %s\n", cfg.Workflow.Name)
	fmt.Printf("📍 Path: %s\n", fileInfo.Path)
	fmt.Printf("📁 Directory: %s\n", fileInfo.Dir)

	// File status
	if fileInfo.Exists {
		fmt.Printf("✅ Status: File exists\n")
		fmt.Printf("📊 Size: %d bytes\n", fileInfo.Size)
		fmt.Printf("🕒 Modified: %s\n", fileInfo.ModTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("🔒 Permissions: %s\n", fileInfo.Mode)
	} else {
		fmt.Printf("📝 Status: New file (will be created)\n")
	}

	// Directory status
	if fileInfo.DirExists {
		fmt.Printf("📂 Directory: Exists (%s)\n", fileInfo.DirMode)
	} else {
		fmt.Printf("📂 Directory: Will be created\n")
	}

	// Configuration summary
	fmt.Printf("\n🔧 Configuration:\n")
	fmt.Printf("🎯 Triggers: ")
	var triggers []string
	if cfg.Workflow.Triggers.Push.Enabled {
		triggers = append(triggers, "Push")
	}
	if cfg.Workflow.Triggers.PullRequest.Enabled {
		triggers = append(triggers, "Pull Request")
	}
	if cfg.Workflow.Triggers.Manual {
		triggers = append(triggers, "Manual")
	}
	if cfg.Workflow.Triggers.Release {
		triggers = append(triggers, "Release")
	}
	if len(cfg.Workflow.Triggers.Schedule) > 0 {
		triggers = append(triggers, "Schedule")
	}
	fmt.Printf("%s\n", strings.Join(triggers, ", "))

	if len(cfg.Workflow.Advanced.Environments) > 0 {
		var envNames []string
		for name := range cfg.Workflow.Advanced.Environments {
			envNames = append(envNames, name)
		}
		fmt.Printf("🌍 Environments: %s\n", strings.Join(envNames, ", "))
	}

	if len(cfg.Workflow.Advanced.HealthChecks) > 0 {
		fmt.Printf("🔍 Health Checks: %d configured\n", len(cfg.Workflow.Advanced.HealthChecks))
	}

	fmt.Printf("🏗️  Project: %s\n", cfg.Workflow.ProjectID)
	fmt.Printf("🌍 Region: %s\n", cfg.Workflow.Region)
	fmt.Printf("☁️  Service: %s\n", cfg.Workflow.ServiceName)
}

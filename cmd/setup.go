package cmd

import (
	"fmt"
	"strings"

	"github.com/Fordjour12/gcp-wif/internal/config"
	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/logging"
	"github.com/Fordjour12/gcp-wif/internal/ui"
	"github.com/spf13/cobra"
)

var (
	// Legacy flag variables for backward compatibility
	projectID        string
	githubRepo       string
	serviceAccount   string
	cloudRunService  string
	cloudRunRegion   string
	artifactRegistry string
	interactive      bool

	// Enhanced flag variables for comprehensive configuration support
	// Project flags
	projectNumber string
	projectRegion string

	// Repository flags
	repoOwner    string
	repoName     string
	repoRef      string
	repoBranches []string
	repoTags     []string
	repoPR       bool

	// Service Account flags
	saDisplayName string
	saDescription string
	saRoles       []string
	saCreateNew   bool

	// Workload Identity flags
	wiPoolName     string
	wiPoolID       string
	wiProviderName string
	wiProviderID   string
	wiConditions   []string

	// Cloud Run flags
	crImage        string
	crPort         int
	crCPULimit     string
	crMemoryLimit  string
	crMaxInstances int
	crMinInstances int

	// Workflow flags
	wfName        string
	wfFilename    string
	wfPath        string
	wfTriggers    []string
	wfEnvironment string
	wfDockerImage string

	// Advanced flags
	dryRun           bool
	skipValidation   bool
	forceUpdate      bool
	backupExisting   bool
	cleanupOnFailure bool
	enableAPIs       []string
	timeout          string
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up Workload Identity Federation for a GitHub repository",
	Long: `Configure Google Cloud Workload Identity Federation (WIF) for GitHub Actions.

This command will:
1. Create a service account with required IAM roles
2. Set up Workload Identity Pool and Provider
3. Configure security conditions for repository access
4. Generate GitHub Actions workflow file

You can run this command in three ways:
1. Interactive mode (default): gcp-wif setup
2. With flags: gcp-wif setup --project my-project --repo myorg/myrepo
3. With config file: gcp-wif setup --config config.json

Comprehensive flag support includes:
- Project: --project-id, --project-number, --project-region
- Repository: --repo-owner, --repo-name, --repo-branches, --repo-tags
- Service Account: --service-account, --sa-display-name, --sa-roles
- Workload Identity: --wi-pool-id, --wi-provider-id, --wi-conditions
- Cloud Run: --cr-image, --cr-port, --cr-cpu-limit, --cr-memory-limit
- Workflow: --wf-name, --wf-filename, --wf-triggers, --wf-environment
- Advanced: --dry-run, --skip-validation, --force-update, --timeout

Use --help to see all available flags with detailed descriptions.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSetup(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)

	// Setup command flags (maintaining backward compatibility)
	setupCmd.Flags().StringVarP(&projectID, "project", "p", "", "Google Cloud Project ID")
	setupCmd.Flags().StringVar(&projectID, "project-id", "", "Google Cloud Project ID (alias for --project)")
	setupCmd.Flags().StringVarP(&githubRepo, "repo", "r", "", "GitHub repository (format: org/repo)")
	setupCmd.Flags().StringVarP(&serviceAccount, "service-account", "s", "", "Service account name")
	setupCmd.Flags().StringVar(&cloudRunService, "service", "", "Cloud Run service name")
	setupCmd.Flags().StringVar(&cloudRunRegion, "region", "us-central1", "Cloud Run region")
	setupCmd.Flags().StringVar(&artifactRegistry, "registry", "", "Artifact Registry repository name")
	setupCmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "Run in interactive mode")

	// Additional flags for comprehensive configuration support
	setupCmd.Flags().StringVar(&projectNumber, "project-number", "", "Google Cloud Project Number")
	setupCmd.Flags().StringVar(&projectRegion, "project-region", "", "Google Cloud Project Region")
	setupCmd.Flags().StringVar(&repoOwner, "repo-owner", "", "GitHub repository owner")
	setupCmd.Flags().StringVar(&repoName, "repo-name", "", "GitHub repository name")
	setupCmd.Flags().StringVar(&repoRef, "repo-ref", "", "GitHub repository reference")
	setupCmd.Flags().StringSliceVar(&repoBranches, "repo-branches", []string{}, "GitHub repository branches")
	setupCmd.Flags().StringSliceVar(&repoTags, "repo-tags", []string{}, "GitHub repository tags")
	setupCmd.Flags().BoolVar(&repoPR, "repo-pr", false, "GitHub repository is a pull request")
	setupCmd.Flags().StringVar(&saDisplayName, "sa-display-name", "", "Service account display name")
	setupCmd.Flags().StringVar(&saDescription, "sa-description", "", "Service account description")
	setupCmd.Flags().StringSliceVar(&saRoles, "sa-roles", []string{}, "Service account IAM roles")
	setupCmd.Flags().BoolVar(&saCreateNew, "sa-create-new", false, "Create a new service account")
	setupCmd.Flags().StringVar(&wiPoolName, "wi-pool-name", "", "Workload Identity Pool name")
	setupCmd.Flags().StringVar(&wiPoolID, "wi-pool-id", "", "Workload Identity Pool ID")
	setupCmd.Flags().StringVar(&wiProviderName, "wi-provider-name", "", "Workload Identity Provider name")
	setupCmd.Flags().StringVar(&wiProviderID, "wi-provider-id", "", "Workload Identity Provider ID")
	setupCmd.Flags().StringSliceVar(&wiConditions, "wi-conditions", []string{}, "Workload Identity conditions")
	setupCmd.Flags().StringVar(&crImage, "cr-image", "", "Cloud Run image")
	setupCmd.Flags().IntVar(&crPort, "cr-port", 0, "Cloud Run port")
	setupCmd.Flags().StringVar(&crCPULimit, "cr-cpu-limit", "", "Cloud Run CPU limit")
	setupCmd.Flags().StringVar(&crMemoryLimit, "cr-memory-limit", "", "Cloud Run memory limit")
	setupCmd.Flags().IntVar(&crMaxInstances, "cr-max-instances", 0, "Cloud Run maximum instances")
	setupCmd.Flags().IntVar(&crMinInstances, "cr-min-instances", 0, "Cloud Run minimum instances")
	setupCmd.Flags().StringVar(&wfName, "wf-name", "", "Workflow name")
	setupCmd.Flags().StringVar(&wfFilename, "wf-filename", "", "Workflow filename")
	setupCmd.Flags().StringVar(&wfPath, "wf-path", "", "Workflow path")
	setupCmd.Flags().StringSliceVar(&wfTriggers, "wf-triggers", []string{}, "Workflow triggers")
	setupCmd.Flags().StringVar(&wfEnvironment, "wf-environment", "", "Workflow environment")
	setupCmd.Flags().StringVar(&wfDockerImage, "wf-docker-image", "", "Workflow Docker image")
	setupCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Enable dry run")
	setupCmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "Skip configuration validation")
	setupCmd.Flags().BoolVar(&forceUpdate, "force-update", false, "Force update")
	setupCmd.Flags().BoolVar(&backupExisting, "backup-existing", false, "Backup existing configuration")
	setupCmd.Flags().BoolVar(&cleanupOnFailure, "cleanup-on-failure", false, "Cleanup on failure")
	setupCmd.Flags().StringSliceVar(&enableAPIs, "enable-apis", []string{}, "Enable APIs")
	setupCmd.Flags().StringVar(&timeout, "timeout", "", "Timeout")
}

func runSetup(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "setup")
	logger.Info("Starting Workload Identity Federation setup")

	fmt.Println("🚀 Starting Workload Identity Federation setup...")

	// Load or create configuration
	cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}

	// Override config with command-line flags if provided
	if err := applyFlagsToConfig(cfg); err != nil {
		return err
	}

	// Run interactive mode if enabled and missing required fields
	if interactive && (cfg.Project.ID == "" || cfg.Repository.Owner == "" || cfg.Repository.Name == "") {
		logger.Info("Starting interactive configuration mode")
		fmt.Println("📝 Running interactive configuration...")

		interactiveCfg, err := ui.RunInteractiveConfig(cfg)
		if err != nil {
			logger.Error("Interactive configuration failed", "error", err)
			return err
		}
		cfg = interactiveCfg
		logger.Info("Interactive configuration completed successfully")
	}

	// Validate configuration
	result := cfg.ValidateSchema()
	if !result.Valid {
		logger.Error("Configuration validation failed", "errors", len(result.Errors))
		return formatValidationErrors(result)
	}

	// Log warnings if any
	for _, warning := range result.Warnings {
		logger.Warn("Configuration warning", "field", warning.Field, "message", warning.Message)
	}

	// Display configuration summary
	displayConfigSummary(cfg)

	logger.Info("Setup command ready with complete configuration")
	fmt.Println("✅ Configuration complete! Ready to proceed with Workload Identity Federation setup.")
	return nil
}

// loadOrCreateConfig loads configuration from file or creates a new one
func loadOrCreateConfig() (*config.Config, error) {
	logger := logging.WithField("function", "loadOrCreateConfig")

	// If config file is specified, try to load it
	if cfgFile != "" {
		logger.Debug("Loading configuration from file", "path", cfgFile)
		cfg, err := config.LoadFromFile(cfgFile)
		if err != nil {
			return nil, err
		}
		logger.Info("Configuration loaded from file", "path", cfgFile, "version", cfg.Version)
		return cfg, nil
	}

	// If we have basic parameters from flags, create a new config
	if projectID != "" && githubRepo != "" {
		parts := strings.Split(githubRepo, "/")
		if len(parts) != 2 {
			return nil, errors.NewValidationError(
				"Invalid repository format",
				"Repository should be in format 'owner/name'",
				"Example: --repo myusername/my-repository")
		}

		logger.Debug("Creating new configuration from flags")
		cfg := config.NewConfig(projectID, parts[0], parts[1])
		logger.Info("Configuration created from command-line flags")
		return cfg, nil
	}

	// Create default config for interactive mode
	logger.Debug("Creating default configuration for interactive mode")
	cfg := config.DefaultConfig()
	logger.Info("Default configuration created for interactive mode")
	return cfg, nil
}

// applyFlagsToConfig applies command-line flags to override configuration values
func applyFlagsToConfig(cfg *config.Config) error {
	logger := logging.WithField("function", "applyFlagsToConfig")

	// Apply project ID
	if projectID != "" {
		cfg.Project.ID = projectID
		logger.Debug("Applied project ID from flag", "project_id", projectID)
	}

	// Apply repository
	if githubRepo != "" {
		parts := strings.Split(githubRepo, "/")
		if len(parts) != 2 {
			return errors.NewValidationError(
				"Invalid repository format",
				"Repository should be in format 'owner/name'",
				"Example: --repo myusername/my-repository")
		}
		cfg.Repository.Owner = parts[0]
		cfg.Repository.Name = parts[1]
		logger.Debug("Applied repository from flag", "owner", parts[0], "name", parts[1])
	}

	// Apply service account name
	if serviceAccount != "" {
		cfg.ServiceAccount.Name = serviceAccount
		logger.Debug("Applied service account name from flag", "name", serviceAccount)
	}

	// Apply Cloud Run service name
	if cloudRunService != "" {
		cfg.CloudRun.ServiceName = cloudRunService
		logger.Debug("Applied Cloud Run service name from flag", "name", cloudRunService)
	}

	// Apply Cloud Run region
	if cloudRunRegion != "us-central1" || cfg.CloudRun.Region == "" {
		cfg.CloudRun.Region = cloudRunRegion
		logger.Debug("Applied Cloud Run region from flag", "region", cloudRunRegion)
	}

	// Apply artifact registry
	if artifactRegistry != "" {
		cfg.CloudRun.Registry = artifactRegistry
		logger.Debug("Applied artifact registry from flag", "registry", artifactRegistry)
	}

	// Apply project number
	if projectNumber != "" {
		cfg.Project.Number = projectNumber
		logger.Debug("Applied project number from flag", "project_number", projectNumber)
	}

	// Apply project region
	if projectRegion != "" {
		cfg.Project.Region = projectRegion
		logger.Debug("Applied project region from flag", "project_region", projectRegion)
	}

	// Apply repository owner
	if repoOwner != "" {
		cfg.Repository.Owner = repoOwner
		logger.Debug("Applied repository owner from flag", "owner", repoOwner)
	}

	// Apply repository name
	if repoName != "" {
		cfg.Repository.Name = repoName
		logger.Debug("Applied repository name from flag", "name", repoName)
	}

	// Apply repository reference
	if repoRef != "" {
		cfg.Repository.Ref = repoRef
		logger.Debug("Applied repository reference from flag", "ref", repoRef)
	}

	// Apply repository branches
	if len(repoBranches) > 0 {
		cfg.Repository.Branches = repoBranches
		logger.Debug("Applied repository branches from flag", "branches", strings.Join(repoBranches, ", "))
	}

	// Apply repository tags
	if len(repoTags) > 0 {
		cfg.Repository.Tags = repoTags
		logger.Debug("Applied repository tags from flag", "tags", strings.Join(repoTags, ", "))
	}

	// Apply repository pull request
	if repoPR {
		cfg.Repository.PullRequest = repoPR
		logger.Debug("Applied repository pull request from flag", "pull_request", repoPR)
	}

	// Apply service account display name
	if saDisplayName != "" {
		cfg.ServiceAccount.DisplayName = saDisplayName
		logger.Debug("Applied service account display name from flag", "display_name", saDisplayName)
	}

	// Apply service account description
	if saDescription != "" {
		cfg.ServiceAccount.Description = saDescription
		logger.Debug("Applied service account description from flag", "description", saDescription)
	}

	// Apply service account roles
	if len(saRoles) > 0 {
		cfg.ServiceAccount.Roles = saRoles
		logger.Debug("Applied service account roles from flag", "roles", strings.Join(saRoles, ", "))
	}

	// Apply workload identity pool name
	if wiPoolName != "" {
		cfg.WorkloadIdentity.PoolName = wiPoolName
		logger.Debug("Applied workload identity pool name from flag", "pool_name", wiPoolName)
	}

	// Apply workload identity pool ID
	if wiPoolID != "" {
		cfg.WorkloadIdentity.PoolID = wiPoolID
		logger.Debug("Applied workload identity pool ID from flag", "pool_id", wiPoolID)
	}

	// Apply workload identity provider name
	if wiProviderName != "" {
		cfg.WorkloadIdentity.ProviderName = wiProviderName
		logger.Debug("Applied workload identity provider name from flag", "provider_name", wiProviderName)
	}

	// Apply workload identity provider ID
	if wiProviderID != "" {
		cfg.WorkloadIdentity.ProviderID = wiProviderID
		logger.Debug("Applied workload identity provider ID from flag", "provider_id", wiProviderID)
	}

	// Apply workload identity conditions
	if len(wiConditions) > 0 {
		cfg.WorkloadIdentity.Conditions = wiConditions
		logger.Debug("Applied workload identity conditions from flag", "conditions", strings.Join(wiConditions, ", "))
	}

	// Apply Cloud Run image
	if crImage != "" {
		cfg.CloudRun.Image = crImage
		logger.Debug("Applied Cloud Run image from flag", "image", crImage)
	}

	// Apply Cloud Run port
	if crPort != 0 {
		cfg.CloudRun.Port = crPort
		logger.Debug("Applied Cloud Run port from flag", "port", crPort)
	}

	// Apply Cloud Run CPU limit
	if crCPULimit != "" {
		cfg.CloudRun.CPULimit = crCPULimit
		logger.Debug("Applied Cloud Run CPU limit from flag", "cpu_limit", crCPULimit)
	}

	// Apply Cloud Run memory limit
	if crMemoryLimit != "" {
		cfg.CloudRun.MemoryLimit = crMemoryLimit
		logger.Debug("Applied Cloud Run memory limit from flag", "memory_limit", crMemoryLimit)
	}

	// Apply Cloud Run maximum instances
	if crMaxInstances != 0 {
		cfg.CloudRun.MaxInstances = crMaxInstances
		logger.Debug("Applied Cloud Run maximum instances from flag", "max_instances", crMaxInstances)
	}

	// Apply Cloud Run minimum instances
	if crMinInstances != 0 {
		cfg.CloudRun.MinInstances = crMinInstances
		logger.Debug("Applied Cloud Run minimum instances from flag", "min_instances", crMinInstances)
	}

	// Apply workflow name
	if wfName != "" {
		cfg.Workflow.Name = wfName
		logger.Debug("Applied workflow name from flag", "name", wfName)
	}

	// Apply workflow filename
	if wfFilename != "" {
		cfg.Workflow.Filename = wfFilename
		logger.Debug("Applied workflow filename from flag", "filename", wfFilename)
	}

	// Apply workflow path
	if wfPath != "" {
		cfg.Workflow.Path = wfPath
		logger.Debug("Applied workflow path from flag", "path", wfPath)
	}

	// Apply workflow triggers
	if len(wfTriggers) > 0 {
		cfg.Workflow.Triggers = wfTriggers
		logger.Debug("Applied workflow triggers from flag", "triggers", strings.Join(wfTriggers, ", "))
	}

	// Apply workflow environment
	if wfEnvironment != "" {
		cfg.Workflow.Environment = wfEnvironment
		logger.Debug("Applied workflow environment from flag", "environment", wfEnvironment)
	}

	// Apply workflow Docker image
	if wfDockerImage != "" {
		cfg.Workflow.DockerImage = wfDockerImage
		logger.Debug("Applied workflow Docker image from flag", "docker_image", wfDockerImage)
	}

	// Apply dry run
	if dryRun {
		cfg.Advanced.DryRun = dryRun
		logger.Debug("Applied dry run from flag", "dry_run", dryRun)
	}

	// Apply skip validation
	if skipValidation {
		cfg.Advanced.SkipValidation = skipValidation
		logger.Debug("Applied skip validation from flag", "skip_validation", skipValidation)
	}

	// Apply force update
	if forceUpdate {
		cfg.Advanced.ForceUpdate = forceUpdate
		logger.Debug("Applied force update from flag", "force_update", forceUpdate)
	}

	// Apply backup existing
	if backupExisting {
		cfg.Advanced.BackupExisting = backupExisting
		logger.Debug("Applied backup existing from flag", "backup_existing", backupExisting)
	}

	// Apply cleanup on failure
	if cleanupOnFailure {
		cfg.Advanced.CleanupOnFailure = cleanupOnFailure
		logger.Debug("Applied cleanup on failure from flag", "cleanup_on_failure", cleanupOnFailure)
	}

	// Apply enable APIs
	if len(enableAPIs) > 0 {
		cfg.Advanced.EnableAPIs = enableAPIs
		logger.Debug("Applied enable APIs from flag", "enable_apis", strings.Join(enableAPIs, ", "))
	}

	// Apply timeout
	if timeout != "" {
		cfg.Advanced.Timeout = timeout
		logger.Debug("Applied timeout from flag", "timeout", timeout)
	}

	// Apply defaults after flag overrides
	cfg.SetDefaults()

	return nil
}

// formatValidationErrors formats validation errors into a user-friendly error
func formatValidationErrors(result *config.ValidationResult) error {
	var errorMessages []string
	for _, valErr := range result.Errors {
		errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", valErr.Field, valErr.Message))
	}

	return errors.NewValidationError(
		"Configuration validation failed",
		errorMessages...)
}

// displayConfigSummary displays a summary of the current configuration
func displayConfigSummary(cfg *config.Config) {
	logger := logging.WithField("function", "displayConfigSummary")

	fmt.Println("\n📋 Configuration Summary:")
	fmt.Println("========================")

	// Project information
	fmt.Printf("🏗️  Project ID: %s\n", cfg.Project.ID)
	if cfg.Project.Region != "" {
		fmt.Printf("🌍 Project Region: %s\n", cfg.Project.Region)
	}

	// Repository information
	fmt.Printf("📚 Repository: %s\n", cfg.GetRepoFullName())
	if len(cfg.Repository.Branches) > 0 {
		fmt.Printf("🌿 Branches: %s\n", strings.Join(cfg.Repository.Branches, ", "))
	}

	// Service Account information
	fmt.Printf("👤 Service Account: %s\n", cfg.ServiceAccount.Name)
	fmt.Printf("📧 Service Account Email: %s\n", cfg.GetServiceAccountEmail())
	if len(cfg.ServiceAccount.Roles) > 0 {
		fmt.Printf("🔐 IAM Roles: %s\n", strings.Join(cfg.ServiceAccount.Roles, ", "))
	}

	// Workload Identity information
	fmt.Printf("🔗 Workload Identity Pool: %s\n", cfg.WorkloadIdentity.PoolID)
	fmt.Printf("🔌 Workload Identity Provider: %s\n", cfg.WorkloadIdentity.ProviderID)

	// Cloud Run information (if configured)
	if cfg.CloudRun.ServiceName != "" {
		fmt.Printf("☁️  Cloud Run Service: %s\n", cfg.CloudRun.ServiceName)
		fmt.Printf("🌍 Cloud Run Region: %s\n", cfg.CloudRun.Region)
		if cfg.GetCloudRunURL() != "" {
			fmt.Printf("🌐 Cloud Run URL: %s\n", cfg.GetCloudRunURL())
		}
	}

	// Workflow information
	fmt.Printf("⚡ Workflow File: %s\n", cfg.GetWorkflowFilePath())
	if len(cfg.Workflow.Triggers) > 0 {
		fmt.Printf("🎯 Triggers: %s\n", strings.Join(cfg.Workflow.Triggers, ", "))
	}

	// Advanced settings
	if cfg.Advanced.DryRun {
		fmt.Println("🧪 Dry Run: Enabled")
	}
	if cfg.Advanced.Timeout != "" {
		fmt.Printf("⏱️  Timeout: %s\n", cfg.Advanced.Timeout)
	}

	fmt.Printf("🔧 Interactive Mode: %t\n", interactive)
	fmt.Println()

	// Log the configuration for debugging
	logger.Info("Configuration summary displayed",
		"project", cfg.Project.ID,
		"repository", cfg.GetRepoFullName(),
		"service_account", cfg.ServiceAccount.Name,
		"interactive", interactive)
}

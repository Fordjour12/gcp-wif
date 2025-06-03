package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Fordjour12/gcp-wif/internal/config"
	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/gcp"
	"github.com/Fordjour12/gcp-wif/internal/github"
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

	// Environment and secrets flags
	envNames          []string
	envVariables      []string // format: "env:key=value"
	envSecrets        []string // format: "env:key=SECRET_NAME"
	envProtection     []string // format: "env:reviewers=@team,wait=5"
	globalSecrets     []string // format: "key=SECRET_NAME"
	buildSecrets      []string // format: "key=SECRET_NAME"
	createStandardEnv bool

	// Health check flags
	healthChecks        []string // format: "name:url:method:timeout:retries:wait_time:healthy_code"
	createDefaultHealth bool
	healthCheckTimeout  string
	healthCheckRetries  int
	healthCheckWaitTime string

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
- Environments: --env-names, --env-variables, --env-secrets, --env-protection, --create-standard-env
- Secrets: --global-secrets, --build-secrets
- Health Checks: --health-checks, --create-default-health, --health-check-timeout, --health-check-retries, --health-check-wait-time
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

	// Environment and secrets flags
	setupCmd.Flags().StringSliceVar(&envNames, "env-names", []string{}, "Environment names to create")
	setupCmd.Flags().StringSliceVar(&envVariables, "env-variables", []string{}, "Environment variables (format: env:key=value)")
	setupCmd.Flags().StringSliceVar(&envSecrets, "env-secrets", []string{}, "Environment secrets (format: env:key=SECRET_NAME)")
	setupCmd.Flags().StringSliceVar(&envProtection, "env-protection", []string{}, "Environment protection rules (format: env:reviewers=@team,wait=5)")
	setupCmd.Flags().StringSliceVar(&globalSecrets, "global-secrets", []string{}, "Global workflow secrets (format: key=SECRET_NAME)")
	setupCmd.Flags().StringSliceVar(&buildSecrets, "build-secrets", []string{}, "Build-time secrets (format: key=SECRET_NAME)")
	setupCmd.Flags().BoolVar(&createStandardEnv, "create-standard-env", false, "Create standard environments (dev, staging, production)")

	// Health check flags
	setupCmd.Flags().StringSliceVar(&healthChecks, "health-checks", []string{}, "Health check configuration (format: name:url:method:timeout:retries:wait_time:healthy_code)")
	setupCmd.Flags().BoolVar(&createDefaultHealth, "create-default-health", false, "Create default health checks")
	setupCmd.Flags().StringVar(&healthCheckTimeout, "health-check-timeout", "", "Health check timeout")
	setupCmd.Flags().IntVar(&healthCheckRetries, "health-check-retries", 0, "Health check retries")
	setupCmd.Flags().StringVar(&healthCheckWaitTime, "health-check-wait-time", "", "Health check wait time")

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

	// Handle dry-run mode
	if cfg.Advanced.DryRun {
		fmt.Println("\n🔍 Dry run mode - showing what would be executed without making changes")
		return runDryRunMode(cfg)
	}

	// Prompt for confirmation unless in non-interactive mode
	if interactive {
		fmt.Println("\n❓ Do you want to proceed with the setup? (y/N)")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Setup cancelled by user")
			return nil
		}
	}

	// Start orchestrated setup process
	logger.Info("Starting orchestrated WIF setup")
	fmt.Println("\n🔧 Starting orchestrated Workload Identity Federation setup...")

	if err := runOrchestration(cfg); err != nil {
		if cfg.Advanced.CleanupOnFailure {
			logger.Warn("Setup failed, attempting cleanup", "error", err)
			fmt.Println("\n🧹 Setup failed, running cleanup...")
			if cleanupErr := runCleanup(cfg); cleanupErr != nil {
				logger.Error("Cleanup failed", "error", cleanupErr)
				fmt.Printf("❌ Cleanup failed: %v\n", cleanupErr)
			} else {
				fmt.Println("✅ Cleanup completed successfully")
			}
		}
		return err
	}

	logger.Info("Workload Identity Federation setup completed successfully")
	fmt.Println("\n🎉 Workload Identity Federation setup completed successfully!")
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
		logger.Debug("Applying workflow triggers from flag", "triggers_input", strings.Join(wfTriggers, ", "))
		// Initialize Triggers struct if it's zero, but preserve defaults from DefaultWorkflowConfig if already set.
		// If flags are provided, they will selectively enable parts of the Triggers struct.
		// We don't zero out cfg.Workflow.Triggers here because DefaultConfig already called DefaultWorkflowConfig.
		// Flags should ADD or OVERRIDE specific parts of triggers.

		// Create a new Triggers struct to apply flag changes, then merge if needed or replace.
		// For simplicity with current StringSlice flag, flags will enable trigger types.
		// More granular control (branches for push from flag) would require more complex flag parsing.
		tempTriggers := cfg.Workflow.Triggers // Start with existing (default or loaded) triggers

		for _, t := range wfTriggers {
			switch strings.ToLower(t) {
			case "push":
				tempTriggers.Push.Enabled = true
				// If branches for push are not set by other means (e.g. defaults), set some basic ones.
				if len(tempTriggers.Push.Branches) == 0 {
					tempTriggers.Push.Branches = []string{"main", "master"}
				}
			case "pull_request", "pr":
				tempTriggers.PullRequest.Enabled = true
				if len(tempTriggers.PullRequest.Branches) == 0 {
					tempTriggers.PullRequest.Branches = []string{"main", "master"}
				}
				if len(tempTriggers.PullRequest.Types) == 0 {
					tempTriggers.PullRequest.Types = []string{"opened", "synchronize", "reopened"}
				}
			case "manual", "workflow_dispatch":
				tempTriggers.Manual = true
			case "release":
				tempTriggers.Release = true
			// Note: Schedule trigger is complex (cron string) and harder to set with a simple string slice flag.
			// It would typically be set via config file or a dedicated flag.
			default:
				logger.Warn("Unknown workflow trigger type from flag", "trigger", t)
			}
		}
		cfg.Workflow.Triggers = tempTriggers // Assign the modified triggers back
	}

	// Apply workflow environment name (maps to Advanced.Environments)
	if wfEnvironment != "" {
		if cfg.Workflow.Advanced.Environments == nil {
			cfg.Workflow.Advanced.Environments = make(map[string]github.Environment)
		}
		// If the environment doesn't exist, create a basic entry.
		// More detailed environment config would need more flags or config file.
		if _, exists := cfg.Workflow.Advanced.Environments[wfEnvironment]; !exists {
			cfg.Workflow.Advanced.Environments[wfEnvironment] = github.Environment{Name: wfEnvironment}
		}
		logger.Debug("Applied workflow environment from flag", "environment_name", wfEnvironment)
	}

	// Apply workflow Docker image name (maps to Workflow.ServiceName, which influences image name)
	// The actual image URI is constructed by the template using ServiceName, ProjectID, and Region.
	// The 'Registry' field in WorkflowConfig specifies the Docker registry host (e.g., gcr.io, pkg.dev).
	if wfDockerImage != "" {
		// If wfDockerImage looks like a full path (e.g., gcr.io/my-project/my-image), it's more complex.
		// For now, assume wfDockerImage is intended to be the *service name* or *base image name*.
		cfg.Workflow.ServiceName = wfDockerImage
		logger.Debug("Applied workflow Docker image (as ServiceName) from flag", "service_name_for_image", wfDockerImage)
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

	// Apply environment and secrets configurations
	if err := applyEnvironmentFlags(cfg); err != nil {
		return err
	}

	// Apply health check configurations
	if err := applyHealthCheckFlags(cfg); err != nil {
		return err
	}

	// Apply defaults after flag overrides
	cfg.SetDefaults()

	return nil
}

// applyEnvironmentFlags applies environment and secrets related flags to the configuration
func applyEnvironmentFlags(cfg *config.Config) error {
	logger := logging.WithField("function", "applyEnvironmentFlags")

	// Create standard environments if requested
	if createStandardEnv {
		cfg.Workflow.CreateStandardEnvironments()
		logger.Debug("Created standard environments (development, staging, production)")
	}

	// Add custom environment names
	for _, envName := range envNames {
		if strings.TrimSpace(envName) != "" {
			cfg.Workflow.AddEnvironment(envName, github.Environment{Name: envName})
			logger.Debug("Added environment from flag", "environment", envName)
		}
	}

	// Parse and apply environment variables (format: "env:key=value")
	for _, envVar := range envVariables {
		if err := parseAndApplyEnvironmentVariable(cfg, envVar); err != nil {
			return fmt.Errorf("invalid environment variable format '%s': %w", envVar, err)
		}
	}

	// Parse and apply environment secrets (format: "env:key=SECRET_NAME")
	for _, envSecret := range envSecrets {
		if err := parseAndApplyEnvironmentSecret(cfg, envSecret); err != nil {
			return fmt.Errorf("invalid environment secret format '%s': %w", envSecret, err)
		}
	}

	// Parse and apply environment protection rules (format: "env:reviewers=@team,wait=5")
	for _, envProt := range envProtection {
		if err := parseAndApplyEnvironmentProtection(cfg, envProt); err != nil {
			return fmt.Errorf("invalid environment protection format '%s': %w", envProt, err)
		}
	}

	// Parse and apply global secrets (format: "key=SECRET_NAME")
	for _, globalSecret := range globalSecrets {
		if err := parseAndApplyGlobalSecret(cfg, globalSecret); err != nil {
			return fmt.Errorf("invalid global secret format '%s': %w", globalSecret, err)
		}
	}

	// Parse and apply build secrets (format: "key=SECRET_NAME")
	for _, buildSecret := range buildSecrets {
		if err := parseAndApplyBuildSecret(cfg, buildSecret); err != nil {
			return fmt.Errorf("invalid build secret format '%s': %w", buildSecret, err)
		}
	}

	logger.Debug("Applied all environment and secrets flags successfully")
	return nil
}

// parseAndApplyEnvironmentVariable parses and applies environment variable (format: "env:key=value")
func parseAndApplyEnvironmentVariable(cfg *config.Config, envVar string) error {
	parts := strings.SplitN(envVar, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected format 'env:key=value', got '%s'", envVar)
	}

	envName := strings.TrimSpace(parts[0])
	keyValue := strings.TrimSpace(parts[1])

	kvParts := strings.SplitN(keyValue, "=", 2)
	if len(kvParts) != 2 {
		return fmt.Errorf("expected format 'env:key=value', got '%s'", envVar)
	}

	key := strings.TrimSpace(kvParts[0])
	value := strings.TrimSpace(kvParts[1])

	if envName == "" || key == "" {
		return fmt.Errorf("environment name and key cannot be empty")
	}

	return cfg.Workflow.AddEnvironmentVariable(envName, key, value)
}

// parseAndApplyEnvironmentSecret parses and applies environment secret (format: "env:key=SECRET_NAME")
func parseAndApplyEnvironmentSecret(cfg *config.Config, envSecret string) error {
	parts := strings.SplitN(envSecret, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected format 'env:key=SECRET_NAME', got '%s'", envSecret)
	}

	envName := strings.TrimSpace(parts[0])
	keyValue := strings.TrimSpace(parts[1])

	kvParts := strings.SplitN(keyValue, "=", 2)
	if len(kvParts) != 2 {
		return fmt.Errorf("expected format 'env:key=SECRET_NAME', got '%s'", envSecret)
	}

	key := strings.TrimSpace(kvParts[0])
	secretRef := strings.TrimSpace(kvParts[1])

	if envName == "" || key == "" || secretRef == "" {
		return fmt.Errorf("environment name, key, and secret reference cannot be empty")
	}

	return cfg.Workflow.AddEnvironmentSecret(envName, key, secretRef)
}

// parseAndApplyEnvironmentProtection parses and applies environment protection (format: "env:reviewers=@team,wait=5")
func parseAndApplyEnvironmentProtection(cfg *config.Config, envProt string) error {
	parts := strings.SplitN(envProt, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected format 'env:protection_rules', got '%s'", envProt)
	}

	envName := strings.TrimSpace(parts[0])
	protectionRules := strings.TrimSpace(parts[1])

	if envName == "" {
		return fmt.Errorf("environment name cannot be empty")
	}

	// Get or create environment
	env, exists := cfg.Workflow.GetEnvironment(envName)
	if !exists {
		env = github.Environment{Name: envName}
	}

	// Parse protection rules (format: "reviewers=@team1,@team2,wait=5,prevent_self=true")
	rules := strings.Split(protectionRules, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		if strings.HasPrefix(rule, "reviewers=") {
			reviewersStr := strings.TrimPrefix(rule, "reviewers=")
			reviewers := strings.Split(reviewersStr, " ")
			for i, reviewer := range reviewers {
				reviewers[i] = strings.TrimSpace(reviewer)
			}
			env.Protection.RequiredReviewers = reviewers
		} else if strings.HasPrefix(rule, "wait=") {
			waitStr := strings.TrimPrefix(rule, "wait=")
			var waitTime int
			if _, err := fmt.Sscanf(waitStr, "%d", &waitTime); err != nil {
				return fmt.Errorf("invalid wait time '%s': %w", waitStr, err)
			}
			env.Protection.WaitTimer = waitTime
		} else if rule == "prevent_self=true" {
			env.Protection.PreventSelfReview = true
		} else if rule == "prevent_self=false" {
			env.Protection.PreventSelfReview = false
		} else {
			return fmt.Errorf("unknown protection rule '%s'", rule)
		}
	}

	cfg.Workflow.AddEnvironment(envName, env)
	return nil
}

// parseAndApplyGlobalSecret parses and applies global secret (format: "key=SECRET_NAME")
func parseAndApplyGlobalSecret(cfg *config.Config, globalSecret string) error {
	parts := strings.SplitN(globalSecret, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected format 'key=SECRET_NAME', got '%s'", globalSecret)
	}

	key := strings.TrimSpace(parts[0])
	secretRef := strings.TrimSpace(parts[1])

	if key == "" || secretRef == "" {
		return fmt.Errorf("key and secret reference cannot be empty")
	}

	cfg.Workflow.AddGlobalSecret(key, secretRef)
	return nil
}

// parseAndApplyBuildSecret parses and applies build secret (format: "key=SECRET_NAME")
func parseAndApplyBuildSecret(cfg *config.Config, buildSecret string) error {
	parts := strings.SplitN(buildSecret, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected format 'key=SECRET_NAME', got '%s'", buildSecret)
	}

	key := strings.TrimSpace(parts[0])
	secretRef := strings.TrimSpace(parts[1])

	if key == "" || secretRef == "" {
		return fmt.Errorf("key and secret reference cannot be empty")
	}

	cfg.Workflow.AddBuildSecret(key, secretRef)
	return nil
}

// applyHealthCheckFlags applies health check related flags to the configuration
func applyHealthCheckFlags(cfg *config.Config) error {
	logger := logging.WithField("function", "applyHealthCheckFlags")

	// Apply health check configurations
	for _, healthCheck := range healthChecks {
		if err := parseAndApplyHealthCheck(cfg, healthCheck); err != nil {
			return fmt.Errorf("invalid health check format '%s': %w", healthCheck, err)
		}
	}

	// Apply create default health checks
	if createDefaultHealth {
		cfg.Workflow.CreateDefaultHealthChecks()
		logger.Debug("Created default health checks")
	}

	// Apply health check timeout (create basic health check with timeout setting)
	if healthCheckTimeout != "" {
		// Apply timeout to all existing health checks or create a basic one
		if len(cfg.Workflow.Advanced.HealthChecks) == 0 {
			cfg.Workflow.AddHealthCheck(github.HealthCheck{
				Name:        "basic-health",
				URL:         "/health",
				Method:      "GET",
				Timeout:     healthCheckTimeout,
				Retries:     3,
				WaitTime:    "5s",
				HealthyCode: 200,
			})
		} else {
			// Update timeout for existing health checks
			for i := range cfg.Workflow.Advanced.HealthChecks {
				cfg.Workflow.Advanced.HealthChecks[i].Timeout = healthCheckTimeout
			}
		}
		logger.Debug("Applied health check timeout from flag", "timeout", healthCheckTimeout)
	}

	// Apply health check retries
	if healthCheckRetries != 0 {
		// Apply retries to all existing health checks or create a basic one
		if len(cfg.Workflow.Advanced.HealthChecks) == 0 {
			cfg.Workflow.AddHealthCheck(github.HealthCheck{
				Name:        "basic-health",
				URL:         "/health",
				Method:      "GET",
				Timeout:     "10s",
				Retries:     healthCheckRetries,
				WaitTime:    "5s",
				HealthyCode: 200,
			})
		} else {
			// Update retries for existing health checks
			for i := range cfg.Workflow.Advanced.HealthChecks {
				cfg.Workflow.Advanced.HealthChecks[i].Retries = healthCheckRetries
			}
		}
		logger.Debug("Applied health check retries from flag", "retries", healthCheckRetries)
	}

	// Apply health check wait time
	if healthCheckWaitTime != "" {
		// Apply wait time to all existing health checks or create a basic one
		if len(cfg.Workflow.Advanced.HealthChecks) == 0 {
			cfg.Workflow.AddHealthCheck(github.HealthCheck{
				Name:        "basic-health",
				URL:         "/health",
				Method:      "GET",
				Timeout:     "10s",
				Retries:     3,
				WaitTime:    healthCheckWaitTime,
				HealthyCode: 200,
			})
		} else {
			// Update wait time for existing health checks
			for i := range cfg.Workflow.Advanced.HealthChecks {
				cfg.Workflow.Advanced.HealthChecks[i].WaitTime = healthCheckWaitTime
			}
		}
		logger.Debug("Applied health check wait time from flag", "wait_time", healthCheckWaitTime)
	}

	logger.Debug("Applied all health check flags successfully")
	return nil
}

// parseAndApplyHealthCheck parses and applies health check configuration (format: "name:url:method:timeout:retries:wait_time:healthy_code")
func parseAndApplyHealthCheck(cfg *config.Config, healthCheck string) error {
	parts := strings.SplitN(healthCheck, ":", 7)
	if len(parts) != 7 {
		return fmt.Errorf("expected format 'name:url:method:timeout:retries:wait_time:healthy_code', got '%s'", healthCheck)
	}

	name := strings.TrimSpace(parts[0])
	url := strings.TrimSpace(parts[1])
	method := strings.TrimSpace(parts[2])
	timeout := strings.TrimSpace(parts[3])
	retries := strings.TrimSpace(parts[4])
	waitTime := strings.TrimSpace(parts[5])
	healthyCode := strings.TrimSpace(parts[6])

	if name == "" || url == "" || method == "" || timeout == "" || retries == "" || waitTime == "" || healthyCode == "" {
		return fmt.Errorf("health check name, url, method, timeout, retries, wait_time, and healthy_code cannot be empty")
	}

	var retriesCount int
	if retries != "" {
		var err error
		retriesCount, err = strconv.Atoi(retries)
		if err != nil {
			return fmt.Errorf("invalid retries format '%s': %w", retries, err)
		}
	}

	var healthyCodeInt int
	if healthyCode != "" {
		var err error
		healthyCodeInt, err = strconv.Atoi(healthyCode)
		if err != nil {
			return fmt.Errorf("invalid healthy_code format '%s': %w", healthyCode, err)
		}
	}

	healthCheckConfig := github.HealthCheck{
		Name:        name,
		URL:         url,
		Method:      method,
		Timeout:     timeout, // Keep as string
		Retries:     retriesCount,
		WaitTime:    waitTime, // Keep as string
		HealthyCode: healthyCodeInt,
	}

	cfg.Workflow.AddHealthCheck(healthCheckConfig)
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

	// Format Triggers for display
	var triggerDisplayStrings []string
	if cfg.Workflow.Triggers.Push.Enabled {
		pushStr := "Push"
		if len(cfg.Workflow.Triggers.Push.Branches) > 0 {
			pushStr += fmt.Sprintf(" (branches: %s)", strings.Join(cfg.Workflow.Triggers.Push.Branches, ", "))
		}
		triggerDisplayStrings = append(triggerDisplayStrings, pushStr)
	}
	if cfg.Workflow.Triggers.PullRequest.Enabled {
		prStr := "Pull Request"
		if len(cfg.Workflow.Triggers.PullRequest.Branches) > 0 {
			prStr += fmt.Sprintf(" (branches: %s)", strings.Join(cfg.Workflow.Triggers.PullRequest.Branches, ", "))
		}
		if len(cfg.Workflow.Triggers.PullRequest.Types) > 0 {
			prStr += fmt.Sprintf(" (types: %s)", strings.Join(cfg.Workflow.Triggers.PullRequest.Types, ", "))
		}
		triggerDisplayStrings = append(triggerDisplayStrings, prStr)
	}
	if cfg.Workflow.Triggers.Manual {
		triggerDisplayStrings = append(triggerDisplayStrings, "Manual (workflow_dispatch)")
	}
	if cfg.Workflow.Triggers.Release {
		triggerDisplayStrings = append(triggerDisplayStrings, "Release")
	}
	for _, s := range cfg.Workflow.Triggers.Schedule {
		triggerDisplayStrings = append(triggerDisplayStrings, fmt.Sprintf("Schedule (%s)", s.Cron))
	}
	if len(triggerDisplayStrings) > 0 {
		fmt.Printf("🎯 Triggers: %s\n", strings.Join(triggerDisplayStrings, "; "))
	}

	// Display configured deployment environments from Workflow.Advanced.Environments
	if len(cfg.Workflow.Advanced.Environments) > 0 {
		var envNames []string
		for name := range cfg.Workflow.Advanced.Environments {
			envNames = append(envNames, name)
		}
		fmt.Printf("🌍 Deployment Environments: %s\n", strings.Join(envNames, ", "))
	}

	// Health check information
	if len(cfg.Workflow.Advanced.HealthChecks) > 0 {
		fmt.Println("\n🔍 Health Checks:")
		fmt.Println("================")
		for _, healthCheck := range cfg.Workflow.Advanced.HealthChecks {
			fmt.Printf("🔗 Health Check: %s\n", healthCheck.Name)
			fmt.Printf("🌍 URL: %s\n", healthCheck.URL)
			fmt.Printf("🔧 Method: %s\n", healthCheck.Method)
			fmt.Printf("⏱️  Timeout: %s\n", healthCheck.Timeout)
			fmt.Printf("🔄 Retries: %d\n", healthCheck.Retries)
			fmt.Printf("⏳ Wait Time: %s\n", healthCheck.WaitTime)
			fmt.Printf("🔍 Healthy Code: %d\n", healthCheck.HealthyCode)
		}
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

// runDryRunMode shows what would be executed in the setup without making changes
func runDryRunMode(cfg *config.Config) error {
	logger := logging.WithField("function", "runDryRunMode")
	logger.Info("Running dry-run mode", "project_id", cfg.Project.ID)

	fmt.Println("\n📋 Dry Run - Operations that would be performed:")
	fmt.Println("============================================")

	// 1. Service Account Creation
	fmt.Printf("1. 🔧 Service Account Creation:\n")
	fmt.Printf("   • Name: %s\n", cfg.ServiceAccount.Name)
	fmt.Printf("   • Email: %s@%s.iam.gserviceaccount.com\n", cfg.ServiceAccount.Name, cfg.Project.ID)
	fmt.Printf("   • Display Name: %s\n", cfg.ServiceAccount.DisplayName)
	fmt.Printf("   • Description: %s\n", cfg.ServiceAccount.Description)
	fmt.Printf("   • Roles to Grant: %s\n", strings.Join(cfg.ServiceAccount.Roles, ", "))

	// 2. Workload Identity Pool
	fmt.Printf("\n2. 🏊 Workload Identity Pool Creation:\n")
	fmt.Printf("   • Pool ID: %s\n", cfg.WorkloadIdentity.PoolID)
	fmt.Printf("   • Pool Name: %s\n", cfg.WorkloadIdentity.PoolName)
	fmt.Printf("   • Repository: %s\n", cfg.GetRepoFullName())

	// 3. Workload Identity Provider
	fmt.Printf("\n3. 🔗 Workload Identity Provider Creation:\n")
	fmt.Printf("   • Provider ID: %s\n", cfg.WorkloadIdentity.ProviderID)
	fmt.Printf("   • Provider Name: %s\n", cfg.WorkloadIdentity.ProviderName)
	fmt.Printf("   • GitHub OIDC Issuer: https://token.actions.githubusercontent.com\n")
	fmt.Printf("   • Conditions: %s\n", strings.Join(cfg.WorkloadIdentity.Conditions, "; "))

	// 4. IAM Bindings
	fmt.Printf("\n4. 🔐 IAM Policy Bindings:\n")
	fmt.Printf("   • Bind service account to workload identity\n")
	fmt.Printf("   • Grant roles/iam.serviceAccountTokenCreator\n")
	fmt.Printf("   • Apply security conditions for repository: %s\n", cfg.GetRepoFullName())

	// 5. Workflow Generation
	fmt.Printf("\n5. 📄 GitHub Actions Workflow Generation:\n")
	fmt.Printf("   • Filename: %s\n", cfg.Workflow.Filename)
	fmt.Printf("   • Path: %s\n", cfg.Workflow.Path)
	fmt.Printf("   • Full Path: %s\n", cfg.Workflow.GetWorkflowFilePath())
	fmt.Printf("   • Template: %s\n", cfg.Workflow.Name)

	// 6. Summary
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   • Project: %s\n", cfg.Project.ID)
	fmt.Printf("   • Repository: %s\n", cfg.GetRepoFullName())
	fmt.Printf("   • Service Account: %s\n", cfg.GetServiceAccountEmail())
	fmt.Printf("   • WIF Provider: %s\n", cfg.GetWorkloadIdentityProviderName())

	fmt.Println("\n💡 To execute these operations, run without --dry-run flag")
	return nil
}

// runOrchestration executes the complete setup orchestration
func runOrchestration(cfg *config.Config) error {
	logger := logging.WithField("function", "runOrchestration")
	logger.Info("Starting WIF orchestration", "project_id", cfg.Project.ID)

	// Initialize GCP client
	fmt.Println("🔗 Initializing GCP client...")
	gcpClient, err := initializeGCPClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize GCP client: %w", err)
	}

	// Step 1: Create Service Account
	fmt.Println("\n1. 🔧 Creating Service Account...")
	if err := orchestrateServiceAccount(gcpClient, cfg); err != nil {
		return fmt.Errorf("service account creation failed: %w", err)
	}

	// Step 2: Create Workload Identity Pool
	fmt.Println("\n2. 🏊 Creating Workload Identity Pool...")
	if err := orchestrateWorkloadIdentityPool(gcpClient, cfg); err != nil {
		return fmt.Errorf("workload identity pool creation failed: %w", err)
	}

	// Step 3: Create Workload Identity Provider
	fmt.Println("\n3. 🔗 Creating Workload Identity Provider...")
	if err := orchestrateWorkloadIdentityProvider(gcpClient, cfg); err != nil {
		return fmt.Errorf("workload identity provider creation failed: %w", err)
	}

	// Step 4: Bind Service Account to Workload Identity
	fmt.Println("\n4. 🔐 Binding Service Account to Workload Identity...")
	if err := orchestrateServiceAccountBinding(gcpClient, cfg); err != nil {
		return fmt.Errorf("service account binding failed: %w", err)
	}

	// Step 5: Generate GitHub Actions Workflow
	fmt.Println("\n5. 📄 Generating GitHub Actions Workflow...")
	if err := orchestrateWorkflowGeneration(cfg); err != nil {
		return fmt.Errorf("workflow generation failed: %w", err)
	}

	// Step 6: Save Configuration
	fmt.Println("\n6. 💾 Saving Configuration...")
	if err := orchestrateConfigurationSave(cfg); err != nil {
		return fmt.Errorf("configuration save failed: %w", err)
	}

	// Step 7: Display Success Summary
	displaySuccessSummary(cfg)

	return nil
}

// runCleanup handles cleanup operations when setup fails
func runCleanup(cfg *config.Config) error {
	logger := logging.WithField("function", "runCleanup")
	logger.Warn("Starting cleanup operations", "project_id", cfg.Project.ID)

	// Initialize GCP client for cleanup
	gcpClient, err := initializeGCPClient(cfg)
	if err != nil {
		logger.Error("Failed to initialize GCP client for cleanup", "error", err)
		return fmt.Errorf("cleanup failed: could not initialize GCP client: %w", err)
	}

	var cleanupErrors []error

	// Clean up in reverse order of creation
	fmt.Println("🧹 Cleaning up resources...")

	// 1. Remove IAM bindings
	fmt.Println("   • Removing IAM bindings...")
	if err := cleanupServiceAccountBindings(gcpClient, cfg); err != nil {
		logger.Warn("Failed to cleanup IAM bindings", "error", err)
		cleanupErrors = append(cleanupErrors, err)
	}

	// 2. Delete Workload Identity Provider
	fmt.Println("   • Deleting Workload Identity Provider...")
	if err := cleanupWorkloadIdentityProvider(gcpClient, cfg); err != nil {
		logger.Warn("Failed to cleanup Workload Identity Provider", "error", err)
		cleanupErrors = append(cleanupErrors, err)
	}

	// 3. Delete Workload Identity Pool
	fmt.Println("   • Deleting Workload Identity Pool...")
	if err := cleanupWorkloadIdentityPool(gcpClient, cfg); err != nil {
		logger.Warn("Failed to cleanup Workload Identity Pool", "error", err)
		cleanupErrors = append(cleanupErrors, err)
	}

	// 4. Delete Service Account (optional - usually keep for safety)
	if cfg.Advanced.ForceUpdate {
		fmt.Println("   • Deleting Service Account...")
		if err := cleanupServiceAccount(gcpClient, cfg); err != nil {
			logger.Warn("Failed to cleanup Service Account", "error", err)
			cleanupErrors = append(cleanupErrors, err)
		}
	} else {
		fmt.Println("   • Keeping Service Account (use --force-update to delete)")
	}

	if len(cleanupErrors) > 0 {
		logger.Error("Cleanup completed with errors", "error_count", len(cleanupErrors))
		return fmt.Errorf("cleanup completed with %d errors (check logs for details)", len(cleanupErrors))
	}

	logger.Info("Cleanup completed successfully")
	return nil
}

// Helper Functions for Orchestration

// initializeGCPClient creates and initializes a GCP client
func initializeGCPClient(cfg *config.Config) (*gcp.Client, error) {
	ctx := context.Background()
	client, err := gcp.NewClient(ctx, cfg.Project.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP client: %w", err)
	}

	fmt.Printf("   ✅ Connected to project: %s\n", cfg.Project.ID)
	return client, nil
}

// orchestrateServiceAccount handles service account creation
func orchestrateServiceAccount(client *gcp.Client, cfg *config.Config) error {
	serviceAccountConfig := &gcp.ServiceAccountConfig{
		Name:        cfg.ServiceAccount.Name,
		DisplayName: cfg.ServiceAccount.DisplayName,
		Description: cfg.ServiceAccount.Description,
		Roles:       cfg.ServiceAccount.Roles,
		CreateNew:   cfg.ServiceAccount.CreateNew,
	}

	fmt.Printf("   • Creating service account: %s\n", cfg.ServiceAccount.Name)

	serviceAccountInfo, err := client.CreateServiceAccount(serviceAccountConfig)
	if err != nil {
		return err
	}

	fmt.Printf("   ✅ Service account created: %s\n", serviceAccountInfo.Email)

	return nil
}

// orchestrateWorkloadIdentityPool handles workload identity pool creation
func orchestrateWorkloadIdentityPool(client *gcp.Client, cfg *config.Config) error {
	workloadIdentityConfig := &gcp.WorkloadIdentityConfig{
		PoolName:   cfg.WorkloadIdentity.PoolName,
		PoolID:     cfg.WorkloadIdentity.PoolID,
		Repository: cfg.GetRepoFullName(),
		CreateNew:  true, // Always create new for orchestration
	}

	fmt.Printf("   • Creating workload identity pool: %s\n", cfg.WorkloadIdentity.PoolID)

	poolInfo, err := client.CreateWorkloadIdentityPool(workloadIdentityConfig)
	if err != nil {
		return err
	}

	fmt.Printf("   ✅ Workload identity pool created: %s\n", poolInfo.Name)
	return nil
}

// orchestrateWorkloadIdentityProvider handles workload identity provider creation
func orchestrateWorkloadIdentityProvider(client *gcp.Client, cfg *config.Config) error {
	workloadIdentityConfig := &gcp.WorkloadIdentityConfig{
		PoolID:              cfg.WorkloadIdentity.PoolID,
		ProviderName:        cfg.WorkloadIdentity.ProviderName,
		ProviderID:          cfg.WorkloadIdentity.ProviderID,
		Repository:          cfg.GetRepoFullName(),
		ServiceAccountEmail: cfg.GetServiceAccountEmail(),
		AllowedBranches:     cfg.Repository.Branches,
		AllowedTags:         cfg.Repository.Tags,
		AllowPullRequests:   cfg.Repository.PullRequest,
		CreateNew:           true, // Always create new for orchestration
	}

	fmt.Printf("   • Creating workload identity provider: %s\n", cfg.WorkloadIdentity.ProviderID)

	providerInfo, err := client.CreateWorkloadIdentityProvider(workloadIdentityConfig)
	if err != nil {
		return err
	}

	fmt.Printf("   ✅ Workload identity provider created: %s\n", providerInfo.Name)
	return nil
}

// orchestrateServiceAccountBinding handles service account to workload identity binding
func orchestrateServiceAccountBinding(client *gcp.Client, cfg *config.Config) error {
	workloadIdentityConfig := &gcp.WorkloadIdentityConfig{
		PoolID:              cfg.WorkloadIdentity.PoolID,
		ProviderID:          cfg.WorkloadIdentity.ProviderID,
		Repository:          cfg.GetRepoFullName(),
		ServiceAccountEmail: cfg.GetServiceAccountEmail(),
	}

	fmt.Printf("   • Binding service account to workload identity\n")

	if err := client.BindServiceAccountToWorkloadIdentity(workloadIdentityConfig); err != nil {
		return err
	}

	fmt.Printf("   ✅ Service account bound successfully\n")
	return nil
}

// orchestrateWorkflowGeneration handles GitHub Actions workflow generation
func orchestrateWorkflowGeneration(cfg *config.Config) error {
	fmt.Printf("   • Generating workflow: %s\n", cfg.Workflow.Filename)

	// Generate and write workflow with backup
	writeOptions := github.WriteWorkflowFileOptions{
		CreateBackup:      cfg.Advanced.BackupExisting,
		OverwriteExisting: cfg.Advanced.ForceUpdate,
		DryRun:            false,
		Validate:          true,
	}

	if err := cfg.Workflow.GenerateAndWriteWorkflowWithOptions(writeOptions); err != nil {
		return err
	}

	fmt.Printf("   ✅ Workflow generated: %s\n", cfg.Workflow.GetWorkflowFilePath())
	return nil
}

// orchestrateConfigurationSave handles saving the final configuration
func orchestrateConfigurationSave(cfg *config.Config) error {
	configFile := "wif-config.json"

	fmt.Printf("   • Saving configuration to: %s\n", configFile)

	// Save with backup if file exists
	if err := cfg.SaveWithBackup(configFile); err != nil {
		return err
	}

	fmt.Printf("   ✅ Configuration saved successfully\n")
	return nil
}

// displaySuccessSummary shows the final success summary
func displaySuccessSummary(cfg *config.Config) {
	fmt.Println("\n🎉 Setup Complete!")
	fmt.Println("=================")

	fmt.Printf("✅ Project: %s\n", cfg.Project.ID)
	fmt.Printf("✅ Repository: %s\n", cfg.GetRepoFullName())
	fmt.Printf("✅ Service Account: %s\n", cfg.GetServiceAccountEmail())
	fmt.Printf("✅ Workload Identity Provider: %s\n", cfg.GetWorkloadIdentityProviderName())
	fmt.Printf("✅ GitHub Actions Workflow: %s\n", cfg.Workflow.GetWorkflowFilePath())

	fmt.Println("\n📋 Next Steps:")
	fmt.Println("==============")
	fmt.Println("1. 📤 Commit and push the generated workflow file to your repository")
	fmt.Println("2. 🔐 Set up any required GitHub secrets in your repository settings")
	fmt.Println("3. 🚀 Push changes to trigger the workflow and test deployment")
	fmt.Println("4. 🔍 Monitor the workflow execution in the GitHub Actions tab")

	fmt.Printf("\n💡 Workflow file location: %s\n", cfg.Workflow.GetWorkflowFilePath())
	fmt.Printf("💡 Configuration saved to: wif-config.json\n")

	fmt.Println("\n🔗 Useful commands:")
	fmt.Println("   • gcp-wif workflow validate --config wif-config.json")
	fmt.Println("   • gcp-wif workflow preview --config wif-config.json")
	fmt.Printf("   • git add %s wif-config.json && git commit -m \"Add WIF workflow\"\n", cfg.Workflow.GetWorkflowFilePath())
}

// Cleanup Helper Functions

// cleanupServiceAccountBindings removes IAM bindings
func cleanupServiceAccountBindings(client *gcp.Client, cfg *config.Config) error {
	workloadIdentityConfig := &gcp.WorkloadIdentityConfig{
		PoolID:              cfg.WorkloadIdentity.PoolID,
		ProviderID:          cfg.WorkloadIdentity.ProviderID,
		Repository:          cfg.GetRepoFullName(),
		ServiceAccountEmail: cfg.GetServiceAccountEmail(),
	}

	return client.RemoveServiceAccountWorkloadIdentityBinding(workloadIdentityConfig)
}

// cleanupWorkloadIdentityProvider removes the workload identity provider
func cleanupWorkloadIdentityProvider(client *gcp.Client, cfg *config.Config) error {
	return client.DeleteWorkloadIdentityProvider(cfg.WorkloadIdentity.PoolID, cfg.WorkloadIdentity.ProviderID)
}

// cleanupWorkloadIdentityPool removes the workload identity pool
func cleanupWorkloadIdentityPool(client *gcp.Client, cfg *config.Config) error {
	return client.DeleteWorkloadIdentityPool(cfg.WorkloadIdentity.PoolID)
}

// cleanupServiceAccount removes the service account
func cleanupServiceAccount(client *gcp.Client, cfg *config.Config) error {
	return client.DeleteServiceAccount(cfg.ServiceAccount.Name)
}

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/Fordjour12/gcp-wif/internal/config"
	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/logging"
	"github.com/spf13/cobra"
)

var (
	envName         string
	envType         string
	envDescription  string
	envRegion       string
	envEnabled      bool
	envTemplate     string
	envShowDetails  bool
	envOutputFormat string
)

// envCmd represents the environment management command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage deployment environments and regions",
	Long: `Manage deployment environments and regions for your Workload Identity Federation setup.

This command provides comprehensive environment management capabilities including:
• Creating and configuring multiple deployment environments (production, staging, development)
• Managing cloud regions and geographic deployment settings
• Switching between environments for different deployment contexts
• Configuring environment-specific settings and variables

Environment Types:
• production  - Production environment with strict security and monitoring
• staging     - Pre-production environment for final testing
• development - Development environment for feature development
• testing     - Dedicated testing environment for automated tests

Examples:
  # List all environments
  gcp-wif env list

  # Create a new production environment
  gcp-wif env create --name prod --type production --region us-central1

  # Switch to a specific environment
  gcp-wif env use staging

  # Show current environment details
  gcp-wif env current

  # Configure environment variables
  gcp-wif env set-var prod DATABASE_URL "postgres://..."

  # List available regions
  gcp-wif env regions

  # Add a new region
  gcp-wif env add-region --name europe-west1 --enabled`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
	},
}

// envListCmd lists all environments
var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all deployment environments",
	Long: `List all configured deployment environments with their status and details.

Output Formats:
• table - Human-readable table format (default)
• json  - Machine-readable JSON format
• yaml  - YAML format

Examples:
  # List environments in table format
  gcp-wif env list

  # List environments with detailed information
  gcp-wif env list --details

  # List environments in JSON format
  gcp-wif env list --output json`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runEnvListCommand(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

// envCreateCmd creates a new environment
var envCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new deployment environment",
	Long: `Create a new deployment environment with specified configuration.

Environment Types:
• production  - Production environment with enhanced security
• staging     - Pre-production environment for testing
• development - Development environment for feature work
• testing     - Dedicated testing environment

Examples:
  # Create a production environment
  gcp-wif env create --name prod --type production --region us-central1

  # Create a development environment with custom template
  gcp-wif env create --name dev --type development --region us-west1 --template development

  # Create a staging environment with description
  gcp-wif env create --name staging --type staging --region europe-west1 --description "Pre-production testing"`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runEnvCreateCommand(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

// envUseCmd switches to a specific environment
var envUseCmd = &cobra.Command{
	Use:   "use [environment]",
	Short: "Switch to a specific environment",
	Long: `Switch the current active environment for all operations.

When you switch environments, all subsequent commands will use the configuration
and settings specific to that environment.

Examples:
  # Switch to production environment
  gcp-wif env use production

  # Switch to staging environment
  gcp-wif env use staging`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runEnvUseCommand(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

// envCurrentCmd shows the current environment
var envCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current active environment",
	Long: `Display information about the currently active environment.

Examples:
  # Show current environment
  gcp-wif env current

  # Show current environment with details
  gcp-wif env current --details`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runEnvCurrentCommand(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

// envRegionsCmd manages regions
var envRegionsCmd = &cobra.Command{
	Use:   "regions",
	Short: "Manage cloud regions",
	Long: `Manage Google Cloud regions for deployment environments.

Examples:
  # List all configured regions
  gcp-wif env regions

  # Add a new region
  gcp-wif env add-region --name europe-west1 --enabled

  # Show region details
  gcp-wif env regions --details`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runEnvRegionsCommand(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)

	// Add subcommands
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envCreateCmd)
	envCmd.AddCommand(envUseCmd)
	envCmd.AddCommand(envCurrentCmd)
	envCmd.AddCommand(envRegionsCmd)

	// Environment creation flags
	envCreateCmd.Flags().StringVar(&envName, "name", "", "Environment name (required)")
	envCreateCmd.Flags().StringVar(&envType, "type", "development", "Environment type: production, staging, development, testing")
	envCreateCmd.Flags().StringVar(&envDescription, "description", "", "Environment description")
	envCreateCmd.Flags().StringVar(&envRegion, "region", "us-central1", "GCP region for the environment")
	envCreateCmd.Flags().BoolVar(&envEnabled, "enabled", true, "Enable the environment")
	envCreateCmd.Flags().StringVar(&envTemplate, "template", "", "Workflow template: production, staging, development")
	envCreateCmd.MarkFlagRequired("name")

	// Output formatting flags
	envListCmd.Flags().BoolVar(&envShowDetails, "details", false, "Show detailed environment information")
	envListCmd.Flags().StringVar(&envOutputFormat, "output", "table", "Output format: table, json, yaml")
	envCurrentCmd.Flags().BoolVar(&envShowDetails, "details", false, "Show detailed environment information")
	envRegionsCmd.Flags().BoolVar(&envShowDetails, "details", false, "Show detailed region information")
}

func runEnvListCommand(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "env_list")
	logger.Info("Listing deployment environments")

	// Load configuration
	cfg, err := loadConfigWithFallback()
	if err != nil {
		return err
	}

	if len(cfg.Environments) == 0 {
		fmt.Println("No environments configured.")
		fmt.Println("\n💡 Create your first environment with:")
		fmt.Println("   gcp-wif env create --name dev --type development --region us-central1")
		return nil
	}

	switch strings.ToLower(envOutputFormat) {
	case "json":
		return outputEnvironmentsJSON(cfg)
	case "yaml":
		return outputEnvironmentsYAML(cfg)
	default:
		return outputEnvironmentsTable(cfg)
	}
}

func runEnvCreateCommand(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "env_create")
	logger.Info("Creating new environment", "name", envName, "type", envType)

	// Load configuration
	cfg, err := loadConfigWithFallback()
	if err != nil {
		return err
	}

	// Validate environment type
	validTypes := []string{config.EnvironmentProduction, config.EnvironmentStaging, config.EnvironmentDevelopment, config.EnvironmentTesting}
	if !envContains(validTypes, envType) {
		return errors.NewValidationError(
			fmt.Sprintf("Invalid environment type: %s", envType),
			fmt.Sprintf("Valid types are: %s", strings.Join(validTypes, ", ")))
	}

	// Check if environment already exists
	if cfg.Environments == nil {
		cfg.Environments = make(map[string]config.EnvironmentConfig)
	}

	if _, exists := cfg.Environments[envName]; exists {
		return errors.NewValidationError(
			fmt.Sprintf("Environment '%s' already exists", envName),
			"Use a different name or update the existing environment")
	}

	// Set default template based on environment type
	if envTemplate == "" {
		envTemplate = envType
	}

	// Create environment configuration
	env := config.EnvironmentConfig{
		Name:        envName,
		Type:        envType,
		Description: envDescription,
		Enabled:     envEnabled,
		Region:      envRegion,
		Resources: config.ResourceConfig{
			ServiceAccount: struct {
				NameSuffix string            `json:"name_suffix,omitempty" yaml:"name_suffix,omitempty"`
				Roles      []string          `json:"roles" yaml:"roles"`
				Tags       map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
			}{
				NameSuffix: envName,
				Roles:      getDefaultRolesForEnvironment(envType),
				Tags: map[string]string{
					"environment": envName,
					"type":        envType,
				},
			},
			WorkloadIdentity: struct {
				PoolSuffix     string `json:"pool_suffix,omitempty" yaml:"pool_suffix,omitempty"`
				ProviderSuffix string `json:"provider_suffix,omitempty" yaml:"provider_suffix,omitempty"`
				TTL            string `json:"ttl,omitempty" yaml:"ttl,omitempty"`
			}{
				PoolSuffix:     envName + "-pool",
				ProviderSuffix: envName + "-provider",
				TTL:            getDefaultTTLForEnvironment(envType),
			},
		},
		Security: config.EnvSecurityConfig{
			RequireApproval:      envType == config.EnvironmentProduction,
			RequireSignedCommits: envType == config.EnvironmentProduction || envType == config.EnvironmentStaging,
			RestrictBranches:     getDefaultBranchesForEnvironment(envType),
			SecretManagement:     true,
		},
		Variables: make(map[string]string),
		Workflow: config.EnvWorkflowConfig{
			Template:    envTemplate,
			Environment: envName,
			Variables: map[string]string{
				"ENVIRONMENT": envName,
				"REGION":      envRegion,
			},
		},
	}

	// Add environment to configuration
	cfg.Environments[envName] = env

	// Save configuration
	configFile := getEnvConfigFilePath()
	if err := cfg.SaveToFile(configFile); err != nil {
		return err
	}

	fmt.Printf("✅ Environment '%s' created successfully!\n", envName)
	fmt.Printf("   Type: %s\n", envType)
	fmt.Printf("   Region: %s\n", envRegion)
	fmt.Printf("   Template: %s\n", envTemplate)

	if envType == config.EnvironmentProduction {
		fmt.Println("\n🔒 Production environment created with enhanced security:")
		fmt.Println("   • Approval required for deployments")
		fmt.Println("   • Signed commits required")
		fmt.Println("   • Branch restrictions enabled")
	}

	fmt.Printf("\n💡 Switch to this environment with: gcp-wif env use %s\n", envName)

	logger.Info("Environment created successfully", "name", envName, "type", envType, "region", envRegion)
	return nil
}

func runEnvUseCommand(cmd *cobra.Command, args []string) error {
	envName := args[0]
	logger := logging.WithField("command", "env_use")
	logger.Info("Switching to environment", "name", envName)

	// Load configuration
	cfg, err := loadConfigWithFallback()
	if err != nil {
		return err
	}

	// Check if environment exists
	env, exists := cfg.Environments[envName]
	if !exists {
		return errors.NewValidationError(
			fmt.Sprintf("Environment '%s' not found", envName),
			"Create the environment first or check available environments with: gcp-wif env list")
	}

	// Set current environment
	cfg.CurrentEnv = envName

	// Save configuration
	configFile := getEnvConfigFilePath()
	if err := cfg.SaveToFile(configFile); err != nil {
		return err
	}

	fmt.Printf("✅ Switched to environment: %s\n", envName)
	fmt.Printf("   Type: %s\n", env.Type)
	fmt.Printf("   Region: %s\n", env.Region)

	if !env.Enabled {
		fmt.Println("⚠️  Warning: This environment is currently disabled")
	}

	logger.Info("Switched to environment", "name", envName, "type", env.Type)
	return nil
}

func runEnvCurrentCommand(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "env_current")
	logger.Info("Showing current environment")

	// Load configuration
	cfg, err := loadConfigWithFallback()
	if err != nil {
		return err
	}

	if cfg.CurrentEnv == "" {
		fmt.Println("No current environment set.")
		fmt.Println("\n💡 Set an environment with: gcp-wif env use <environment>")
		return nil
	}

	env, err := cfg.GetCurrentEnvironment()
	if err != nil {
		return err
	}

	fmt.Printf("📍 Current Environment: %s\n", cfg.CurrentEnv)
	fmt.Printf("   Type: %s\n", env.Type)
	fmt.Printf("   Region: %s\n", env.Region)
	fmt.Printf("   Enabled: %t\n", env.Enabled)

	if env.Description != "" {
		fmt.Printf("   Description: %s\n", env.Description)
	}

	if envShowDetails {
		fmt.Println("\n🔧 Configuration Details:")
		fmt.Printf("   Service Account Suffix: %s\n", env.Resources.ServiceAccount.NameSuffix)
		fmt.Printf("   Roles: %s\n", strings.Join(env.Resources.ServiceAccount.Roles, ", "))
		fmt.Printf("   Workflow Template: %s\n", env.Workflow.Template)

		if len(env.Variables) > 0 {
			fmt.Println("\n🔄 Environment Variables:")
			for k, v := range env.Variables {
				fmt.Printf("   %s: %s\n", k, v)
			}
		}

		fmt.Println("\n🔒 Security Settings:")
		fmt.Printf("   Require Approval: %t\n", env.Security.RequireApproval)
		fmt.Printf("   Require Signed Commits: %t\n", env.Security.RequireSignedCommits)
		if len(env.Security.RestrictBranches) > 0 {
			fmt.Printf("   Restricted Branches: %s\n", strings.Join(env.Security.RestrictBranches, ", "))
		}
	}

	logger.Info("Displayed current environment", "name", cfg.CurrentEnv)
	return nil
}

func runEnvRegionsCommand(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "env_regions")
	logger.Info("Managing regions")

	// Load configuration
	cfg, err := loadConfigWithFallback()
	if err != nil {
		return err
	}

	if len(cfg.Regions) == 0 {
		fmt.Println("No regions configured.")
		fmt.Println("\n🌍 Available GCP regions include:")
		fmt.Println("   • us-central1 (Iowa)")
		fmt.Println("   • us-east1 (South Carolina)")
		fmt.Println("   • us-west1 (Oregon)")
		fmt.Println("   • europe-west1 (Belgium)")
		fmt.Println("   • asia-east1 (Taiwan)")
		return nil
	}

	// Display regions table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "REGION\tZONE\tENABLED\tPRIORITY")
	fmt.Fprintln(w, "------\t----\t-------\t--------")

	// Sort regions by priority
	regions := cfg.Regions
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].Priority < regions[j].Priority
	})

	for _, region := range regions {
		enabled := "No"
		if region.Enabled {
			enabled = "Yes"
		}
		priority := "-"
		if region.Priority > 0 {
			priority = fmt.Sprintf("%d", region.Priority)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", region.Name, region.Zone, enabled, priority)
	}

	w.Flush()
	return nil
}

// Helper functions

func outputEnvironmentsJSON(cfg *config.Config) error {
	data, err := json.MarshalIndent(cfg.Environments, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func outputEnvironmentsYAML(cfg *config.Config) error {
	// Simple YAML-like output since we don't have a YAML dependency
	fmt.Println("environments:")
	for name, env := range cfg.Environments {
		fmt.Printf("  %s:\n", name)
		fmt.Printf("    type: %s\n", env.Type)
		fmt.Printf("    region: %s\n", env.Region)
		fmt.Printf("    enabled: %t\n", env.Enabled)
		if env.Description != "" {
			fmt.Printf("    description: %s\n", env.Description)
		}
	}
	return nil
}

func outputEnvironmentsTable(cfg *config.Config) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tREGION\tENABLED\tCURRENT")
	fmt.Fprintln(w, "----\t----\t------\t-------\t-------")

	// Sort environments by name
	names := make([]string, 0, len(cfg.Environments))
	for name := range cfg.Environments {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		env := cfg.Environments[name]
		enabled := "No"
		if env.Enabled {
			enabled = "Yes"
		}
		current := ""
		if cfg.CurrentEnv == name {
			current = "✓"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", name, env.Type, env.Region, enabled, current)
	}

	w.Flush()

	if envShowDetails {
		fmt.Println("\nDetailed Information:")
		for _, name := range names {
			env := cfg.Environments[name]
			fmt.Printf("\n📋 %s (%s)\n", name, env.Type)
			if env.Description != "" {
				fmt.Printf("   Description: %s\n", env.Description)
			}
			fmt.Printf("   Region: %s\n", env.Region)
			fmt.Printf("   Template: %s\n", env.Workflow.Template)
			fmt.Printf("   Variables: %d\n", len(env.Variables))
		}
	}

	return nil
}

func getDefaultRolesForEnvironment(envType string) []string {
	baseRoles := []string{
		"roles/run.admin",
		"roles/storage.admin",
		"roles/artifactregistry.admin",
	}

	switch envType {
	case config.EnvironmentProduction:
		return append(baseRoles, "roles/logging.logWriter", "roles/monitoring.metricWriter")
	case config.EnvironmentStaging:
		return append(baseRoles, "roles/logging.logWriter")
	default:
		return baseRoles
	}
}

func getDefaultTTLForEnvironment(envType string) string {
	switch envType {
	case config.EnvironmentProduction:
		return "3600s" // 1 hour for production
	case config.EnvironmentStaging:
		return "7200s" // 2 hours for staging
	default:
		return "14400s" // 4 hours for development/testing
	}
}

func getDefaultBranchesForEnvironment(envType string) []string {
	switch envType {
	case config.EnvironmentProduction:
		return []string{"main", "master"}
	case config.EnvironmentStaging:
		return []string{"main", "master", "staging"}
	default:
		return []string{"*"} // Allow all branches for dev/testing
	}
}

func loadConfigWithFallback() (*config.Config, error) {
	// Try to load from specified config file
	if cfgFile != "" {
		return config.LoadFromFile(cfgFile)
	}

	// Try auto-discovery
	cfg, err := config.LoadFromFileWithDiscovery("")
	if err == nil {
		return cfg, nil
	}

	// Create minimal config if none found
	return config.DefaultConfig(), nil
}

// envContains checks if a slice contains a string
func envContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getEnvConfigFilePath() string {
	if cfgFile != "" {
		return cfgFile
	}

	// Try auto-discovery
	if path, err := config.AutoDiscoverConfigFile(); err == nil {
		return path
	}

	// Default fallback
	return "wif-config.json"
}

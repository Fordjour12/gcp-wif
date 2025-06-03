package cmd

import (
	"fmt"
	"os"

	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/logging"
	"github.com/spf13/cobra"
)

var (
	cfgFile  string
	verbose  bool
	logFile  string
	logLevel string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gcp-wif",
	Short: "Automate Google Cloud Workload Identity Federation setup for GitHub repositories",
	Long: `GCP WIF CLI Tool automates the setup of Google Cloud Workload Identity Federation (WIF) 
for GitHub repositories. This tool eliminates the need for manual configuration through the 
slow and unreliable Google Cloud Console web interface.

The tool automatically configures:
- Google Cloud Service Accounts with proper IAM roles
- Workload Identity Pools and Providers
- GitHub Actions workflow files with WIF authentication
- Security conditions to restrict repository access

Example usage:
  gcp-wif setup --project my-project --repo myorg/myrepo
  gcp-wif setup --config config.json
  gcp-wif setup  # Interactive mode`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, show help
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./wif-config.json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", "", "log file path (default: stderr)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	// Set up error handling
	rootCmd.SilenceErrors = true // We'll handle errors ourselves
	rootCmd.SilenceUsage = true  // Don't show usage on every error
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Initialize logging
	if err := initLogging(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logging: %v\n", err)
		os.Exit(1)
	}

	if cfgFile != "" {
		// Use config file from the flag if provided
		logging.Info("Using config file", "path", cfgFile)
	}
}

// initLogging initializes the logging framework
func initLogging() error {
	// Parse log level
	var level logging.LogLevel
	switch logLevel {
	case "debug":
		level = logging.LevelDebug
	case "info":
		level = logging.LevelInfo
	case "warn":
		level = logging.LevelWarn
	case "error":
		level = logging.LevelError
	default:
		return errors.NewValidationError(
			fmt.Sprintf("Invalid log level: %s", logLevel),
			"Valid log levels are: debug, info, warn, error")
	}

	// Create logger config
	config := &logging.LoggerConfig{
		Level:     level,
		Verbose:   verbose,
		FilePath:  logFile,
		AddSource: verbose,
	}

	// Initialize global logger
	if err := logging.InitGlobalLogger(config); err != nil {
		return errors.WrapError(err, errors.ErrorTypeInternal, "LOGGER_INIT_FAILED",
			"Failed to initialize logging framework")
	}

	return nil
}

// HandleError handles errors in a consistent way across the CLI
func HandleError(err error) {
	if err == nil {
		return
	}

	// Log the error
	logging.Error("Command failed", "error", err)

	// Print user-friendly error message
	fmt.Fprintln(os.Stderr, errors.FormatError(err))

	// Exit with appropriate code based on error type
	exitCode := 1
	switch errors.GetErrorType(err) {
	case errors.ErrorTypeValidation:
		exitCode = 2
	case errors.ErrorTypeConfiguration:
		exitCode = 3
	case errors.ErrorTypeAuthentication:
		exitCode = 4
	case errors.ErrorTypeGCP:
		exitCode = 5
	case errors.ErrorTypeGitHub:
		exitCode = 6
	case errors.ErrorTypeFileSystem:
		exitCode = 7
	default:
		exitCode = 1
	}

	os.Exit(exitCode)
}

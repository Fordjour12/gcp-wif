package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Fordjour12/gcp-wif/internal/config"
	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/logging"
	"github.com/Fordjour12/gcp-wif/internal/validation"
	"github.com/spf13/cobra"
)

var (
	// Test execution flags
	testDryRun          bool
	testVerbose         bool
	testParallel        bool
	testFailFast        bool
	testTimeout         string
	testSuitesToRun     []string
	testSkipCategories  []string
	testMinSeverity     string
	testTags            []string
	testOutputFormat    string
	testSaveResults     bool
	testCreateSnapshots bool

	// Test filtering flags
	testConfigOnly      bool
	testGCPOnly         bool
	testWorkflowOnly    bool
	testSecurityOnly    bool
	testIntegrationOnly bool
	testPerformanceOnly bool
	testResilienceOnly  bool

	// Output and reporting flags
	testOutputFile  string
	testShowSummary bool
	testShowDetails bool
	testQuiet       bool
	testJSONOutput  bool
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run comprehensive validation and testing framework",
	Long: `Run comprehensive validation and testing of your Workload Identity Federation configuration.

This command provides a complete testing framework that validates:
• Configuration schema and structure
• GCP resource access and permissions  
• GitHub repository and workflow integration
• Security best practices and compliance
• End-to-end integration and orchestration
• Performance and resilience testing

Test Categories:
• configuration - Configuration validation and schema checks
• gcp - Google Cloud Platform resources and permissions
• github - GitHub repository and workflow integration
• workflow - GitHub Actions workflow generation and validation
• integration - End-to-end integration and orchestration
• security - Security configuration and best practices
• performance - Performance and efficiency validation
• resilience - Error handling and resilience testing

Output Formats:
• summary - High-level test results summary (default)
• detailed - Detailed test execution information
• json - Machine-readable JSON output for automation
• junit - JUnit XML format for CI/CD integration

Examples:
  # Run all tests with default configuration
  gcp-wif test

  # Run only configuration and GCP tests
  gcp-wif test --suites configuration,gcp

  # Run tests with detailed output
  gcp-wif test --verbose --show-details

  # Run tests in dry-run mode
  gcp-wif test --dry-run

  # Run security and compliance tests only
  gcp-wif test --security-only --min-severity high

  # Run tests with JSON output for automation
  gcp-wif test --output-format json --output-file test-results.json

  # Run specific test categories
  gcp-wif test --skip-categories performance,resilience

  # Run with custom timeout and fail-fast
  gcp-wif test --timeout 10m --fail-fast`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTestCommand(cmd, args); err != nil {
			HandleError(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)

	// Test execution flags
	testCmd.Flags().BoolVar(&testDryRun, "dry-run", false, "Preview test execution without running tests")
	testCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, "Enable verbose test output")
	testCmd.Flags().BoolVar(&testParallel, "parallel", false, "Run tests in parallel for faster execution")
	testCmd.Flags().BoolVar(&testFailFast, "fail-fast", false, "Stop testing on first failure")
	testCmd.Flags().StringVar(&testTimeout, "timeout", "30m", "Maximum time for test execution")
	testCmd.Flags().StringSliceVar(&testSuitesToRun, "suites", nil, "Specific test suites to run (comma-separated)")
	testCmd.Flags().StringSliceVar(&testSkipCategories, "skip-categories", nil, "Test categories to skip (comma-separated)")
	testCmd.Flags().StringVar(&testMinSeverity, "min-severity", "low", "Minimum test severity to run: low, medium, high, critical")
	testCmd.Flags().StringSliceVar(&testTags, "tags", nil, "Test tags to filter by (comma-separated)")
	testCmd.Flags().StringVar(&testOutputFormat, "output-format", "summary", "Output format: summary, detailed, json, junit")
	testCmd.Flags().BoolVar(&testSaveResults, "save-results", false, "Save test results to file")
	testCmd.Flags().BoolVar(&testCreateSnapshots, "create-snapshots", false, "Create configuration snapshots before testing")

	// Test filtering flags
	testCmd.Flags().BoolVar(&testConfigOnly, "config-only", false, "Run only configuration tests")
	testCmd.Flags().BoolVar(&testGCPOnly, "gcp-only", false, "Run only GCP resource tests")
	testCmd.Flags().BoolVar(&testWorkflowOnly, "workflow-only", false, "Run only workflow tests")
	testCmd.Flags().BoolVar(&testSecurityOnly, "security-only", false, "Run only security tests")
	testCmd.Flags().BoolVar(&testIntegrationOnly, "integration-only", false, "Run only integration tests")
	testCmd.Flags().BoolVar(&testPerformanceOnly, "performance-only", false, "Run only performance tests")
	testCmd.Flags().BoolVar(&testResilienceOnly, "resilience-only", false, "Run only resilience tests")

	// Output and reporting flags
	testCmd.Flags().StringVar(&testOutputFile, "output-file", "", "File to save test results")
	testCmd.Flags().BoolVar(&testShowSummary, "show-summary", true, "Show test summary")
	testCmd.Flags().BoolVar(&testShowDetails, "show-details", false, "Show detailed test information")
	testCmd.Flags().BoolVar(&testQuiet, "quiet", false, "Suppress non-essential output")
	testCmd.Flags().BoolVar(&testJSONOutput, "json", false, "Output results in JSON format")
}

func runTestCommand(cmd *cobra.Command, args []string) error {
	logger := logging.WithField("command", "test")
	logger.Info("Starting comprehensive testing framework")

	fmt.Println("🧪 WIF Comprehensive Testing Framework")
	fmt.Println("=====================================")

	// Load configuration
	cfg, err := loadTestConfig()
	if err != nil {
		return err
	}

	// Create test framework
	testFramework := validation.NewTestFramework(cfg)

	// Build test options
	options, err := buildTestOptions()
	if err != nil {
		return err
	}

	// Apply test filtering
	if err := applyTestFiltering(options); err != nil {
		return err
	}

	// Display test plan
	if !testQuiet {
		if err := displayTestPlan(testFramework, options); err != nil {
			return err
		}
	}

	// Handle dry-run mode
	if testDryRun {
		fmt.Println("\n💡 This was a dry-run. Use --dry-run=false to execute tests.")
		return nil
	}

	// Execute tests
	logger.Info("Starting test execution")
	fmt.Println("\n🔧 Executing comprehensive tests...")

	execution, err := testFramework.ExecuteAllTests(*options)
	if err != nil {
		return err
	}

	// Display results
	if err := displayTestResults(execution, options); err != nil {
		return err
	}

	// Save results if requested
	if testSaveResults || testOutputFile != "" {
		if err := saveTestResults(execution, options); err != nil {
			logger.Warn("Failed to save test results", "error", err)
		}
	}

	// Exit with non-zero code if tests failed
	if execution.Summary.Failed > 0 {
		logger.Error("Tests failed", "failed_count", execution.Summary.Failed)
		os.Exit(1)
	}

	fmt.Println("\n🎉 All tests completed successfully!")
	logger.Info("Test execution completed successfully",
		"total", execution.Summary.Total,
		"passed", execution.Summary.Passed,
		"duration", execution.Duration)

	return nil
}

// loadTestConfig loads configuration for testing
func loadTestConfig() (*config.Config, error) {
	logger := logging.WithField("function", "loadTestConfig")

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

	// Create minimal config for testing if no file found
	logger.Debug("Creating minimal configuration for testing")
	cfg = config.DefaultConfig()

	// Set some test defaults if not provided
	if cfg.Project.ID == "" {
		cfg.Project.ID = "test-project-123"
	}
	if cfg.Repository.Owner == "" {
		cfg.Repository.Owner = "testowner"
	}
	if cfg.Repository.Name == "" {
		cfg.Repository.Name = "test-repo"
	}

	logger.Info("Created default configuration for testing")
	return cfg, nil
}

// buildTestOptions builds test options from command-line flags
func buildTestOptions() (*validation.TestOptions, error) {
	timeout, err := time.ParseDuration(testTimeout)
	if err != nil {
		return nil, errors.NewValidationError(
			fmt.Sprintf("Invalid timeout format: %s", testTimeout),
			"Use duration format like '30m', '1h', '90s'")
	}

	return &validation.TestOptions{
		DryRun:          testDryRun,
		Verbose:         testVerbose,
		Parallel:        testParallel,
		FailFast:        testFailFast,
		Timeout:         timeout,
		SuitesToRun:     testSuitesToRun,
		SkipCategories:  testSkipCategories,
		MinSeverity:     testMinSeverity,
		Tags:            testTags,
		OutputFormat:    testOutputFormat,
		SaveResults:     testSaveResults,
		CreateSnapshots: testCreateSnapshots,
	}, nil
}

// applyTestFiltering applies test category filtering based on flags
func applyTestFiltering(options *validation.TestOptions) error {
	// Handle mutually exclusive category flags
	categoryFlags := map[string]bool{
		"configuration": testConfigOnly,
		"gcp":           testGCPOnly,
		"workflow":      testWorkflowOnly,
		"security":      testSecurityOnly,
		"integration":   testIntegrationOnly,
		"performance":   testPerformanceOnly,
		"resilience":    testResilienceOnly,
	}

	// Count active category flags
	activeCategories := []string{}
	for category, active := range categoryFlags {
		if active {
			activeCategories = append(activeCategories, category)
		}
	}

	// If specific categories are selected, run only those
	if len(activeCategories) > 0 {
		suitesToRun := []string{}
		for _, category := range activeCategories {
			switch category {
			case "configuration":
				suitesToRun = append(suitesToRun, "Configuration")
			case "gcp":
				suitesToRun = append(suitesToRun, "GCP")
			case "workflow":
				suitesToRun = append(suitesToRun, "Workflow")
			case "security":
				suitesToRun = append(suitesToRun, "Security")
			case "integration":
				suitesToRun = append(suitesToRun, "Integration")
			case "performance":
				suitesToRun = append(suitesToRun, "Performance")
			case "resilience":
				suitesToRun = append(suitesToRun, "Resilience")
			}
		}
		options.SuitesToRun = suitesToRun
	}

	return nil
}

// displayTestPlan shows what tests will be executed
func displayTestPlan(testFramework *validation.TestFramework, options *validation.TestOptions) error {
	fmt.Println("\n📋 Test Execution Plan:")
	fmt.Println("========================")

	allSuites := testFramework.GetAllTestSuites()

	// Filter suites based on options
	suitesToRun := allSuites
	if len(options.SuitesToRun) > 0 {
		filteredSuites := []validation.TestSuite{}
		for _, suite := range allSuites {
			for _, runSuite := range options.SuitesToRun {
				if suite.Name == runSuite {
					filteredSuites = append(filteredSuites, suite)
					break
				}
			}
		}
		suitesToRun = filteredSuites
	}

	fmt.Printf("📊 Test Suites: %d\n", len(suitesToRun))

	totalTests := 0
	for _, suite := range suitesToRun {
		testCount := len(suite.Tests)

		// Apply category filtering
		if len(options.SkipCategories) > 0 {
			filteredCount := 0
			for _, test := range suite.Tests {
				skip := false
				for _, skipCat := range options.SkipCategories {
					if test.Category == skipCat {
						skip = true
						break
					}
				}
				if !skip {
					filteredCount++
				}
			}
			testCount = filteredCount
		}

		totalTests += testCount
		fmt.Printf("   • %s: %d tests\n", suite.Name, testCount)
	}

	fmt.Printf("🧪 Total Tests: %d\n", totalTests)

	// Display execution settings
	fmt.Println("\n⚙️  Execution Settings:")
	if testDryRun {
		fmt.Println("   • Mode: DRY RUN (no tests will be executed)")
	} else {
		fmt.Println("   • Mode: LIVE (tests will be executed)")
	}

	if testParallel {
		fmt.Println("   • Execution: PARALLEL (faster but less detailed logging)")
	} else {
		fmt.Println("   • Execution: SEQUENTIAL (detailed progress reporting)")
	}

	fmt.Printf("   • Timeout: %s\n", options.Timeout)
	fmt.Printf("   • Output Format: %s\n", options.OutputFormat)

	if testFailFast {
		fmt.Println("   • Fail Fast: ENABLED (stop on first failure)")
	}

	if len(options.SkipCategories) > 0 {
		fmt.Printf("   • Skipped Categories: %s\n", strings.Join(options.SkipCategories, ", "))
	}

	return nil
}

// displayTestResults displays test execution results
func displayTestResults(execution *validation.TestExecution, options *validation.TestOptions) error {
	if testJSONOutput || options.OutputFormat == "json" {
		return displayJSONResults(execution)
	}

	if testQuiet {
		return displayQuietResults(execution)
	}

	if testShowDetails || options.OutputFormat == "detailed" {
		return displayDetailedResults(execution)
	}

	return displaySummaryResults(execution)
}

// displaySummaryResults displays a summary of test results
func displaySummaryResults(execution *validation.TestExecution) error {
	fmt.Println("\n📊 Test Results Summary:")
	fmt.Println("=========================")

	fmt.Printf("🕐 Session ID: %s\n", execution.SessionID)
	fmt.Printf("⏱️  Duration: %s\n", execution.Duration.Round(time.Second))
	fmt.Printf("📈 Total Suites: %d\n", len(execution.Suites))

	// Overall summary
	fmt.Printf("\n🧪 Test Summary:\n")
	fmt.Printf("   ✅ Passed: %d\n", execution.Summary.Passed)
	fmt.Printf("   ❌ Failed: %d\n", execution.Summary.Failed)
	fmt.Printf("   ⏭️  Skipped: %d\n", execution.Summary.Skipped)
	fmt.Printf("   📊 Total: %d\n", execution.Summary.Total)

	if execution.Summary.Warnings > 0 {
		fmt.Printf("   ⚠️  Warnings: %d\n", execution.Summary.Warnings)
	}

	// Suite breakdown
	if len(execution.Suites) > 0 {
		fmt.Println("\n📋 Suite Results:")
		for _, suiteResult := range execution.Suites {
			status := "✅"
			if suiteResult.Summary.Failed > 0 {
				status = "❌"
			} else if suiteResult.Summary.Skipped == suiteResult.Summary.Total {
				status = "⏭️"
			}

			fmt.Printf("   %s %s: %d/%d passed (%s)\n",
				status,
				suiteResult.Suite.Name,
				suiteResult.Summary.Passed,
				suiteResult.Summary.Total,
				suiteResult.Duration.Round(time.Millisecond))
		}
	}

	// Success rate
	if execution.Summary.Total > 0 {
		successRate := float64(execution.Summary.Passed) / float64(execution.Summary.Total) * 100
		fmt.Printf("\n🎯 Success Rate: %.1f%%\n", successRate)
	}

	return nil
}

// displayDetailedResults displays detailed test results
func displayDetailedResults(execution *validation.TestExecution) error {
	// First show summary
	if err := displaySummaryResults(execution); err != nil {
		return err
	}

	// Then show detailed results
	fmt.Println("\n🔍 Detailed Test Results:")
	fmt.Println("=========================")

	for _, suiteResult := range execution.Suites {
		fmt.Printf("\n📋 Suite: %s (%s)\n", suiteResult.Suite.Name, suiteResult.Duration.Round(time.Millisecond))
		fmt.Printf("   Description: %s\n", suiteResult.Suite.Description)

		for i, result := range suiteResult.Results {
			test := suiteResult.Suite.Tests[i]
			status := "✅ PASS"
			if !result.Success {
				if skipped, exists := result.Metadata["skipped"]; exists && skipped == "true" {
					status = "⏭️ SKIP"
				} else {
					status = "❌ FAIL"
				}
			}

			fmt.Printf("      %s %s (%s)\n", status, test.Name, result.Duration.Round(time.Millisecond))

			if !result.Success && result.ErrorMessage != "" {
				fmt.Printf("         Error: %s\n", result.ErrorMessage)
			}

			if len(result.Warnings) > 0 {
				for _, warning := range result.Warnings {
					fmt.Printf("         Warning: %s\n", warning)
				}
			}
		}
	}

	return nil
}

// displayJSONResults displays results in JSON format
func displayJSONResults(execution *validation.TestExecution) error {
	data, err := json.MarshalIndent(execution, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// displayQuietResults displays minimal results
func displayQuietResults(execution *validation.TestExecution) error {
	if execution.Summary.Failed > 0 {
		fmt.Printf("FAIL: %d/%d tests failed\n", execution.Summary.Failed, execution.Summary.Total)
	} else {
		fmt.Printf("PASS: %d/%d tests passed\n", execution.Summary.Passed, execution.Summary.Total)
	}
	return nil
}

// saveTestResults saves test results to file
func saveTestResults(execution *validation.TestExecution, options *validation.TestOptions) error {
	outputFile := testOutputFile
	if outputFile == "" {
		timestamp := time.Now().Format("20060102-150405")
		outputFile = fmt.Sprintf("test-results-%s.json", timestamp)
	}

	data, err := json.MarshalIndent(execution, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return err
	}

	fmt.Printf("💾 Test results saved to: %s\n", outputFile)
	return nil
}

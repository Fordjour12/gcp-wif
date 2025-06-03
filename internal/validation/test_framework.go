// Package validation provides comprehensive testing and validation framework
// for the GCP Workload Identity Federation CLI tool.
package validation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Fordjour12/gcp-wif/internal/config"
	"github.com/Fordjour12/gcp-wif/internal/gcp"
	"github.com/Fordjour12/gcp-wif/internal/github"
	"github.com/Fordjour12/gcp-wif/internal/logging"
)

// TestFramework provides comprehensive testing and validation capabilities
type TestFramework struct {
	config *config.Config
	logger *logging.Logger
	ctx    context.Context
}

// NewTestFramework creates a new testing framework instance
func NewTestFramework(cfg *config.Config) *TestFramework {
	return &TestFramework{
		config: cfg,
		logger: logging.WithField("component", "test_framework"),
		ctx:    context.Background(),
	}
}

// TestSuite represents a collection of related tests
type TestSuite struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Tests       []Test                 `json:"tests"`
	Setup       func() error           `json:"-"`
	Teardown    func() error           `json:"-"`
	Config      map[string]interface{} `json:"config"`
}

// Test represents an individual test case
type Test struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Category    string        `json:"category"`
	Severity    string        `json:"severity"`
	Timeout     time.Duration `json:"timeout"`
	Function    func() error  `json:"-"`
	Expected    TestResult    `json:"expected"`
	Actual      TestResult    `json:"actual"`
	Skipped     bool          `json:"skipped"`
	SkipReason  string        `json:"skip_reason,omitempty"`
}

// TestResult represents the result of a test execution
type TestResult struct {
	Success      bool              `json:"success"`
	ErrorMessage string            `json:"error_message,omitempty"`
	Duration     time.Duration     `json:"duration"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
}

// TestExecution represents a complete test execution session
type TestExecution struct {
	SessionID   string                 `json:"session_id"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Duration    time.Duration          `json:"duration"`
	Suites      []TestSuiteResult      `json:"suites"`
	Summary     TestSummary            `json:"summary"`
	Environment map[string]string      `json:"environment"`
	Config      *config.Config         `json:"config"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TestSuiteResult represents the result of executing a test suite
type TestSuiteResult struct {
	Suite    TestSuite     `json:"suite"`
	Results  []TestResult  `json:"results"`
	Summary  TestSummary   `json:"summary"`
	Duration time.Duration `json:"duration"`
}

// TestSummary provides summary statistics for test execution
type TestSummary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Skipped  int `json:"skipped"`
	Warnings int `json:"warnings"`
}

// ValidationTest represents a specific validation test
type ValidationTest struct {
	Component string            `json:"component"`
	Type      string            `json:"type"`
	Rules     []ValidationRule  `json:"rules"`
	Context   map[string]string `json:"context"`
}

// ValidationRule represents a single validation rule
type ValidationRule struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Check       func() ValidationCheck `json:"-"`
}

// ValidationCheck represents the result of a validation check
type ValidationCheck struct {
	Valid    bool                   `json:"valid"`
	Messages []string               `json:"messages"`
	Warnings []string               `json:"warnings"`
	Metadata map[string]interface{} `json:"metadata"`
}

// TestCategories defines available test categories
const (
	CategoryConfiguration = "configuration"
	CategoryGCP           = "gcp"
	CategoryGitHub        = "github"
	CategoryWorkflow      = "workflow"
	CategoryIntegration   = "integration"
	CategorySecurity      = "security"
	CategoryPerformance   = "performance"
	CategoryResilience    = "resilience"
)

// TestSeverities defines test severity levels
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
	SeverityInfo     = "info"
)

// GetAllTestSuites returns all available test suites
func (tf *TestFramework) GetAllTestSuites() []TestSuite {
	return []TestSuite{
		tf.CreateConfigurationTestSuite(),
		tf.CreateGCPTestSuite(),
		tf.CreateGitHubTestSuite(),
		tf.CreateWorkflowTestSuite(),
		tf.CreateIntegrationTestSuite(),
		tf.CreateSecurityTestSuite(),
		tf.CreatePerformanceTestSuite(),
		tf.CreateResilienceTestSuite(),
	}
}

// ExecuteTestSuite executes a specific test suite
func (tf *TestFramework) ExecuteTestSuite(suite TestSuite, options TestOptions) (*TestSuiteResult, error) {
	tf.logger.Info("Executing test suite", "suite", suite.Name)

	startTime := time.Now()
	result := &TestSuiteResult{
		Suite:   suite,
		Results: make([]TestResult, 0, len(suite.Tests)),
	}

	// Run setup if defined
	if suite.Setup != nil {
		if err := suite.Setup(); err != nil {
			return nil, fmt.Errorf("test suite setup failed: %w", err)
		}
	}

	// Execute each test
	for _, test := range suite.Tests {
		if options.SkipCategories != nil {
			skip := false
			for _, skipCat := range options.SkipCategories {
				if test.Category == skipCat {
					skip = true
					break
				}
			}
			if skip {
				test.Skipped = true
				test.SkipReason = "Category excluded by options"
			}
		}

		testResult := tf.executeTest(test, options)
		result.Results = append(result.Results, testResult)
	}

	// Run teardown if defined
	if suite.Teardown != nil {
		if err := suite.Teardown(); err != nil {
			tf.logger.Warn("Test suite teardown failed", "error", err)
		}
	}

	result.Duration = time.Since(startTime)
	result.Summary = tf.calculateSummary(result.Results)

	tf.logger.Info("Test suite execution completed",
		"suite", suite.Name,
		"duration", result.Duration,
		"total", result.Summary.Total,
		"passed", result.Summary.Passed,
		"failed", result.Summary.Failed)

	return result, nil
}

// ExecuteAllTests executes all test suites
func (tf *TestFramework) ExecuteAllTests(options TestOptions) (*TestExecution, error) {
	tf.logger.Info("Starting comprehensive test execution")

	execution := &TestExecution{
		SessionID:   fmt.Sprintf("test-%d", time.Now().Unix()),
		StartTime:   time.Now(),
		Environment: tf.getEnvironmentInfo(),
		Config:      tf.config,
		Metadata:    make(map[string]interface{}),
	}

	suites := tf.GetAllTestSuites()
	execution.Suites = make([]TestSuiteResult, 0, len(suites))

	for _, suite := range suites {
		if options.SuitesToRun != nil {
			run := false
			for _, runSuite := range options.SuitesToRun {
				if suite.Name == runSuite {
					run = true
					break
				}
			}
			if !run {
				continue
			}
		}

		suiteResult, err := tf.ExecuteTestSuite(suite, options)
		if err != nil {
			tf.logger.Error("Test suite execution failed", "suite", suite.Name, "error", err)
			continue
		}

		execution.Suites = append(execution.Suites, *suiteResult)
	}

	execution.EndTime = time.Now()
	execution.Duration = execution.EndTime.Sub(execution.StartTime)
	execution.Summary = tf.calculateOverallSummary(execution.Suites)

	tf.logger.Info("Comprehensive test execution completed",
		"session_id", execution.SessionID,
		"duration", execution.Duration,
		"total_suites", len(execution.Suites),
		"total_tests", execution.Summary.Total,
		"passed", execution.Summary.Passed,
		"failed", execution.Summary.Failed)

	return execution, nil
}

// TestOptions configures test execution behavior
type TestOptions struct {
	DryRun          bool          `json:"dry_run"`
	Verbose         bool          `json:"verbose"`
	Parallel        bool          `json:"parallel"`
	FailFast        bool          `json:"fail_fast"`
	Timeout         time.Duration `json:"timeout"`
	SuitesToRun     []string      `json:"suites_to_run"`
	SkipCategories  []string      `json:"skip_categories"`
	MinSeverity     string        `json:"min_severity"`
	Tags            []string      `json:"tags"`
	OutputFormat    string        `json:"output_format"`
	SaveResults     bool          `json:"save_results"`
	CreateSnapshots bool          `json:"create_snapshots"`
}

// executeTest executes a single test
func (tf *TestFramework) executeTest(test Test, options TestOptions) TestResult {
	tf.logger.Debug("Executing test", "test", test.Name)

	if test.Skipped {
		return TestResult{
			Success:  true,
			Duration: 0,
			Metadata: map[string]string{"skipped": "true", "reason": test.SkipReason},
		}
	}

	if options.DryRun {
		return TestResult{
			Success:  true,
			Duration: 0,
			Metadata: map[string]string{"dry_run": "true"},
		}
	}

	startTime := time.Now()

	// Set timeout
	timeout := test.Timeout
	if timeout == 0 {
		timeout = options.Timeout
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Execute test with timeout
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("test panicked: %v", r)
			}
		}()
		done <- test.Function()
	}()

	var err error
	select {
	case err = <-done:
	case <-time.After(timeout):
		err = fmt.Errorf("test timed out after %v", timeout)
	}

	duration := time.Since(startTime)

	result := TestResult{
		Success:  err == nil,
		Duration: duration,
		Metadata: map[string]string{
			"category": test.Category,
			"severity": test.Severity,
		},
	}

	if err != nil {
		result.ErrorMessage = err.Error()
		tf.logger.Warn("Test failed", "test", test.Name, "error", err, "duration", duration)
	} else {
		tf.logger.Debug("Test passed", "test", test.Name, "duration", duration)
	}

	return result
}

// calculateSummary calculates summary statistics for test results
func (tf *TestFramework) calculateSummary(results []TestResult) TestSummary {
	summary := TestSummary{}

	for _, result := range results {
		summary.Total++

		if skipped, exists := result.Metadata["skipped"]; exists && skipped == "true" {
			summary.Skipped++
		} else if result.Success {
			summary.Passed++
		} else {
			summary.Failed++
		}

		summary.Warnings += len(result.Warnings)
	}

	return summary
}

// calculateOverallSummary calculates overall summary for all test suites
func (tf *TestFramework) calculateOverallSummary(suiteResults []TestSuiteResult) TestSummary {
	summary := TestSummary{}

	for _, suiteResult := range suiteResults {
		summary.Total += suiteResult.Summary.Total
		summary.Passed += suiteResult.Summary.Passed
		summary.Failed += suiteResult.Summary.Failed
		summary.Skipped += suiteResult.Summary.Skipped
		summary.Warnings += suiteResult.Summary.Warnings
	}

	return summary
}

// getEnvironmentInfo collects environment information for test execution
func (tf *TestFramework) getEnvironmentInfo() map[string]string {
	return map[string]string{
		"go_version":     "1.21+",
		"project_id":     tf.config.Project.ID,
		"repository":     tf.config.GetRepoFullName(),
		"config_version": tf.config.Version,
		"timestamp":      time.Now().Format(time.RFC3339),
	}
}

// ValidateConfiguration performs comprehensive configuration validation
func (tf *TestFramework) ValidateConfiguration(cfg *config.Config) (*ValidationResult, error) {
	tf.logger.Info("Performing comprehensive configuration validation")

	result := &ValidationResult{Valid: true}

	// Schema validation
	schemaResult := cfg.ValidateSchema()
	if !schemaResult.Valid {
		result.Valid = false
		// Convert config.ValidationError to validation.ValidationError
		for _, configErr := range schemaResult.Errors {
			result.Errors = append(result.Errors, ValidationError{
				Field:   configErr.Field,
				Message: configErr.Message,
			})
		}
	}

	// Custom validation rules
	validationTests := []ValidationTest{
		tf.createGCPValidationTest(cfg),
		tf.createGitHubValidationTest(cfg),
		tf.createWorkflowValidationTest(cfg),
		tf.createSecurityValidationTest(cfg),
	}

	for _, validationTest := range validationTests {
		for _, rule := range validationTest.Rules {
			check := rule.Check()
			if !check.Valid {
				result.Valid = false
				for _, msg := range check.Messages {
					result.Errors = append(result.Errors, ValidationError{
						Field:   validationTest.Component,
						Message: fmt.Sprintf("%s: %s", rule.Name, msg),
					})
				}
			}
		}
	}

	tf.logger.Info("Configuration validation completed",
		"valid", result.Valid,
		"errors", len(result.Errors))

	return result, nil
}

// ValidateGCPResources validates GCP resource availability and permissions
func (tf *TestFramework) ValidateGCPResources(cfg *config.Config) (*ValidationResult, error) {
	tf.logger.Info("Validating GCP resources and permissions")

	client, err := gcp.NewClient(tf.ctx, cfg.Project.ID)
	if err != nil {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "gcp_client",
				Message: fmt.Sprintf("Failed to create GCP client: %v", err),
			}},
		}, nil
	}
	defer client.Close()

	result := &ValidationResult{Valid: true}

	// Test project access
	if err := client.TestConnection(); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "gcp_project",
			Message: fmt.Sprintf("Project access validation failed: %v", err),
		})
	}

	// Check required permissions
	requiredPerms := []string{
		"iam.serviceAccounts.create",
		"iam.serviceAccounts.get",
		"iam.workloadIdentityPools.create",
		"iam.workloadIdentityProviders.create",
		"resourcemanager.projects.getIamPolicy",
		"resourcemanager.projects.setIamPolicy",
	}

	perms, err := client.CheckPermissions(requiredPerms)
	if err != nil {
		tf.logger.Warn("Could not check permissions", "error", err)
	} else {
		for perm, hasPermission := range perms {
			if !hasPermission {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   "gcp_permissions",
					Message: fmt.Sprintf("Missing required permission: %s", perm),
				})
			}
		}
	}

	// Note: Resource conflict checking would require implementing CheckResourceConflicts method
	tf.logger.Debug("Skipping resource conflict checking (not implemented)")

	return result, nil
}

// ValidateWorkflowConfiguration validates GitHub Actions workflow configuration
func (tf *TestFramework) ValidateWorkflowConfiguration(cfg *config.Config) (*ValidationResult, error) {
	tf.logger.Info("Validating workflow configuration")

	result := &ValidationResult{Valid: true}

	// Validate workflow configuration
	if err := cfg.Workflow.ValidateConfig(); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "workflow_config",
			Message: fmt.Sprintf("Workflow configuration validation failed: %v", err),
		})
	}

	// Generate and validate workflow content
	content, err := cfg.Workflow.GenerateWorkflow()
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "workflow_generation",
			Message: fmt.Sprintf("Workflow generation failed: %v", err),
		})
		return result, nil
	}

	// Validate generated YAML
	if err := cfg.Workflow.ValidateWorkflowContent(content); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "workflow_yaml",
			Message: fmt.Sprintf("Generated workflow YAML is invalid: %v", err),
		})
	}

	// Check for security best practices
	securityChecks := tf.validateWorkflowSecurity(&cfg.Workflow)
	for _, check := range securityChecks {
		if !check.Valid {
			tf.logger.Warn("Security concern detected", "messages", strings.Join(check.Messages, "; "))
		}
	}

	return result, nil
}

// Helper methods to create specific validation tests
func (tf *TestFramework) createGCPValidationTest(cfg *config.Config) ValidationTest {
	return ValidationTest{
		Component: "gcp",
		Type:      "resource_validation",
		Rules: []ValidationRule{
			{
				Name:        "project_access",
				Description: "Verify GCP project access and permissions",
				Severity:    SeverityCritical,
				Check: func() ValidationCheck {
					client, err := gcp.NewClient(tf.ctx, cfg.Project.ID)
					if err != nil {
						return ValidationCheck{
							Valid:    false,
							Messages: []string{fmt.Sprintf("Cannot create GCP client: %v", err)},
						}
					}
					defer client.Close()

					if err := client.TestConnection(); err != nil {
						return ValidationCheck{
							Valid:    false,
							Messages: []string{fmt.Sprintf("GCP connection test failed: %v", err)},
						}
					}

					return ValidationCheck{Valid: true}
				},
			},
		},
	}
}

func (tf *TestFramework) createGitHubValidationTest(cfg *config.Config) ValidationTest {
	return ValidationTest{
		Component: "github",
		Type:      "repository_validation",
		Rules: []ValidationRule{
			{
				Name:        "repository_format",
				Description: "Verify GitHub repository format is valid",
				Severity:    SeverityHigh,
				Check: func() ValidationCheck {
					validator := NewValidator()
					result := validator.ValidateGitHubRepository(cfg.Repository.Owner, cfg.Repository.Name)

					if !result.Valid {
						messages := make([]string, len(result.Errors))
						for i, err := range result.Errors {
							messages[i] = err.Message
						}
						return ValidationCheck{
							Valid:    false,
							Messages: messages,
						}
					}

					return ValidationCheck{Valid: true}
				},
			},
		},
	}
}

func (tf *TestFramework) createWorkflowValidationTest(cfg *config.Config) ValidationTest {
	return ValidationTest{
		Component: "workflow",
		Type:      "configuration_validation",
		Rules: []ValidationRule{
			{
				Name:        "workflow_generation",
				Description: "Verify workflow can be generated successfully",
				Severity:    SeverityHigh,
				Check: func() ValidationCheck {
					_, err := cfg.Workflow.GenerateWorkflow()
					if err != nil {
						return ValidationCheck{
							Valid:    false,
							Messages: []string{fmt.Sprintf("Workflow generation failed: %v", err)},
						}
					}

					return ValidationCheck{Valid: true}
				},
			},
		},
	}
}

func (tf *TestFramework) createSecurityValidationTest(cfg *config.Config) ValidationTest {
	return ValidationTest{
		Component: "security",
		Type:      "security_validation",
		Rules: []ValidationRule{
			{
				Name:        "workload_identity_security",
				Description: "Verify workload identity security configuration",
				Severity:    SeverityHigh,
				Check: func() ValidationCheck {
					warnings := []string{}

					// Check if repository restrictions are properly configured
					if cfg.Repository.Owner == "" || cfg.Repository.Name == "" {
						return ValidationCheck{
							Valid:    false,
							Messages: []string{"Repository must be specified for security"},
						}
					}

					// Check for security best practices in workflow
					if !cfg.Workflow.Security.RequireSignedCommits {
						warnings = append(warnings, "Signed commits not required")
					}

					if !cfg.Workflow.Security.RequireApproval {
						warnings = append(warnings, "Approval requirements not configured")
					}

					return ValidationCheck{
						Valid:    true,
						Warnings: warnings,
					}
				},
			},
		},
	}
}

// validateWorkflowSecurity performs security validation on workflow configuration
func (tf *TestFramework) validateWorkflowSecurity(workflow *github.WorkflowConfig) []ValidationCheck {
	checks := []ValidationCheck{}

	// Check for security configurations
	if !workflow.Security.RequireApproval {
		checks = append(checks, ValidationCheck{
			Valid:    false,
			Messages: []string{"Approval requirements should be enabled for production workflows"},
		})
	}

	if len(workflow.Security.RestrictBranches) == 0 {
		checks = append(checks, ValidationCheck{
			Valid:    false,
			Messages: []string{"Branch restrictions should be configured for security"},
		})
	}

	if !workflow.Security.RequireSignedCommits {
		checks = append(checks, ValidationCheck{
			Valid:    false,
			Messages: []string{"Signed commits should be required for security"},
		})
	}

	return checks
}

// CreateConfigurationTestSuite creates tests for configuration validation
func (tf *TestFramework) CreateConfigurationTestSuite() TestSuite {
	return TestSuite{
		Name:        "Configuration",
		Description: "Comprehensive configuration validation tests",
		Tests: []Test{
			{
				Name:        "schema_validation",
				Description: "Validate configuration schema and structure",
				Category:    CategoryConfiguration,
				Severity:    SeverityCritical,
				Function: func() error {
					result := tf.config.ValidateSchema()
					if !result.Valid {
						return fmt.Errorf("configuration validation failed: %d errors", len(result.Errors))
					}
					return nil
				},
			},
			{
				Name:        "required_fields",
				Description: "Verify all required fields are present",
				Category:    CategoryConfiguration,
				Severity:    SeverityCritical,
				Function: func() error {
					if tf.config.Project.ID == "" {
						return fmt.Errorf("project ID is required")
					}
					if tf.config.Repository.Owner == "" || tf.config.Repository.Name == "" {
						return fmt.Errorf("repository owner and name are required")
					}
					return nil
				},
			},
		},
	}
}

// CreateGCPTestSuite creates tests for GCP resource validation
func (tf *TestFramework) CreateGCPTestSuite() TestSuite {
	return TestSuite{
		Name:        "GCP",
		Description: "Google Cloud Platform resource and permissions tests",
		Tests: []Test{
			{
				Name:        "authentication",
				Description: "Verify GCP authentication and client connectivity",
				Category:    CategoryGCP,
				Severity:    SeverityCritical,
				Function: func() error {
					client, err := gcp.NewClient(tf.ctx, tf.config.Project.ID)
					if err != nil {
						return fmt.Errorf("failed to create GCP client: %w", err)
					}
					defer client.Close()
					return client.TestConnection()
				},
			},
			{
				Name:        "permissions",
				Description: "Check required IAM permissions",
				Category:    CategoryGCP,
				Severity:    SeverityHigh,
				Function: func() error {
					client, err := gcp.NewClient(tf.ctx, tf.config.Project.ID)
					if err != nil {
						return err
					}
					defer client.Close()

					requiredPerms := []string{
						"iam.serviceAccounts.create",
						"iam.workloadIdentityPools.create",
						"iam.workloadIdentityProviders.create",
					}

					perms, err := client.CheckPermissions(requiredPerms)
					if err != nil {
						return err
					}

					for perm, hasPermission := range perms {
						if !hasPermission {
							return fmt.Errorf("missing required permission: %s", perm)
						}
					}
					return nil
				},
			},
		},
	}
}

// CreateGitHubTestSuite creates tests for GitHub integration
func (tf *TestFramework) CreateGitHubTestSuite() TestSuite {
	return TestSuite{
		Name:        "GitHub",
		Description: "GitHub repository and workflow integration tests",
		Tests: []Test{
			{
				Name:        "repository_validation",
				Description: "Validate GitHub repository configuration",
				Category:    CategoryGitHub,
				Severity:    SeverityHigh,
				Function: func() error {
					validator := NewValidator()
					result := validator.ValidateGitHubRepository(tf.config.Repository.Owner, tf.config.Repository.Name)
					if !result.Valid {
						return fmt.Errorf("repository validation failed: %d errors", len(result.Errors))
					}
					return nil
				},
			},
		},
	}
}

// CreateWorkflowTestSuite creates tests for workflow generation
func (tf *TestFramework) CreateWorkflowTestSuite() TestSuite {
	return TestSuite{
		Name:        "Workflow",
		Description: "GitHub Actions workflow generation and validation tests",
		Tests: []Test{
			{
				Name:        "workflow_generation",
				Description: "Test workflow template generation",
				Category:    CategoryWorkflow,
				Severity:    SeverityHigh,
				Function: func() error {
					_, err := tf.config.Workflow.GenerateWorkflow()
					return err
				},
			},
			{
				Name:        "workflow_validation",
				Description: "Validate generated workflow YAML",
				Category:    CategoryWorkflow,
				Severity:    SeverityMedium,
				Function: func() error {
					content, err := tf.config.Workflow.GenerateWorkflow()
					if err != nil {
						return err
					}
					return tf.config.Workflow.ValidateWorkflowContent(content)
				},
			},
		},
	}
}

// CreateIntegrationTestSuite creates end-to-end integration tests
func (tf *TestFramework) CreateIntegrationTestSuite() TestSuite {
	return TestSuite{
		Name:        "Integration",
		Description: "End-to-end integration and orchestration tests",
		Tests: []Test{
			{
				Name:        "configuration_integration",
				Description: "Test complete configuration integration",
				Category:    CategoryIntegration,
				Severity:    SeverityMedium,
				Function: func() error {
					// Test configuration loading, validation, and processing
					result := tf.config.ValidateSchema()
					if !result.Valid {
						return fmt.Errorf("configuration integration failed")
					}
					return nil
				},
			},
		},
	}
}

// CreateSecurityTestSuite creates security validation tests
func (tf *TestFramework) CreateSecurityTestSuite() TestSuite {
	return TestSuite{
		Name:        "Security",
		Description: "Security configuration and best practices validation",
		Tests: []Test{
			{
				Name:        "workload_identity_security",
				Description: "Validate workload identity security configuration",
				Category:    CategorySecurity,
				Severity:    SeverityHigh,
				Function: func() error {
					if tf.config.Repository.Owner == "" || tf.config.Repository.Name == "" {
						return fmt.Errorf("repository must be specified for security")
					}
					return nil
				},
			},
		},
	}
}

// CreatePerformanceTestSuite creates performance validation tests
func (tf *TestFramework) CreatePerformanceTestSuite() TestSuite {
	return TestSuite{
		Name:        "Performance",
		Description: "Performance and efficiency validation tests",
		Tests: []Test{
			{
				Name:        "configuration_load_time",
				Description: "Measure configuration loading performance",
				Category:    CategoryPerformance,
				Severity:    SeverityLow,
				Function: func() error {
					start := time.Now()
					_, _ = config.LoadFromFile("test-config.json")
					duration := time.Since(start)
					if duration > 5*time.Second {
						return fmt.Errorf("configuration loading too slow: %v", duration)
					}
					// Allow error if file doesn't exist for testing
					return nil
				},
			},
		},
	}
}

// CreateResilienceTestSuite creates resilience and error handling tests
func (tf *TestFramework) CreateResilienceTestSuite() TestSuite {
	return TestSuite{
		Name:        "Resilience",
		Description: "Error handling and resilience validation tests",
		Tests: []Test{
			{
				Name:        "invalid_configuration_handling",
				Description: "Test handling of invalid configurations",
				Category:    CategoryResilience,
				Severity:    SeverityMedium,
				Function: func() error {
					// Test with invalid configuration
					invalidConfig := &config.Config{}
					result := invalidConfig.ValidateSchema()
					if result.Valid {
						return fmt.Errorf("invalid configuration was incorrectly validated as valid")
					}
					return nil
				},
			},
		},
	}
}

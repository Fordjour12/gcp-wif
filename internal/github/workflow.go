// Package github handles GitHub Actions workflow file generation
// for the GCP Workload Identity Federation CLI tool.
package github

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/logging"
)

// WorkflowConfig holds comprehensive configuration for generating GitHub Actions workflows
type WorkflowConfig struct {
	// Workflow metadata
	Name        string `json:"name"`
	Filename    string `json:"filename"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	Version     string `json:"version,omitempty"`

	// Trigger configuration
	Triggers WorkflowTriggers `json:"triggers"`

	// GCP configuration
	ProjectID                string `json:"project_id"`
	ProjectNumber            string `json:"project_number,omitempty"`
	ServiceAccountEmail      string `json:"service_account_email"`
	WorkloadIdentityProvider string `json:"workload_identity_provider"`

	// Repository configuration
	Repository string   `json:"repository"`
	Branches   []string `json:"branches,omitempty"`
	Tags       []string `json:"tags,omitempty"`

	// Cloud Run configuration
	ServiceName  string            `json:"service_name"`
	Region       string            `json:"region"`
	Registry     string            `json:"registry,omitempty"`
	EnvVars      map[string]string `json:"env_vars,omitempty"`
	Secrets      map[string]string `json:"secrets,omitempty"`
	CPULimit     string            `json:"cpu_limit,omitempty"`
	MemoryLimit  string            `json:"memory_limit,omitempty"`
	MaxInstances int               `json:"max_instances,omitempty"`
	MinInstances int               `json:"min_instances,omitempty"`
	Port         int               `json:"port,omitempty"`

	// Build configuration
	DockerfilePath  string            `json:"dockerfile_path"`
	BuildContext    string            `json:"build_context"`
	BuildArgs       map[string]string `json:"build_args,omitempty"`
	BuildSecrets    map[string]string `json:"build_secrets,omitempty"`
	CacheFromImages []string          `json:"cache_from_images,omitempty"`
	MultiPlatform   bool              `json:"multi_platform,omitempty"`
	Platforms       []string          `json:"platforms,omitempty"`

	// Security configuration
	Security SecurityConfig `json:"security"`

	// Advanced configuration
	Advanced AdvancedWorkflowConfig `json:"advanced"`
}

// WorkflowTriggers defines when the workflow should run
type WorkflowTriggers struct {
	Push        PushTrigger        `json:"push,omitempty"`
	PullRequest PullRequestTrigger `json:"pull_request,omitempty"`
	Schedule    []ScheduleTrigger  `json:"schedule,omitempty"`
	Manual      bool               `json:"manual,omitempty"` // workflow_dispatch
	Release     bool               `json:"release,omitempty"`
}

// PushTrigger defines push trigger configuration
type PushTrigger struct {
	Enabled  bool     `json:"enabled"`
	Branches []string `json:"branches,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Paths    []string `json:"paths,omitempty"`
	Ignore   []string `json:"ignore,omitempty"`
}

// PullRequestTrigger defines pull request trigger configuration
type PullRequestTrigger struct {
	Enabled  bool     `json:"enabled"`
	Branches []string `json:"branches,omitempty"`
	Types    []string `json:"types,omitempty"` // opened, synchronize, reopened, etc.
	Paths    []string `json:"paths,omitempty"`
	Ignore   []string `json:"ignore,omitempty"`
}

// ScheduleTrigger defines scheduled trigger configuration
type ScheduleTrigger struct {
	Cron        string `json:"cron"`
	Description string `json:"description,omitempty"`
}

// SecurityConfig defines security settings for the workflow
type SecurityConfig struct {
	RequireApproval      bool     `json:"require_approval,omitempty"`
	RestrictBranches     []string `json:"restrict_branches,omitempty"`
	AllowedActors        []string `json:"allowed_actors,omitempty"`
	MaxTokenLifetime     string   `json:"max_token_lifetime,omitempty"`
	RequiredChecks       []string `json:"required_checks,omitempty"`
	BlockForkedRepos     bool     `json:"block_forked_repos"`
	RequireSignedCommits bool     `json:"require_signed_commits,omitempty"`
}

// AdvancedWorkflowConfig defines advanced workflow settings
type AdvancedWorkflowConfig struct {
	Timeout           string                 `json:"timeout,omitempty"`
	Concurrency       ConcurrencyConfig      `json:"concurrency,omitempty"`
	MatrixStrategy    MatrixStrategy         `json:"matrix_strategy,omitempty"`
	Environments      map[string]Environment `json:"environments,omitempty"`
	NotificationHooks []NotificationHook     `json:"notification_hooks,omitempty"`
	HealthChecks      []HealthCheck          `json:"health_checks,omitempty"`
	ArtifactRetention string                 `json:"artifact_retention,omitempty"`
}

// ConcurrencyConfig defines concurrency settings
type ConcurrencyConfig struct {
	Group            string `json:"group,omitempty"`
	CancelInProgress bool   `json:"cancel_in_progress,omitempty"`
}

// MatrixStrategy defines matrix build configuration
type MatrixStrategy struct {
	Enabled     bool                     `json:"enabled"`
	Variables   map[string][]interface{} `json:"variables,omitempty"`
	Include     []map[string]interface{} `json:"include,omitempty"`
	Exclude     []map[string]interface{} `json:"exclude,omitempty"`
	FailFast    bool                     `json:"fail_fast"`
	MaxParallel int                      `json:"max_parallel,omitempty"`
}

// Environment defines deployment environment configuration
type Environment struct {
	Name       string                `json:"name"`
	URL        string                `json:"url,omitempty"`
	Variables  map[string]string     `json:"variables,omitempty"`
	Secrets    map[string]string     `json:"secrets,omitempty"`
	Protection EnvironmentProtection `json:"protection,omitempty"`
}

// EnvironmentProtection defines environment protection rules
type EnvironmentProtection struct {
	RequiredReviewers []string `json:"required_reviewers,omitempty"`
	WaitTimer         int      `json:"wait_timer,omitempty"` // minutes
	PreventSelfReview bool     `json:"prevent_self_review,omitempty"`
}

// NotificationHook defines notification configuration
type NotificationHook struct {
	Type   string            `json:"type"` // slack, teams, discord, webhook
	URL    string            `json:"url"`
	Events []string          `json:"events,omitempty"` // success, failure, start, etc.
	Config map[string]string `json:"config,omitempty"`
}

// HealthCheck defines health check configuration
type HealthCheck struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Method      string `json:"method,omitempty"`
	Timeout     string `json:"timeout,omitempty"`
	Retries     int    `json:"retries,omitempty"`
	WaitTime    string `json:"wait_time,omitempty"`
	HealthyCode int    `json:"healthy_code,omitempty"`
}

// DefaultWorkflowConfig returns a comprehensive default workflow configuration
func DefaultWorkflowConfig() *WorkflowConfig {
	return &WorkflowConfig{
		Name:        "Deploy to Cloud Run with WIF",
		Filename:    "deploy.yml",
		Path:        ".github/workflows",
		Description: "Automated deployment to Google Cloud Run using Workload Identity Federation",
		Version:     "1.0",

		// Default triggers
		Triggers: WorkflowTriggers{
			Push: PushTrigger{
				Enabled:  true,
				Branches: []string{"main", "master"},
			},
			PullRequest: PullRequestTrigger{
				Enabled:  true,
				Branches: []string{"main", "master"},
				Types:    []string{"opened", "synchronize", "reopened"},
			},
			Manual: true,
		},

		// Build configuration
		DockerfilePath: "Dockerfile",
		BuildContext:   ".",
		Port:           8080,

		// Default region
		Region: "us-central1",

		// Security defaults
		Security: SecurityConfig{
			BlockForkedRepos: true,
			MaxTokenLifetime: "1h",
			RequireApproval:  false,
		},

		// Advanced defaults
		Advanced: AdvancedWorkflowConfig{
			Timeout: "30m",
			Concurrency: ConcurrencyConfig{
				Group:            "${{ github.workflow }}-${{ github.ref }}",
				CancelInProgress: true,
			},
			MatrixStrategy: MatrixStrategy{
				Enabled:  false,
				FailFast: true,
			},
			ArtifactRetention: "30",
		},
	}
}

// DefaultProductionWorkflowConfig returns a production-ready workflow configuration
func DefaultProductionWorkflowConfig() *WorkflowConfig {
	config := DefaultWorkflowConfig()
	config.Name = "Production Deploy to Cloud Run"
	config.Description = "Production deployment with comprehensive security and monitoring"

	// Production security settings
	config.Security.RequireApproval = true
	config.Security.RestrictBranches = []string{"main", "master"}
	config.Security.RequiredChecks = []string{"build", "test", "security-scan"}
	config.Security.RequireSignedCommits = true

	// Production triggers - more restrictive
	config.Triggers.Push.Branches = []string{"main"}
	config.Triggers.PullRequest.Enabled = false // No auto-deploy on PRs in production

	// Production build configuration
	config.MultiPlatform = true
	config.Platforms = []string{"linux/amd64", "linux/arm64"}

	// Production resource limits
	config.CPULimit = "2"
	config.MemoryLimit = "4Gi"
	config.MaxInstances = 1000
	config.MinInstances = 1

	// Production environment
	config.Advanced.Environments = map[string]Environment{
		"production": {
			Name: "production",
			Protection: EnvironmentProtection{
				RequiredReviewers: []string{"@production-team"},
				WaitTimer:         5,
				PreventSelfReview: true,
			},
		},
	}

	return config
}

// DefaultStagingWorkflowConfig returns a staging environment workflow configuration
func DefaultStagingWorkflowConfig() *WorkflowConfig {
	config := DefaultWorkflowConfig()
	config.Name = "Staging Deploy to Cloud Run"
	config.Description = "Staging deployment for testing and validation"

	// Staging triggers - more permissive
	config.Triggers.Push.Branches = []string{"main", "develop", "staging"}
	config.Triggers.PullRequest.Enabled = true

	// Staging security - less restrictive than production
	config.Security.RequireApproval = false
	config.Security.RestrictBranches = []string{"main", "develop", "staging"}

	// Staging resources
	config.CPULimit = "1"
	config.MemoryLimit = "2Gi"
	config.MaxInstances = 10
	config.MinInstances = 0

	return config
}

// DefaultDevelopmentWorkflowConfig returns a development environment workflow configuration
func DefaultDevelopmentWorkflowConfig() *WorkflowConfig {
	config := DefaultWorkflowConfig()
	config.Name = "Development Deploy to Cloud Run"
	config.Description = "Development deployment for feature testing and experimentation"

	// Development triggers - very permissive
	config.Triggers.Push.Branches = []string{"develop", "feature/*", "dev/*"}
	config.Triggers.PullRequest.Enabled = true
	config.Triggers.PullRequest.Branches = []string{"develop", "main"}
	config.Triggers.PullRequest.Types = []string{"opened", "synchronize", "reopened", "ready_for_review"}

	// Development security - minimal restrictions
	config.Security.RequireApproval = false
	config.Security.RestrictBranches = []string{} // No branch restrictions
	config.Security.BlockForkedRepos = false      // Allow forks for development
	config.Security.MaxTokenLifetime = "2h"       // Longer token lifetime for development

	// Development resources - minimal
	config.CPULimit = "0.5"
	config.MemoryLimit = "1Gi"
	config.MaxInstances = 5
	config.MinInstances = 0

	// Development environment with minimal protection
	config.Advanced.Environments = map[string]Environment{
		"development": {
			Name: "development",
			Variables: map[string]string{
				"NODE_ENV":  "development",
				"DEBUG":     "true",
				"LOG_LEVEL": "debug",
			},
			Protection: EnvironmentProtection{
				WaitTimer: 0, // No wait time for development
			},
		},
	}

	// Development timeout - shorter for faster feedback
	config.Advanced.Timeout = "15m"

	return config
}

// GenerateWorkflow generates a comprehensive GitHub Actions workflow file with enhanced WIF authentication
func (w *WorkflowConfig) GenerateWorkflow() (string, error) {
	logger := logging.WithField("function", "GenerateWorkflow")
	logger.Debug("Generating GitHub Actions workflow", "name", w.Name, "filename", w.Filename)

	// Validate configuration before generating
	if err := w.ValidateConfig(); err != nil {
		return "", errors.WrapError(err, errors.ErrorTypeValidation, "WORKFLOW_CONFIG_INVALID",
			"Invalid workflow configuration")
	}

	tmpl := template.New("workflow").Funcs(template.FuncMap{
		"join":      strings.Join,
		"quote":     func(s string) string { return fmt.Sprintf(`"%s"`, s) },
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"now":       func() string { return time.Now().Format("2006-01-02T15:04:05Z") },
		"default": func(defaultValue interface{}, value interface{}) interface{} {
			if value == nil || value == 0 || value == "" {
				return defaultValue
			}
			return value
		},
		"indent": func(spaces int, text string) string {
			lines := strings.Split(text, "\n")
			prefix := strings.Repeat(" ", spaces)
			for i, line := range lines {
				if line != "" {
					lines[i] = prefix + line
				}
			}
			return strings.Join(lines, "\n")
		},
	})

	workflowTemplate := w.buildWorkflowTemplate()

	parsedTemplate, err := tmpl.Parse(workflowTemplate)
	if err != nil {
		return "", errors.WrapError(err, errors.ErrorTypeValidation, "WORKFLOW_TEMPLATE_PARSE_FAILED",
			"Failed to parse workflow template")
	}

	// Prepare comprehensive template data
	data := w.buildTemplateData()

	var output strings.Builder
	if err := parsedTemplate.Execute(&output, data); err != nil {
		return "", errors.WrapError(err, errors.ErrorTypeValidation, "WORKFLOW_TEMPLATE_EXECUTE_FAILED",
			"Failed to execute workflow template")
	}

	logger.Info("GitHub Actions workflow generated successfully",
		"name", w.Name,
		"triggers", fmt.Sprintf("push:%t pr:%t manual:%t", w.Triggers.Push.Enabled, w.Triggers.PullRequest.Enabled, w.Triggers.Manual))

	return output.String(), nil
}

// buildTemplateData prepares comprehensive template data for workflow generation
func (w *WorkflowConfig) buildTemplateData() map[string]interface{} {
	return map[string]interface{}{
		"Name":                     w.Name,
		"Description":              w.Description,
		"Version":                  w.Version,
		"Repository":               w.Repository,
		"ProjectID":                w.ProjectID,
		"ProjectNumber":            w.ProjectNumber,
		"ServiceAccountEmail":      w.ServiceAccountEmail,
		"WorkloadIdentityProvider": w.WorkloadIdentityProvider,
		"ServiceName":              w.ServiceName,
		"Region":                   w.Region,
		"Port":                     w.Port,
		"DockerfilePath":           w.DockerfilePath,
		"BuildContext":             w.BuildContext,
		"MultiPlatform":            w.MultiPlatform,
		"Platforms":                w.Platforms,
		"CacheFromImages":          w.CacheFromImages,
		"BuildArgs":                w.BuildArgs,
		"BuildSecrets":             w.BuildSecrets,
		"EnvVars":                  w.EnvVars,
		"Secrets":                  w.Secrets,
		"CPULimit":                 w.CPULimit,
		"MemoryLimit":              w.MemoryLimit,
		"MaxInstances":             w.MaxInstances,
		"MinInstances":             w.MinInstances,
		"Triggers":                 w.Triggers,
		"Security":                 w.Security,
		"Advanced":                 w.Advanced,
		"RegistryHost":             w.getRegistryHost(),
		"ImageName":                w.getImageName(),
		"ImageTag":                 w.getImageTag(),
		"ImageURI":                 w.getImageURI(),
		"TimeoutMinutes":           w.getTimeoutMinutes(),
		"EnvironmentName":          w.getEnvironmentName(),
		"EnvironmentURL":           w.getEnvironmentURL(),
		"HealthCheckCommands":      w.GenerateHealthCheckCommands("$SERVICE_URL"),
	}
}

// buildWorkflowTemplate builds the comprehensive workflow template
func (w *WorkflowConfig) buildWorkflowTemplate() string {
	return `# {{ .Description }}
# Generated on {{ now }} by GCP WIF CLI Tool v{{ .Version }}
# Repository: {{ .Repository }}
# Project: {{ .ProjectID }}

name: {{ .Name }}

on:{{ if .Triggers.Push.Enabled }}
  push:{{ if .Triggers.Push.Branches }}
    branches: {{ range .Triggers.Push.Branches }}
      - {{ quote . }}{{ end }}{{ end }}{{ if .Triggers.Push.Tags }}
    tags: {{ range .Triggers.Push.Tags }}
      - {{ quote . }}{{ end }}{{ end }}{{ if .Triggers.Push.Paths }}
    paths: {{ range .Triggers.Push.Paths }}
      - {{ quote . }}{{ end }}{{ end }}{{ if .Triggers.Push.Ignore }}
    paths-ignore: {{ range .Triggers.Push.Ignore }}
      - {{ quote . }}{{ end }}{{ end }}{{ end }}{{ if .Triggers.PullRequest.Enabled }}
  pull_request:{{ if .Triggers.PullRequest.Branches }}
    branches: {{ range .Triggers.PullRequest.Branches }}
      - {{ quote . }}{{ end }}{{ end }}{{ if .Triggers.PullRequest.Types }}
    types: {{ range .Triggers.PullRequest.Types }}
      - {{ quote . }}{{ end }}{{ end }}{{ if .Triggers.PullRequest.Paths }}
    paths: {{ range .Triggers.PullRequest.Paths }}
      - {{ quote . }}{{ end }}{{ end }}{{ if .Triggers.PullRequest.Ignore }}
    paths-ignore: {{ range .Triggers.PullRequest.Ignore }}
      - {{ quote . }}{{ end }}{{ end }}{{ end }}{{ if .Triggers.Manual }}
  workflow_dispatch:
    inputs:
      environment:
        description: 'Deployment environment'
        required: false
        default: 'staging'
        type: choice
        options:
          - staging
          - production
      force_deploy:
        description: 'Force deployment (skip health checks)'
        required: false
        default: false
        type: boolean
      debug_mode:
        description: 'Enable debug logging'
        required: false
        default: false
        type: boolean{{ end }}{{ if .Triggers.Release }}
  release:
    types: [published]{{ end }}{{ if .Triggers.Schedule }}
  schedule:{{ range .Triggers.Schedule }}
    - cron: {{ quote .Cron }}  # {{ .Description }}{{ end }}{{ end }}

{{ if .Advanced.Concurrency.Group }}
concurrency:
  group: {{ .Advanced.Concurrency.Group }}
  cancel-in-progress: {{ .Advanced.Concurrency.CancelInProgress }}
{{ end }}

env:
  # GCP Configuration
  PROJECT_ID: {{ .ProjectID }}{{ if .ProjectNumber }}
  PROJECT_NUMBER: {{ .ProjectNumber }}{{ end }}
  REGION: {{ .Region }}
  SERVICE_NAME: {{ .ServiceName }}
  
  # Container Configuration
  REGISTRY: {{ .RegistryHost }}
  IMAGE_NAME: {{ .ImageName }}
  IMAGE_TAG: {{ .ImageTag }}
  DOCKERFILE_PATH: {{ .DockerfilePath }}
  BUILD_CONTEXT: {{ .BuildContext }}
  
  # Workload Identity Configuration
  WORKLOAD_IDENTITY_PROVIDER: {{ .WorkloadIdentityProvider }}
  SERVICE_ACCOUNT: {{ .ServiceAccountEmail }}
  
  # Security Configuration
  MAX_TOKEN_LIFETIME: {{ .Security.MaxTokenLifetime }}{{ if .Port }}
  
  # Application Configuration
  PORT: {{ .Port }}{{ end }}{{ if .EnvVars }}{{ range $key, $value := .EnvVars }}
  {{ $key }}: {{ $value }}{{ end }}{{ end }}

jobs:
  # Security and validation job
  security-checks:{{ if .Security.RequireApproval }}
    environment: {{ .EnvironmentName }}{{ end }}
    runs-on: ubuntu-latest
    outputs:
      should-deploy: ${{ "{{" }} steps.security-gate.outputs.deploy {{ "}}" }}
      environment: ${{ "{{" }} steps.environment.outputs.environment {{ "}}" }}
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      with:
        fetch-depth: 0  # Full history for security scanning

    - name: Determine environment
      id: environment
      run: |
        if [[ "${{ "{{" }} github.event_name {{ "}}" }}" == "workflow_dispatch" ]]; then
          echo "environment=${{ "{{" }} github.event.inputs.environment {{ "}}" }}" >> $GITHUB_OUTPUT
        elif [[ "${{ "{{" }} github.ref {{ "}}" }}" == "refs/heads/main" ]] || [[ "${{ "{{" }} github.ref {{ "}}" }}" == "refs/heads/master" ]]; then
          echo "environment=production" >> $GITHUB_OUTPUT
        else
          echo "environment=staging" >> $GITHUB_OUTPUT
        fi

    {{ if .Security.BlockForkedRepos }}- name: Block forked repositories
      if: github.event.pull_request.head.repo.full_name != github.repository
      run: |
        echo "::error::Deployment from forked repositories is not allowed for security reasons"
        exit 1
    {{ end }}

    {{ if .Security.RequireSignedCommits }}- name: Verify signed commits
      run: |
        if ! git verify-commit HEAD; then
          echo "::error::All commits must be signed for security compliance"
          exit 1
        fi
    {{ end }}

    {{ if .Security.AllowedActors }}- name: Verify allowed actors
      if: github.actor != 'dependabot[bot]'
      run: |
        ALLOWED_ACTORS="{{ join .Security.AllowedActors " " }}"
        if [[ ! " $ALLOWED_ACTORS " =~ " ${{ "{{" }} github.actor {{ "}}" }} " ]]; then
          echo "::error::Actor ${{ "{{" }} github.actor {{ "}}" }} is not in the allowed actors list"
          exit 1
        fi
    {{ end }}

    - name: Security gate
      id: security-gate
      run: |
        # Determine if deployment should proceed based on various factors
        SHOULD_DEPLOY="true"
        
        # Check if this is a PR from a fork
        {{ if .Security.BlockForkedRepos }}if [[ "${{ "{{" }} github.event_name {{ "}}" }}" == "pull_request" ]] && [[ "${{ "{{" }} github.event.pull_request.head.repo.full_name {{ "}}" }}" != "${{ "{{" }} github.repository {{ "}}" }}" ]]; then
          SHOULD_DEPLOY="false"
        fi{{ end }}
        
        # For pull requests, only deploy to staging
        if [[ "${{ "{{" }} github.event_name {{ "}}" }}" == "pull_request" ]] && [[ "${{ "{{" }} steps.environment.outputs.environment {{ "}}" }}" == "production" ]]; then
          SHOULD_DEPLOY="false"
        fi
        
        echo "deploy=$SHOULD_DEPLOY" >> $GITHUB_OUTPUT
        echo "Deployment decision: $SHOULD_DEPLOY"

  # Main deployment job
  deploy:
    needs: security-checks
    if: needs.security-checks.outputs.should-deploy == 'true'{{ if .Security.RequireApproval }}
    environment: 
      name: ${{ "{{" }} needs.security-checks.outputs.environment {{ "}}" }}{{ if .EnvironmentURL }}
      url: ${{ "{{" }} steps.deploy.outputs.url {{ "}}" }}{{ end }}{{ end }}
    
    runs-on: ubuntu-latest{{ if .Advanced.Timeout }}
    timeout-minutes: {{ .TimeoutMinutes }}{{ end }}
    
    # Enhanced permissions for Workload Identity Federation
    permissions:
      contents: read
      id-token: write
      security-events: write
      pull-requests: write  # For commenting deployment status
      issues: write         # For creating deployment issues if needed
    
    outputs:
      deployment-url: ${{ "{{" }} steps.deploy.outputs.url {{ "}}" }}
      image-digest: ${{ "{{" }} steps.build.outputs.digest {{ "}}" }}
      deployment-id: ${{ "{{" }} steps.deploy.outputs.deployment_id {{ "}}" }}
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up build environment
      run: |
        echo "Setting up build environment..."
        echo "Build context: $BUILD_CONTEXT"
        echo "Dockerfile: $DOCKERFILE_PATH"
        echo "Target image: $REGISTRY/$IMAGE_NAME:$IMAGE_TAG"
        
        # Create build info
        cat > build-info.json << EOF
        {
          "git_sha": "${{ "{{" }} github.sha {{ "}}" }}",
          "git_ref": "${{ "{{" }} github.ref {{ "}}" }}",
          "build_time": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
          "workflow_run": "${{ "{{" }} github.run_id {{ "}}" }}",
          "actor": "${{ "{{" }} github.actor {{ "}}" }}",
          "environment": "${{ "{{" }} needs.security-checks.outputs.environment {{ "}}" }}"
        }
        EOF

    # Enhanced Google Cloud authentication with Workload Identity Federation
    - name: Authenticate to Google Cloud
      id: auth
      uses: google-github-actions/auth@v2
      with:
        token_format: access_token
        workload_identity_provider: ${{ "{{" }} env.WORKLOAD_IDENTITY_PROVIDER {{ "}}" }}
        service_account: ${{ "{{" }} env.SERVICE_ACCOUNT {{ "}}" }}
        access_token_lifetime: ${{ "{{" }} env.MAX_TOKEN_LIFETIME {{ "}}" }}{{ if .Advanced.Environments }}
        access_token_scopes: |
          https://www.googleapis.com/auth/cloud-platform
          https://www.googleapis.com/auth/containerregistry
          https://www.googleapis.com/auth/devstorage.read_write{{ end }}

    - name: Verify authentication
      run: |
        echo "Verifying Google Cloud authentication..."
        gcloud auth list
        gcloud config list project
        
        # Verify access to required services
        echo "Verifying access to Cloud Run..."
        gcloud run services list --region=$REGION --limit=1 || echo "No services found (expected for first deployment)"
        
        echo "Verifying access to Artifact Registry..."
        gcloud artifacts repositories list --location=$REGION || echo "No repositories found"

    # Set up Docker Buildx for advanced features
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3
      with:
        install: true{{ if .MultiPlatform }}
        platforms: {{ join .Platforms "," }}{{ end }}

    # Authenticate Docker to Artifact Registry
    - name: Configure Docker authentication
      uses: docker/login-action@v3
      with:
        registry: ${{ "{{" }} env.REGISTRY {{ "}}" }}
        username: oauth2accesstoken
        password: ${{ "{{" }} steps.auth.outputs.access_token {{ "}}" }}

    # Enhanced container build with security scanning
    - name: Build and push container image
      id: build
      uses: docker/build-push-action@v5
      with:
        context: ${{ "{{" }} env.BUILD_CONTEXT {{ "}}" }}
        file: ${{ "{{" }} env.DOCKERFILE_PATH {{ "}}" }}
        push: true
        tags: |
          ${{ "{{" }} env.REGISTRY {{ "}}" }}/${{ "{{" }} env.IMAGE_NAME {{ "}}" }}:${{ "{{" }} env.IMAGE_TAG {{ "}}" }}
          ${{ "{{" }} env.REGISTRY {{ "}}" }}/${{ "{{" }} env.IMAGE_NAME {{ "}}" }}:latest{{ if .MultiPlatform }}
        platforms: {{ join .Platforms "," }}{{ end }}{{ if .CacheFromImages }}
        cache-from: {{ range .CacheFromImages }}
          - type=registry,ref={{ . }}{{ end }}{{ end }}
        cache-to: type=registry,ref=${{ "{{" }} env.REGISTRY {{ "}}" }}/${{ "{{" }} env.IMAGE_NAME {{ "}}" }}:cache,mode=max{{ if .BuildArgs }}
        build-args: |{{ range $key, $value := .BuildArgs }}
          {{ $key }}={{ $value }}{{ end }}{{ end }}{{ if .BuildSecrets }}
        secrets: |{{ range $key, $value := .BuildSecrets }}
          {{ $key }}=${{ "{{" }} secrets.{{ $value }} {{ "}}" }}{{ end }}{{ end }}
        labels: |
          org.opencontainers.image.source=${{ "{{" }} github.server_url {{ "}}" }}/${{ "{{" }} github.repository {{ "}}" }}
          org.opencontainers.image.revision=${{ "{{" }} github.sha {{ "}}" }}
          org.opencontainers.image.created=${{ "{{" }} steps.build.outputs.metadata['org.opencontainers.image.created'] {{ "}}" }}

    # Deploy to Cloud Run with comprehensive configuration
    - name: Deploy to Cloud Run
      id: deploy
      uses: google-github-actions/deploy-cloudrun@v2
      with:
        service: ${{ "{{" }} env.SERVICE_NAME {{ "}}" }}
        region: ${{ "{{" }} env.REGION {{ "}}" }}
        image: ${{ "{{" }} env.REGISTRY {{ "}}" }}/${{ "{{" }} env.IMAGE_NAME {{ "}}" }}:${{ "{{" }} env.IMAGE_TAG {{ "}}" }}{{ if .Port }}
        port: ${{ "{{" }} env.PORT {{ "}}" }}{{ end }}{{ if .EnvVars }}
        env_vars: |{{ range $key, $value := .EnvVars }}
          {{ $key }}={{ $value }}{{ end }}{{ end }}{{ if .Secrets }}
        secrets: |{{ range $key, $value := .Secrets }}
          {{ $key }}=${{ "{{" }} secrets.{{ $value }} {{ "}}" }}{{ end }}{{ end }}{{ if .CPULimit }}
        cpu: {{ .CPULimit }}{{ end }}{{ if .MemoryLimit }}
        memory: {{ .MemoryLimit }}{{ end }}{{ if .MaxInstances }}
        max_instances: {{ .MaxInstances }}{{ end }}{{ if .MinInstances }}
        min_instances: {{ .MinInstances }}{{ end }}
        flags: |
          --max-instances={{ .MaxInstances | default 100 }}
          --min-instances={{ .MinInstances | default 0 }}
          --concurrency=1000
          --timeout=300
          --allow-unauthenticated{{ if .Security.RequireApproval }}
          --ingress=internal{{ else }}
          --ingress=all{{ end }}

    # Health check and verification
    - name: Verify deployment health
      run: |
        echo "Verifying deployment health..."
        SERVICE_URL="${{ "{{" }} steps.deploy.outputs.url {{ "}}" }}"
        
        # Run configured health checks or default basic health check
        {{ .HealthCheckCommands }}
        
        echo "Deployment successful!"
        echo "Service URL: $SERVICE_URL"

    # Post-deployment actions
    - name: Update deployment status
      if: success()
      run: |
        echo "Deployment completed successfully!"
        echo "Service URL: ${{ "{{" }} steps.deploy.outputs.url {{ "}}" }}"
        echo "Image digest: ${{ "{{" }} steps.build.outputs.digest {{ "}}" }}"
        echo "Environment: ${{ "{{" }} needs.security-checks.outputs.environment {{ "}}" }}"

    - name: Comment on PR with deployment info
      if: github.event_name == 'pull_request'
      uses: actions/github-script@v7
      with:
        script: |
          const deploymentUrl = '${{ "{{" }} steps.deploy.outputs.url {{ "}}" }}';
          const environment = '${{ "{{" }} needs.security-checks.outputs.environment {{ "}}" }}';
          const imageDigest = '${{ "{{" }} steps.build.outputs.digest {{ "}}" }}';
          
          const comment = ` + "`" + `## 🚀 Deployment Status
          
          **Environment:** ${environment}
          **Status:** ✅ Deployed successfully
          **Service URL:** [${deploymentUrl}](${deploymentUrl})
          **Image Digest:** ` + "`" + `${imageDigest}` + "`" + `
          **Workflow Run:** [#${{ "{{" }} github.run_number {{ "}}" }}](${{ "{{" }} github.server_url {{ "}}" }}/${{ "{{" }} github.repository {{ "}}" }}/actions/runs/${{ "{{" }} github.run_id {{ "}}" }})
          
          Deployed from commit ${{ "{{" }} github.sha {{ "}}" }} by @${{ "{{" }} github.actor {{ "}}" }}` + "`" + `;
          
          github.rest.issues.createComment({
            issue_number: context.issue.number,
            owner: context.repo.owner,
            repo: context.repo.repo,
            body: comment
          });

    - name: Handle deployment failure
      if: failure()
      run: |
        echo "::error::Deployment failed"
        
        # Get service logs for debugging
        gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=$SERVICE_NAME" \
          --limit=50 \
          --format="table(timestamp,severity,textPayload)" \
          --project=$PROJECT_ID || echo "Could not retrieve logs"

  # Cleanup job (runs on failure)
  cleanup:
    needs: [security-checks, deploy]
    if: failure() && needs.security-checks.outputs.should-deploy == 'true'
    runs-on: ubuntu-latest
    
    permissions:
      contents: read
      id-token: write
    
    steps:
    - name: Authenticate to Google Cloud
      uses: google-github-actions/auth@v2
      with:
        workload_identity_provider: ${{ "{{" }} env.WORKLOAD_IDENTITY_PROVIDER {{ "}}" }}
        service_account: ${{ "{{" }} env.SERVICE_ACCOUNT {{ "}}" }}

    - name: Cleanup failed deployment
      run: |
        echo "Cleaning up failed deployment..."
        
        # Optionally rollback to previous revision
        # gcloud run services update-traffic $SERVICE_NAME --to-revisions=PREVIOUS=100 --region=$REGION
        
        echo "Cleanup completed"
`
}

// WriteWorkflowFileOptions defines options for writing workflow files
type WriteWorkflowFileOptions struct {
	CreateBackup      bool   `json:"create_backup,omitempty"`
	OverwriteExisting bool   `json:"overwrite_existing,omitempty"`
	DryRun            bool   `json:"dry_run,omitempty"`
	Validate          bool   `json:"validate,omitempty"`
	BackupSuffix      string `json:"backup_suffix,omitempty"`
}

// DefaultWriteOptions returns default options for writing workflow files
func DefaultWriteOptions() WriteWorkflowFileOptions {
	return WriteWorkflowFileOptions{
		CreateBackup:      true,
		OverwriteExisting: false,
		DryRun:            false,
		Validate:          true,
		BackupSuffix:      time.Now().Format("20060102-150405"),
	}
}

// WriteWorkflowFile writes the workflow file with basic options (legacy method)
func (w *WorkflowConfig) WriteWorkflowFile(content string) error {
	return w.WriteWorkflowFileWithOptions(content, DefaultWriteOptions())
}

// WriteWorkflowFileWithOptions writes the workflow file with comprehensive options
func (w *WorkflowConfig) WriteWorkflowFileWithOptions(content string, options WriteWorkflowFileOptions) error {
	logger := logging.WithField("function", "WriteWorkflowFileWithOptions")

	// Validate content first if requested
	if options.Validate {
		if err := w.ValidateWorkflowContent(content); err != nil {
			return fmt.Errorf("workflow content validation failed: %w", err)
		}
	}

	// Create the workflows directory if it doesn't exist
	workflowDir := w.Path
	if !options.DryRun {
		if err := os.MkdirAll(workflowDir, 0755); err != nil {
			return fmt.Errorf("failed to create workflow directory: %w", err)
		}
	}

	filePath := filepath.Join(workflowDir, w.Filename)

	// Check if file exists and handle accordingly
	fileExists := false
	if _, err := os.Stat(filePath); err == nil {
		fileExists = true
		logger.Debug("Workflow file already exists", "path", filePath)

		if !options.OverwriteExisting && !options.DryRun {
			return fmt.Errorf("workflow file already exists at %s (use overwrite option to replace)", filePath)
		}
	}

	// Create backup if requested and file exists
	if options.CreateBackup && fileExists && !options.DryRun {
		if err := w.createWorkflowFileBackup(filePath, options.BackupSuffix); err != nil {
			logger.Warn("Failed to create backup", "error", err)
			// Continue with operation but log the warning
		} else {
			logger.Info("Created backup of existing workflow file", "path", filePath)
		}
	}

	// Handle dry run
	if options.DryRun {
		logger.Info("Dry run: Would write workflow file",
			"path", filePath,
			"size", len(content),
			"exists", fileExists,
			"backup", options.CreateBackup && fileExists)
		return nil
	}

	// Write the workflow file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	logger.Info("Successfully wrote workflow file",
		"path", filePath,
		"size", len(content),
		"backup_created", options.CreateBackup && fileExists)

	return nil
}

// createWorkflowFileBackup creates a backup of the existing workflow file
func (w *WorkflowConfig) createWorkflowFileBackup(filePath, suffix string) error {
	// Read existing file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read existing file for backup: %w", err)
	}

	// Create backup filename
	dir := filepath.Dir(filePath)
	filename := filepath.Base(filePath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	backupPath := filepath.Join(dir, fmt.Sprintf("%s.backup-%s%s", nameWithoutExt, suffix, ext))

	// Write backup file
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// ValidateWorkflowContent validates the generated workflow content
func (w *WorkflowConfig) ValidateWorkflowContent(content string) error {
	// Check if content is not empty
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("workflow content is empty")
	}

	// Basic YAML structure validation
	lines := strings.Split(content, "\n")
	if len(lines) < 10 {
		return fmt.Errorf("workflow content appears too short (less than 10 lines)")
	}

	// Check for required sections
	requiredSections := []string{
		"name:",
		"on:",
		"jobs:",
	}

	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			return fmt.Errorf("workflow content missing required section: %s", section)
		}
	}

	// Check for required WIF elements
	requiredWifElements := []string{
		"google-github-actions/auth@v2",
		"workload_identity_provider",
		"service_account",
	}

	for _, element := range requiredWifElements {
		if !strings.Contains(content, element) {
			return fmt.Errorf("workflow content missing required WIF element: %s", element)
		}
	}

	return nil
}

// GetWorkflowFileInfo returns information about the workflow file
func (w *WorkflowConfig) GetWorkflowFileInfo() (*WorkflowFileInfo, error) {
	filePath := w.GetWorkflowFilePath()

	info := &WorkflowFileInfo{
		Path:     filePath,
		Filename: w.Filename,
		Dir:      w.Path,
	}

	// Check if file exists
	if stat, err := os.Stat(filePath); err == nil {
		info.Exists = true
		info.Size = stat.Size()
		info.ModTime = stat.ModTime()
		info.Mode = stat.Mode()
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to stat workflow file: %w", err)
	}

	// Check if directory exists
	if stat, err := os.Stat(w.Path); err == nil {
		info.DirExists = true
		info.DirMode = stat.Mode()
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to stat workflow directory: %w", err)
	}

	return info, nil
}

// WorkflowFileInfo contains information about a workflow file
type WorkflowFileInfo struct {
	Path      string      `json:"path"`
	Filename  string      `json:"filename"`
	Dir       string      `json:"dir"`
	Exists    bool        `json:"exists"`
	Size      int64       `json:"size,omitempty"`
	ModTime   time.Time   `json:"mod_time,omitempty"`
	Mode      os.FileMode `json:"mode,omitempty"`
	DirExists bool        `json:"dir_exists"`
	DirMode   os.FileMode `json:"dir_mode,omitempty"`
}

// GenerateWorkflowPreview generates a preview of the workflow without writing to file
func (w *WorkflowConfig) GenerateWorkflowPreview() (*WorkflowPreview, error) {
	logger := logging.WithField("function", "GenerateWorkflowPreview")

	// Generate workflow content
	content, err := w.GenerateWorkflow()
	if err != nil {
		return nil, fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Get file info
	fileInfo, err := w.GetWorkflowFileInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Validate content
	validationErr := w.ValidateWorkflowContent(content)

	// Count lines and estimate size
	lines := strings.Split(content, "\n")

	preview := &WorkflowPreview{
		Content:       content,
		FileInfo:      fileInfo,
		LineCount:     len(lines),
		ByteSize:      int64(len(content)),
		Valid:         validationErr == nil,
		ValidationErr: validationErr,
		GeneratedAt:   time.Now(),
		Config:        w,
	}

	logger.Info("Generated workflow preview",
		"lines", preview.LineCount,
		"size", preview.ByteSize,
		"valid", preview.Valid)

	return preview, nil
}

// WorkflowPreview contains a preview of the generated workflow
type WorkflowPreview struct {
	Content       string            `json:"content"`
	FileInfo      *WorkflowFileInfo `json:"file_info"`
	LineCount     int               `json:"line_count"`
	ByteSize      int64             `json:"byte_size"`
	Valid         bool              `json:"valid"`
	ValidationErr error             `json:"validation_error,omitempty"`
	GeneratedAt   time.Time         `json:"generated_at"`
	Config        *WorkflowConfig   `json:"config,omitempty"`
}

// GetPreviewSummary returns a formatted summary of the workflow preview
func (wp *WorkflowPreview) GetPreviewSummary() string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("📄 Workflow Preview: %s\n", wp.FileInfo.Filename))
	summary.WriteString(fmt.Sprintf("📍 Path: %s\n", wp.FileInfo.Path))
	summary.WriteString(fmt.Sprintf("📊 Lines: %d | Size: %d bytes\n", wp.LineCount, wp.ByteSize))
	summary.WriteString(fmt.Sprintf("✅ Valid: %t", wp.Valid))

	if wp.ValidationErr != nil {
		summary.WriteString(fmt.Sprintf(" | Error: %s", wp.ValidationErr.Error()))
	}
	summary.WriteString("\n")

	if wp.FileInfo.Exists {
		summary.WriteString(fmt.Sprintf("⚠️  File exists (size: %d bytes, modified: %s)\n",
			wp.FileInfo.Size, wp.FileInfo.ModTime.Format("2006-01-02 15:04:05")))
	} else {
		summary.WriteString("📁 New file (will be created)\n")
	}

	if !wp.FileInfo.DirExists {
		summary.WriteString("📂 Directory will be created\n")
	}

	summary.WriteString(fmt.Sprintf("🕒 Generated: %s\n", wp.GeneratedAt.Format("2006-01-02 15:04:05")))

	return summary.String()
}

// ValidateConfig validates the workflow configuration
func (w *WorkflowConfig) ValidateConfig() error {
	if w.ProjectID == "" {
		return errors.NewValidationError("Workflow: Project ID is required", "workflow.project_id", "REQUIRED")
	}
	if w.ServiceAccountEmail == "" {
		return errors.NewValidationError("Workflow: Service account email is required", "workflow.service_account_email", "REQUIRED")
	}
	if w.WorkloadIdentityProvider == "" {
		return errors.NewValidationError("Workflow: Workload identity provider is required", "workflow.workload_identity_provider", "REQUIRED")
	}
	if w.ServiceName == "" {
		return errors.NewValidationError("Workflow: Service name is required", "workflow.service_name", "REQUIRED")
	}
	if w.Region == "" {
		return errors.NewValidationError("Workflow: Region is required", "workflow.region", "REQUIRED")
	}
	if w.Name == "" {
		return errors.NewValidationError("Workflow: Workflow name is required", "workflow.name", "REQUIRED")
	}
	if w.Filename == "" {
		return errors.NewValidationError("Workflow: Workflow filename is required", "workflow.filename", "REQUIRED")
	}
	if w.Path == "" {
		return errors.NewValidationError("Workflow: Workflow path is required", "workflow.path", "REQUIRED")
	}

	// Validate Triggers
	if !w.Triggers.Push.Enabled && !w.Triggers.PullRequest.Enabled && !w.Triggers.Manual && !w.Triggers.Release && len(w.Triggers.Schedule) == 0 {
		return errors.NewValidationError("Workflow: At least one trigger (push, pull_request, manual, release, or schedule) must be enabled.", "workflow.triggers", "AT_LEAST_ONE_REQUIRED")
	}

	// Validate Security
	if w.Security.MaxTokenLifetime == "" {
		// This might be defaulted elsewhere, but if specified as empty, it's an issue for some auth steps.
		// For now, let's consider it a warning or rely on github.DefaultWorkflowConfig to set a sane default.
		// Or ensure it's always set to a default if empty before this validation.
	}

	// Validate DockerfilePath and BuildContext if Docker build is implied
	if w.DockerfilePath == "" {
		return errors.NewValidationError("Workflow: Dockerfile path is required for building images.", "workflow.dockerfile_path", "REQUIRED")
	}
	if w.BuildContext == "" {
		return errors.NewValidationError("Workflow: Build context is required for building images.", "workflow.build_context", "REQUIRED")
	}

	// Validate environments
	if err := w.ValidateEnvironments(); err != nil {
		return errors.NewValidationError(fmt.Sprintf("Workflow: Environment validation failed: %v", err), "workflow.environments", "INVALID_ENVIRONMENT")
	}

	// Validate health checks
	if err := w.ValidateHealthChecks(); err != nil {
		return errors.NewValidationError(fmt.Sprintf("Workflow: Health check validation failed: %v", err), "workflow.health_checks", "INVALID_HEALTH_CHECK")
	}

	return nil
}

// GetWorkflowFilePath returns the full path to the workflow file
func (w *WorkflowConfig) GetWorkflowFilePath() string {
	return filepath.Join(w.Path, w.Filename)
}

// getImageName extracts the image name from the service name
func (w *WorkflowConfig) getImageName() string {
	if w.ServiceName != "" {
		return w.ServiceName
	}
	// Fallback to repository name
	if w.Repository != "" {
		parts := strings.Split(w.Repository, "/")
		if len(parts) == 2 {
			return strings.ToLower(parts[1])
		}
	}
	return "app"
}

// getImageTag generates the image tag based on git context
func (w *WorkflowConfig) getImageTag() string {
	return "${{ github.sha }}"
}

// getTimeoutMinutes converts timeout string to minutes
func (w *WorkflowConfig) getTimeoutMinutes() int {
	if w.Advanced.Timeout == "" {
		return 30 // default 30 minutes
	}

	// Parse timeout string (e.g., "30m", "1h", "90s")
	timeout := w.Advanced.Timeout
	if strings.HasSuffix(timeout, "m") {
		var minutes int
		if n, err := fmt.Sscanf(timeout, "%dm", &minutes); err == nil && n == 1 {
			return minutes
		}
	} else if strings.HasSuffix(timeout, "h") {
		var hours int
		if n, err := fmt.Sscanf(timeout, "%dh", &hours); err == nil && n == 1 {
			return hours * 60
		}
	} else if strings.HasSuffix(timeout, "s") {
		var seconds int
		if n, err := fmt.Sscanf(timeout, "%ds", &seconds); err == nil && n == 1 {
			return (seconds + 59) / 60 // round up to minutes
		}
	}

	return 30 // fallback
}

// getEnvironmentName determines the environment name for deployment
func (w *WorkflowConfig) getEnvironmentName() string {
	if len(w.Advanced.Environments) > 0 {
		// Return the first environment name
		for name := range w.Advanced.Environments {
			return name
		}
	}

	// Default environment determination logic
	if w.Security.RequireApproval {
		return "production"
	}
	return "staging"
}

// getEnvironmentURL gets the environment URL if configured
func (w *WorkflowConfig) getEnvironmentURL() string {
	envName := w.getEnvironmentName()
	if env, exists := w.Advanced.Environments[envName]; exists {
		return env.URL
	}
	return ""
}

// AddEnvironment adds or updates an environment configuration
func (w *WorkflowConfig) AddEnvironment(name string, env Environment) {
	if w.Advanced.Environments == nil {
		w.Advanced.Environments = make(map[string]Environment)
	}
	env.Name = name // Ensure name matches key
	w.Advanced.Environments[name] = env
}

// RemoveEnvironment removes an environment configuration
func (w *WorkflowConfig) RemoveEnvironment(name string) {
	if w.Advanced.Environments != nil {
		delete(w.Advanced.Environments, name)
	}
}

// GetEnvironment retrieves an environment configuration by name
func (w *WorkflowConfig) GetEnvironment(name string) (Environment, bool) {
	if w.Advanced.Environments == nil {
		return Environment{}, false
	}
	env, exists := w.Advanced.Environments[name]
	return env, exists
}

// ListEnvironments returns a slice of all environment names
func (w *WorkflowConfig) ListEnvironments() []string {
	if w.Advanced.Environments == nil {
		return []string{}
	}

	names := make([]string, 0, len(w.Advanced.Environments))
	for name := range w.Advanced.Environments {
		names = append(names, name)
	}
	return names
}

// AddEnvironmentVariable adds a variable to a specific environment
func (w *WorkflowConfig) AddEnvironmentVariable(envName, key, value string) error {
	env, exists := w.GetEnvironment(envName)
	if !exists {
		return fmt.Errorf("environment '%s' does not exist", envName)
	}

	if env.Variables == nil {
		env.Variables = make(map[string]string)
	}
	env.Variables[key] = value
	w.AddEnvironment(envName, env)
	return nil
}

// AddEnvironmentSecret adds a secret reference to a specific environment
func (w *WorkflowConfig) AddEnvironmentSecret(envName, key, secretRef string) error {
	env, exists := w.GetEnvironment(envName)
	if !exists {
		return fmt.Errorf("environment '%s' does not exist", envName)
	}

	if env.Secrets == nil {
		env.Secrets = make(map[string]string)
	}
	env.Secrets[key] = secretRef
	w.AddEnvironment(envName, env)
	return nil
}

// AddGlobalSecret adds a secret to the top-level workflow secrets
func (w *WorkflowConfig) AddGlobalSecret(key, secretRef string) {
	if w.Secrets == nil {
		w.Secrets = make(map[string]string)
	}
	w.Secrets[key] = secretRef
}

// AddBuildSecret adds a secret for use during the build process
func (w *WorkflowConfig) AddBuildSecret(key, secretRef string) {
	if w.BuildSecrets == nil {
		w.BuildSecrets = make(map[string]string)
	}
	w.BuildSecrets[key] = secretRef
}

// GetEffectiveSecrets returns the merged secrets for a given environment
// Merges global secrets with environment-specific secrets (environment takes precedence)
func (w *WorkflowConfig) GetEffectiveSecrets(envName string) map[string]string {
	effective := make(map[string]string)

	// Start with global secrets
	for key, value := range w.Secrets {
		effective[key] = value
	}

	// Override with environment-specific secrets
	if env, exists := w.GetEnvironment(envName); exists {
		for key, value := range env.Secrets {
			effective[key] = value
		}
	}

	return effective
}

// GetEffectiveVariables returns the merged variables for a given environment
// Merges global env vars with environment-specific variables (environment takes precedence)
func (w *WorkflowConfig) GetEffectiveVariables(envName string) map[string]string {
	effective := make(map[string]string)

	// Start with global environment variables
	for key, value := range w.EnvVars {
		effective[key] = value
	}

	// Override with environment-specific variables
	if env, exists := w.GetEnvironment(envName); exists {
		for key, value := range env.Variables {
			effective[key] = value
		}
	}

	return effective
}

// ValidateEnvironments validates all environment configurations
func (w *WorkflowConfig) ValidateEnvironments() error {
	for envName, env := range w.Advanced.Environments {
		// Validate environment name matches key
		if env.Name != "" && env.Name != envName {
			return fmt.Errorf("environment key '%s' does not match environment name '%s'", envName, env.Name)
		}

		// Validate required reviewers format
		for _, reviewer := range env.Protection.RequiredReviewers {
			if reviewer == "" {
				return fmt.Errorf("environment '%s' has empty required reviewer", envName)
			}
			// Check for valid GitHub username/team format
			if !strings.HasPrefix(reviewer, "@") && !strings.Contains(reviewer, "/") {
				return fmt.Errorf("environment '%s' has invalid reviewer format '%s' (should be @username or org/team)", envName, reviewer)
			}
		}

		// Validate wait timer
		if env.Protection.WaitTimer < 0 || env.Protection.WaitTimer > 43200 { // Max 30 days in minutes
			return fmt.Errorf("environment '%s' has invalid wait timer %d (must be 0-43200 minutes)", envName, env.Protection.WaitTimer)
		}
	}
	return nil
}

// CreateStandardEnvironments creates standard development environments (dev, staging, production)
func (w *WorkflowConfig) CreateStandardEnvironments() {
	if w.Advanced.Environments == nil {
		w.Advanced.Environments = make(map[string]Environment)
	}

	// Development environment - no protection
	w.AddEnvironment("development", Environment{
		Name: "development",
		URL:  "", // Will be set dynamically
		Variables: map[string]string{
			"NODE_ENV": "development",
			"DEBUG":    "true",
		},
	})

	// Staging environment - minimal protection
	w.AddEnvironment("staging", Environment{
		Name: "staging",
		URL:  "", // Will be set dynamically
		Variables: map[string]string{
			"NODE_ENV": "staging",
			"DEBUG":    "false",
		},
		Protection: EnvironmentProtection{
			WaitTimer: 1, // 1 minute wait
		},
	})

	// Production environment - full protection
	w.AddEnvironment("production", Environment{
		Name: "production",
		URL:  "", // Will be set dynamically
		Variables: map[string]string{
			"NODE_ENV": "production",
			"DEBUG":    "false",
		},
		Protection: EnvironmentProtection{
			RequiredReviewers: []string{"@production-team"},
			WaitTimer:         5, // 5 minute wait
			PreventSelfReview: true,
		},
	})
}

// AddHealthCheck adds a health check configuration
func (w *WorkflowConfig) AddHealthCheck(healthCheck HealthCheck) {
	if w.Advanced.HealthChecks == nil {
		w.Advanced.HealthChecks = []HealthCheck{}
	}
	w.Advanced.HealthChecks = append(w.Advanced.HealthChecks, healthCheck)
}

// RemoveHealthCheck removes a health check by name
func (w *WorkflowConfig) RemoveHealthCheck(name string) {
	if w.Advanced.HealthChecks == nil {
		return
	}

	var filtered []HealthCheck
	for _, hc := range w.Advanced.HealthChecks {
		if hc.Name != name {
			filtered = append(filtered, hc)
		}
	}
	w.Advanced.HealthChecks = filtered
}

// GetHealthCheck retrieves a health check by name
func (w *WorkflowConfig) GetHealthCheck(name string) (HealthCheck, bool) {
	for _, hc := range w.Advanced.HealthChecks {
		if hc.Name == name {
			return hc, true
		}
	}
	return HealthCheck{}, false
}

// ListHealthChecks returns all health check names
func (w *WorkflowConfig) ListHealthChecks() []string {
	names := make([]string, 0, len(w.Advanced.HealthChecks))
	for _, hc := range w.Advanced.HealthChecks {
		names = append(names, hc.Name)
	}
	return names
}

// ValidateHealthChecks validates all health check configurations
func (w *WorkflowConfig) ValidateHealthChecks() error {
	for _, hc := range w.Advanced.HealthChecks {
		if err := w.validateSingleHealthCheck(hc); err != nil {
			return fmt.Errorf("health check '%s' validation failed: %w", hc.Name, err)
		}
	}
	return nil
}

// validateSingleHealthCheck validates a single health check configuration
func (w *WorkflowConfig) validateSingleHealthCheck(hc HealthCheck) error {
	if hc.Name == "" {
		return fmt.Errorf("health check name cannot be empty")
	}

	if hc.URL == "" {
		return fmt.Errorf("health check URL cannot be empty")
	}

	// Validate HTTP method
	if hc.Method != "" {
		validMethods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH"}
		valid := false
		for _, method := range validMethods {
			if strings.ToUpper(hc.Method) == method {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid HTTP method '%s'", hc.Method)
		}
	}

	// Validate timeout format
	if hc.Timeout != "" {
		if !strings.HasSuffix(hc.Timeout, "s") && !strings.HasSuffix(hc.Timeout, "m") && !strings.HasSuffix(hc.Timeout, "h") {
			return fmt.Errorf("invalid timeout format '%s' (use format like '30s', '5m', '1h')", hc.Timeout)
		}
	}

	// Validate wait time format
	if hc.WaitTime != "" {
		if !strings.HasSuffix(hc.WaitTime, "s") && !strings.HasSuffix(hc.WaitTime, "m") && !strings.HasSuffix(hc.WaitTime, "h") {
			return fmt.Errorf("invalid wait time format '%s' (use format like '30s', '5m', '1h')", hc.WaitTime)
		}
	}

	// Validate retries
	if hc.Retries < 0 || hc.Retries > 20 {
		return fmt.Errorf("retries must be between 0 and 20, got %d", hc.Retries)
	}

	// Validate healthy code
	if hc.HealthyCode != 0 && (hc.HealthyCode < 100 || hc.HealthyCode > 599) {
		return fmt.Errorf("healthy code must be a valid HTTP status code (100-599), got %d", hc.HealthyCode)
	}

	return nil
}

// CreateDefaultHealthChecks creates default health check configurations for different environments
func (w *WorkflowConfig) CreateDefaultHealthChecks() {
	// Basic health check
	basicHealthCheck := HealthCheck{
		Name:        "basic-health",
		URL:         "/health",
		Method:      "GET",
		Timeout:     "10s",
		Retries:     3,
		WaitTime:    "5s",
		HealthyCode: 200,
	}

	// Readiness check
	readinessCheck := HealthCheck{
		Name:        "readiness-check",
		URL:         "/ready",
		Method:      "GET",
		Timeout:     "15s",
		Retries:     5,
		WaitTime:    "10s",
		HealthyCode: 200,
	}

	// Liveness check
	livenessCheck := HealthCheck{
		Name:        "liveness-check",
		URL:         "/live",
		Method:      "GET",
		Timeout:     "20s",
		Retries:     2,
		WaitTime:    "30s",
		HealthyCode: 200,
	}

	w.AddHealthCheck(basicHealthCheck)
	w.AddHealthCheck(readinessCheck)
	w.AddHealthCheck(livenessCheck)
}

// GetHealthCheckByType returns health checks filtered by type/purpose
func (w *WorkflowConfig) GetHealthCheckByType(checkType string) []HealthCheck {
	var filtered []HealthCheck
	for _, hc := range w.Advanced.HealthChecks {
		switch checkType {
		case "basic":
			if strings.Contains(strings.ToLower(hc.Name), "health") || strings.Contains(strings.ToLower(hc.Name), "basic") {
				filtered = append(filtered, hc)
			}
		case "readiness":
			if strings.Contains(strings.ToLower(hc.Name), "ready") || strings.Contains(strings.ToLower(hc.Name), "readiness") {
				filtered = append(filtered, hc)
			}
		case "liveness":
			if strings.Contains(strings.ToLower(hc.Name), "live") || strings.Contains(strings.ToLower(hc.Name), "liveness") {
				filtered = append(filtered, hc)
			}
		default:
			filtered = append(filtered, hc)
		}
	}
	return filtered
}

// GenerateHealthCheckCommands generates shell commands for health checks in the workflow
func (w *WorkflowConfig) GenerateHealthCheckCommands(serviceURL string) string {
	if len(w.Advanced.HealthChecks) == 0 {
		// Return basic health check if no custom health checks are configured
		return w.generateBasicHealthCheck(serviceURL)
	}

	var commands []string
	commands = append(commands, "echo \"Running configured health checks...\"")

	for _, hc := range w.Advanced.HealthChecks {
		cmd := w.generateHealthCheckCommand(hc, serviceURL)
		commands = append(commands, cmd)
	}

	commands = append(commands, "echo \"All health checks passed!\"")
	return strings.Join(commands, "\n        ")
}

// generateHealthCheckCommand generates a shell command for a single health check
func (w *WorkflowConfig) generateHealthCheckCommand(hc HealthCheck, serviceURL string) string {
	url := serviceURL + hc.URL
	method := "GET"
	if hc.Method != "" {
		method = strings.ToUpper(hc.Method)
	}

	timeout := "30s"
	if hc.Timeout != "" {
		timeout = hc.Timeout
	}

	retries := 3
	if hc.Retries > 0 {
		retries = hc.Retries
	}

	waitTime := "5s"
	if hc.WaitTime != "" {
		waitTime = hc.WaitTime
	}

	expectedCode := 200
	if hc.HealthyCode > 0 {
		expectedCode = hc.HealthyCode
	}

	// Convert timeout format for curl (s, m, h)
	curlTimeout := timeout
	if strings.HasSuffix(timeout, "m") {
		// Convert minutes to seconds for curl
		mins := strings.TrimSuffix(timeout, "m")
		if m, err := fmt.Sscanf(mins, "%d", &retries); err == nil && m == 1 {
			curlTimeout = fmt.Sprintf("%ds", retries*60)
		}
	} else if strings.HasSuffix(timeout, "h") {
		// Convert hours to seconds for curl
		hours := strings.TrimSuffix(timeout, "h")
		if h, err := fmt.Sscanf(hours, "%d", &retries); err == nil && h == 1 {
			curlTimeout = fmt.Sprintf("%ds", retries*3600)
		}
	}

	return fmt.Sprintf(`echo "Running %s health check..."
        for i in {1..%d}; do
          RESPONSE_CODE=$(curl -s -o /dev/null -w "%%{http_code}" -X %s --max-time %s "%s" || echo "000")
          if [ "$RESPONSE_CODE" = "%d" ]; then
            echo "✅ %s health check passed (HTTP %d)"
            break
          else
            echo "❌ %s health check failed (HTTP $RESPONSE_CODE), attempt $i/%d"
            if [ $i -lt %d ]; then
              echo "Waiting %s before retry..."
              sleep %s
            fi
          fi
          if [ $i -eq %d ]; then
            echo "::error::%s health check failed after %d attempts"
            exit 1
          fi
        done`,
		hc.Name, retries, method, curlTimeout, url, expectedCode,
		hc.Name, expectedCode, hc.Name, retries, retries, waitTime, waitTime, retries, hc.Name, retries)
}

// generateBasicHealthCheck generates a basic health check command when no custom health checks are configured
func (w *WorkflowConfig) generateBasicHealthCheck(serviceURL string) string {
	return fmt.Sprintf(`echo "Waiting for service to be ready..."
        for i in {1..30}; do
          if curl -f -s "%s" > /dev/null; then
            echo "✅ Service is healthy!"
            break
          fi
          echo "Attempt $i/30: Service not ready yet, waiting 10 seconds..."
          sleep 10
        done
        
        # Final health check
        if ! curl -f -s "%s" > /dev/null; then
          echo "::error::Service health check failed after 5 minutes"
          exit 1
        fi`, serviceURL, serviceURL)
}

// GenerateAndWriteWorkflow generates and writes the workflow file (legacy method)
func (w *WorkflowConfig) GenerateAndWriteWorkflow() error {
	content, err := w.GenerateWorkflow()
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	if err := w.WriteWorkflowFile(content); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	return nil
}

// GenerateAndWriteWorkflowWithOptions generates and writes the workflow file with options
func (w *WorkflowConfig) GenerateAndWriteWorkflowWithOptions(options WriteWorkflowFileOptions) error {
	content, err := w.GenerateWorkflow()
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	if err := w.WriteWorkflowFileWithOptions(content, options); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	return nil
}

// getRegistryHost extracts the registry host from the registry URL
func (w *WorkflowConfig) getRegistryHost() string {
	if w.Registry == "" {
		// Default Artifact Registry format
		return fmt.Sprintf("%s-docker.pkg.dev", w.Region)
	}

	// Extract host from registry URL
	parts := strings.Split(w.Registry, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return w.Registry
}

// getImageURI returns the full image URI
func (w *WorkflowConfig) getImageURI() string {
	if w.Registry == "" {
		// Default Artifact Registry format
		return fmt.Sprintf("%s-docker.pkg.dev/%s/cloud-run-source-deploy/%s",
			w.Region, w.ProjectID, w.ServiceName)
	}

	// Use provided registry
	if strings.Contains(w.Registry, w.ServiceName) {
		return w.Registry
	}

	return fmt.Sprintf("%s/%s", w.Registry, w.ServiceName)
}

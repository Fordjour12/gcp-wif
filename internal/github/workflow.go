// Package github handles GitHub Actions workflow file generation
// for the GCP Workload Identity Federation CLI tool.
package github

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// WorkflowConfig holds configuration for generating GitHub Actions workflows
type WorkflowConfig struct {
	// Workflow metadata
	Name     string
	Filename string
	Path     string

	// GCP configuration
	ProjectID                string
	ServiceAccountEmail      string
	WorkloadIdentityProvider string

	// Repository configuration
	Repository string

	// Cloud Run configuration
	ServiceName string
	Region      string
	Registry    string

	// Build configuration
	DockerfilePath string
	BuildContext   string
}

// DefaultWorkflowConfig returns a default workflow configuration
func DefaultWorkflowConfig() *WorkflowConfig {
	return &WorkflowConfig{
		Name:           "Deploy to Cloud Run",
		Filename:       "deploy.yml",
		Path:           ".github/workflows",
		DockerfilePath: "Dockerfile",
		BuildContext:   ".",
		Region:         "us-central1",
	}
}

// GenerateWorkflow generates a GitHub Actions workflow file
func (w *WorkflowConfig) GenerateWorkflow() (string, error) {
	tmpl := template.New("workflow")

	workflowTemplate := `name: {{ .Name }}

on:
  push:
    branches: [ "main", "master" ]
  pull_request:
    branches: [ "main", "master" ]

env:
  PROJECT_ID: {{ .ProjectID }}
  SERVICE: {{ .ServiceName }}
  REGION: {{ .Region }}

jobs:
  deploy:
    # Add "id-token" with the intended permissions.
    permissions:
      contents: read
      id-token: write

    runs-on: ubuntu-latest
    steps:
    - name: Checkout
      uses: actions/checkout@v4

    - name: Google Auth
      id: auth
      uses: 'google-github-actions/auth@v2'
      with:
        token_format: 'access_token'
        workload_identity_provider: '{{ .WorkloadIdentityProvider }}'
        service_account: '{{ .ServiceAccountEmail }}'

    - name: Docker Auth
      id: docker-auth
      uses: 'docker/login-action@v3'
      with:
        username: 'oauth2accesstoken'
        password: '${{ "{{" }} steps.auth.outputs.access_token {{ "}}" }}'
        registry: '{{ .RegistryHost }}'

    - name: Build and Push Container
      run: |-
        docker build -f {{ .DockerfilePath }} -t "{{ .ImageURI }}" {{ .BuildContext }}
        docker push "{{ .ImageURI }}"

    - name: Deploy to Cloud Run
      id: deploy
      uses: google-github-actions/deploy-cloudrun@v2
      with:
        service: {{ .ServiceName }}
        region: {{ .Region }}
        image: {{ .ImageURI }}

    - name: Show Output
      run: echo ${{ "{{" }} steps.deploy.outputs.url {{ "}}" }}
`

	parsedTemplate, err := tmpl.Parse(workflowTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse workflow template: %w", err)
	}

	// Prepare template data
	data := struct {
		*WorkflowConfig
		RegistryHost string
		ImageURI     string
	}{
		WorkflowConfig: w,
		RegistryHost:   w.getRegistryHost(),
		ImageURI:       w.getImageURI(),
	}

	var output strings.Builder
	if err := parsedTemplate.Execute(&output, data); err != nil {
		return "", fmt.Errorf("failed to execute workflow template: %w", err)
	}

	return output.String(), nil
}

// WriteWorkflowFile writes the workflow to a file
func (w *WorkflowConfig) WriteWorkflowFile(content string) error {
	// Create the workflows directory if it doesn't exist
	workflowDir := w.Path
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflow directory: %w", err)
	}

	// Write the workflow file
	filePath := filepath.Join(workflowDir, w.Filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	return nil
}

// GenerateAndWriteWorkflow generates and writes the workflow file
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

// ValidateConfig validates the workflow configuration
func (w *WorkflowConfig) ValidateConfig() error {
	if w.ProjectID == "" {
		return fmt.Errorf("project ID is required")
	}
	if w.ServiceAccountEmail == "" {
		return fmt.Errorf("service account email is required")
	}
	if w.WorkloadIdentityProvider == "" {
		return fmt.Errorf("workload identity provider is required")
	}
	if w.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
	if w.Region == "" {
		return fmt.Errorf("region is required")
	}
	if w.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	if w.Filename == "" {
		return fmt.Errorf("workflow filename is required")
	}
	if w.Path == "" {
		return fmt.Errorf("workflow path is required")
	}

	return nil
}

// GetWorkflowFilePath returns the full path to the workflow file
func (w *WorkflowConfig) GetWorkflowFilePath() string {
	return filepath.Join(w.Path, w.Filename)
}

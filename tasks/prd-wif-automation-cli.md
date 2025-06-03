# Product Requirements Document: WIF Automation CLI Tool

## Introduction/Overview

The WIF Automation CLI Tool is a command-line application designed to automate the setup of Google Cloud Workload Identity Federation (WIF) for GitHub repositories. This tool addresses the frustration developers face when manually configuring WIF through the slow and unreliable Google Cloud Console web interface, which frequently crashes and makes the setup process error-prone and time-consuming.

The goal is to provide a fast, reliable, and interactive CLI experience that automatically configures all necessary GCP resources, GitHub repository settings, and generates the required GitHub Actions workflow files for seamless CI/CD deployment to Google Cloud Platform.

## Goals

1. **Eliminate Manual WIF Setup**: Completely automate the WIF configuration process that currently requires multiple manual steps in the GCP Console
2. **Improve Setup Reliability**: Provide a stable alternative to the buggy GCP web interface
3. **Reduce Setup Time**: Decrease WIF configuration time from 30+ minutes to under 5 minutes
4. **Prevent Configuration Errors**: Generate correct configurations automatically to avoid deployment failures
5. **Enhance Developer Experience**: Provide an intuitive, interactive CLI interface using modern terminal UI components

## User Stories

1. **As a developer**, I want to set up WIF configuration correctly so that my deployments don't fail due to authentication issues.

2. **As a developer**, I want to quickly configure WIF for new projects so that I can focus on building features instead of fighting with infrastructure setup.

3. **As a developer**, I want an alternative to the slow and buggy GCP Console so that I can reliably set up WIF without dealing with crashes and timeouts.

4. **As a developer**, I want the tool to generate the correct GitHub Actions workflow file so that I don't have to manually write and debug YAML configurations.

5. **As a developer**, I want clear error messages when something goes wrong so that I can quickly resolve issues without extensive troubleshooting.

## Functional Requirements

1. **System Prerequisites Check**: The tool must verify that `gcloud` CLI is installed and authenticated on the user's system before proceeding.

2. **Interactive Configuration Gathering**: The tool must collect the following information through an interactive Bubble Tea interface:
   - GCP Project ID
   - GitHub repository (org/repo format)
   - Service account name and required permissions
   - Cloud Run service details (name, region)
   - Artifact Registry repository name

3. **JSON Configuration File Support**: The tool must support loading configuration from a JSON file to enable reproducible setups and automation.

4. **Command-Line Flag Support**: The tool must accept configuration values via command-line flags to help users understand available options and enable scripting.

5. **Service Account Creation**: The tool must create a new GCP service account with the following roles:
   - Artifact Registry Writer
   - Cloud Run Admin
   - Service Account User (when applicable)

6. **Workload Identity Pool Creation**: The tool must create a Workload Identity Pool with appropriate configuration for GitHub Actions OIDC integration.

7. **Workload Identity Provider Setup**: The tool must create a Workload Identity Provider with:
   - GitHub OIDC issuer URL configuration
   - Repository-specific attribute mapping
   - Security conditions to restrict access to the specified GitHub repository

8. **GitHub Actions Workflow Generation**: The tool must generate a complete `.github/workflows/deploy.yml` file with:
   - Correct WIF authentication configuration
   - Docker build and push steps for Artifact Registry
   - Cloud Run deployment configuration
   - Proper environment variable handling

9. **Conflict Detection and Warning**: The tool must detect existing resources (service accounts, pools, providers) and warn users about potential conflicts before proceeding.

10. **Detailed Error Reporting**: The tool must provide comprehensive error messages with:
    - Clear description of what went wrong
    - Suggested solutions or next steps
    - Relevant documentation links when applicable

11. **Progress Indication**: The tool must display real-time progress using Bubble Tea components to show users which steps are being executed.

12. **Configuration Summary**: The tool must display a summary of all created resources and configurations before completion.

## Non-Goals (Out of Scope)

1. **Validation and Testing**: The tool will not include built-in validation or testing of the WIF configuration, as this will be verified when users push to their repository and trigger GitHub Actions.

2. **Multi-Cloud Support**: This tool is specifically for Google Cloud Platform and will not support other cloud providers.

3. **GitHub Enterprise Server**: Initial version will only support GitHub.com repositories.

4. **Resource Cleanup**: The tool will not include functionality to remove or clean up created resources.

5. **Advanced Permission Management**: Complex IAM role customization beyond the basic required roles is out of scope.

6. **GUI Interface**: This is specifically a CLI tool; no web or desktop GUI will be provided.

7. **Team/Organization Management**: This is designed as a personal tool and will not include multi-user or organization-wide management features.

## Design Considerations

- **Interactive UI**: Utilize Charm's Bubble Tea framework to create an engaging and intuitive terminal user interface
- **Progressive Disclosure**: Present configuration options step-by-step to avoid overwhelming users
- **Visual Feedback**: Use spinners, progress bars, and color coding to provide clear visual feedback during operations
- **Input Validation**: Implement real-time validation of user inputs with helpful error messages
- **Configuration Preview**: Show users a summary of what will be created before executing changes

## Technical Considerations

- **Language**: Implement in Go to leverage excellent CLI tooling and Bubble Tea framework
- **Authentication**: Integrate with existing `gcloud` CLI authentication using Google Cloud SDK libraries
- **API Integration**: Use Google Cloud APIs for resource creation and management
- **File System Operations**: Generate and write GitHub Actions workflow files to the correct repository path
- **Error Handling**: Implement robust error handling with retry logic for transient API failures
- **Configuration Storage**: Support JSON configuration files with schema validation
- **Cross-Platform Compatibility**: Ensure the tool works on macOS, Linux, and Windows

## Success Metrics

1. **Setup Time Reduction**: Reduce WIF configuration time from 30+ minutes to under 5 minutes
2. **Error Rate Reduction**: Eliminate common configuration errors that cause deployment failures
3. **User Satisfaction**: Positive feedback on ease of use compared to manual GCP Console setup
4. **Adoption Rate**: Personal usage frequency for new project setups
5. **Configuration Accuracy**: Successfully generated configurations that work on first GitHub Actions run

## Open Questions

1. Should the tool support updating existing WIF configurations, or only initial setup?
2. How detailed should the logging be? Should it support different verbosity levels?
3. Should the tool generate example application code (main.py, Dockerfile) for users who don't have existing projects?
4. Would it be valuable to include a "dry-run" mode that shows what would be created without actually creating resources?
5. Should the tool support multiple deployment targets (staging, production) in a single configuration?
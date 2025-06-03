// Package errors provides custom error types and error handling utilities
// for the GCP Workload Identity Federation CLI tool.
package errors

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeConfiguration  ErrorType = "configuration"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeNetwork        ErrorType = "network"
	ErrorTypeGCP            ErrorType = "gcp"
	ErrorTypeGitHub         ErrorType = "github"
	ErrorTypeFileSystem     ErrorType = "filesystem"
	ErrorTypeInternal       ErrorType = "internal"
	ErrorTypeUser           ErrorType = "user"
)

// CustomError represents a custom error with additional context
type CustomError struct {
	Type        ErrorType
	Code        string
	Message     string
	Details     string
	Cause       error
	Suggestions []string
}

// Error implements the error interface
func (e *CustomError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Cause.Error())
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *CustomError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error
func (e *CustomError) Is(target error) bool {
	if t, ok := target.(*CustomError); ok {
		return e.Type == t.Type && e.Code == t.Code
	}
	return false
}

// NewError creates a new custom error
func NewError(errorType ErrorType, code, message string) *CustomError {
	return &CustomError{
		Type:    errorType,
		Code:    code,
		Message: message,
	}
}

// NewErrorWithCause creates a new custom error with an underlying cause
func NewErrorWithCause(errorType ErrorType, code, message string, cause error) *CustomError {
	return &CustomError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WithDetails adds details to the error
func (e *CustomError) WithDetails(details string) *CustomError {
	e.Details = details
	return e
}

// WithSuggestions adds suggestions to resolve the error
func (e *CustomError) WithSuggestions(suggestions ...string) *CustomError {
	e.Suggestions = suggestions
	return e
}

// GetUserFriendlyMessage returns a user-friendly error message with suggestions
func (e *CustomError) GetUserFriendlyMessage() string {
	var msg strings.Builder

	// Main error message
	msg.WriteString(fmt.Sprintf("❌ %s", e.Message))

	// Add details if available
	if e.Details != "" {
		msg.WriteString(fmt.Sprintf("\n   Details: %s", e.Details))
	}

	// Add underlying cause if available
	if e.Cause != nil {
		msg.WriteString(fmt.Sprintf("\n   Cause: %s", e.Cause.Error()))
	}

	// Add suggestions if available
	if len(e.Suggestions) > 0 {
		msg.WriteString("\n\n💡 Suggestions:")
		for i, suggestion := range e.Suggestions {
			msg.WriteString(fmt.Sprintf("\n   %d. %s", i+1, suggestion))
		}
	}

	return msg.String()
}

// Common error definitions
var (
	// Validation errors
	ErrInvalidProjectID = NewError(ErrorTypeValidation, "INVALID_PROJECT_ID",
		"Invalid GCP project ID").WithSuggestions(
		"Project ID must be 6-30 characters long",
		"Must start with a lowercase letter",
		"Can only contain lowercase letters, digits, and hyphens",
		"Cannot end with a hyphen")

	ErrInvalidRepository = NewError(ErrorTypeValidation, "INVALID_REPOSITORY",
		"Invalid GitHub repository").WithSuggestions(
		"Repository should be in format 'owner/name'",
		"Owner and name must contain only alphanumeric characters, hyphens, underscores, and periods",
		"Owner cannot be longer than 39 characters",
		"Repository name cannot be longer than 100 characters")

	ErrInvalidServiceAccount = NewError(ErrorTypeValidation, "INVALID_SERVICE_ACCOUNT",
		"Invalid service account name").WithSuggestions(
		"Service account name must be 6-30 characters long",
		"Must start with a lowercase letter",
		"Can only contain lowercase letters, digits, and hyphens",
		"Cannot end with a hyphen")

	// Configuration errors
	ErrConfigNotFound = NewError(ErrorTypeConfiguration, "CONFIG_NOT_FOUND",
		"Configuration file not found").WithSuggestions(
		"Create a configuration file using the interactive mode",
		"Use the --config flag to specify a different config file path",
		"Run the command with --interactive flag to set up configuration")

	ErrConfigInvalid = NewError(ErrorTypeConfiguration, "CONFIG_INVALID",
		"Configuration file is invalid").WithSuggestions(
		"Check the configuration file syntax",
		"Ensure all required fields are present",
		"Run the command with --interactive flag to recreate the configuration")

	// Authentication errors
	ErrGCloudNotInstalled = NewError(ErrorTypeAuthentication, "GCLOUD_NOT_INSTALLED",
		"Google Cloud CLI (gcloud) is not installed or not in PATH").WithSuggestions(
		"Install Google Cloud CLI from https://cloud.google.com/sdk/docs/install",
		"Ensure gcloud is in your system PATH",
		"Restart your terminal after installation")

	ErrGCloudNotAuthenticated = NewError(ErrorTypeAuthentication, "GCLOUD_NOT_AUTHENTICATED",
		"No active gcloud authentication found").WithSuggestions(
		"Run 'gcloud auth login' to authenticate",
		"Run 'gcloud auth application-default login' for application default credentials",
		"Ensure you have access to the specified GCP project")

	ErrInsufficientPermissions = NewError(ErrorTypeAuthentication, "INSUFFICIENT_PERMISSIONS",
		"Insufficient permissions to perform this operation").WithSuggestions(
		"Ensure your account has the necessary IAM roles",
		"Contact your GCP project administrator",
		"Required roles: Project IAM Admin, Service Account Admin, Workload Identity Pool Admin")

	// GCP errors
	ErrProjectNotFound = NewError(ErrorTypeGCP, "PROJECT_NOT_FOUND",
		"GCP project not found or not accessible").WithSuggestions(
		"Verify the project ID is correct",
		"Ensure you have access to the project",
		"Check if the project exists in the GCP Console")

	ErrServiceAccountExists = NewError(ErrorTypeGCP, "SERVICE_ACCOUNT_EXISTS",
		"Service account already exists").WithSuggestions(
		"Use a different service account name",
		"Delete the existing service account if no longer needed",
		"Use the existing service account if appropriate")

	ErrWorkloadIdentityPoolExists = NewError(ErrorTypeGCP, "WORKLOAD_IDENTITY_POOL_EXISTS",
		"Workload Identity Pool already exists").WithSuggestions(
		"Use a different pool ID",
		"Delete the existing pool if no longer needed",
		"Use the existing pool if appropriate")

	// GitHub errors
	ErrRepositoryNotFound = NewError(ErrorTypeGitHub, "REPOSITORY_NOT_FOUND",
		"GitHub repository not found or not accessible").WithSuggestions(
		"Verify the repository name is correct",
		"Ensure you have access to the repository",
		"Check if the repository exists on GitHub")

	ErrWorkflowFileExists = NewError(ErrorTypeGitHub, "WORKFLOW_FILE_EXISTS",
		"GitHub Actions workflow file already exists").WithSuggestions(
		"Use a different filename",
		"Backup and remove the existing workflow file",
		"Merge the configurations manually")

	// File system errors
	ErrFileNotFound = NewError(ErrorTypeFileSystem, "FILE_NOT_FOUND",
		"Required file not found").WithSuggestions(
		"Check if the file path is correct",
		"Ensure the file exists",
		"Check file permissions")

	ErrDirectoryNotWritable = NewError(ErrorTypeFileSystem, "DIRECTORY_NOT_WRITABLE",
		"Directory is not writable").WithSuggestions(
		"Check directory permissions",
		"Ensure the directory exists",
		"Run with appropriate privileges")
)

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errorType ErrorType) bool {
	var customErr *CustomError
	if errors.As(err, &customErr) {
		return customErr.Type == errorType
	}
	return false
}

// IsErrorCode checks if an error has a specific code
func IsErrorCode(err error, code string) bool {
	var customErr *CustomError
	if errors.As(err, &customErr) {
		return customErr.Code == code
	}
	return false
}

// WrapError wraps an existing error with additional context
func WrapError(err error, errorType ErrorType, code, message string) error {
	if err == nil {
		return nil
	}
	return NewErrorWithCause(errorType, code, message, err)
}

// GetErrorType extracts the error type from an error
func GetErrorType(err error) ErrorType {
	var customErr *CustomError
	if errors.As(err, &customErr) {
		return customErr.Type
	}
	return ErrorTypeInternal
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) string {
	var customErr *CustomError
	if errors.As(err, &customErr) {
		return customErr.Code
	}
	return "UNKNOWN"
}

// FormatError formats an error for display to the user
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	var customErr *CustomError
	if errors.As(err, &customErr) {
		return customErr.GetUserFriendlyMessage()
	}

	// For non-custom errors, provide a basic format
	return fmt.Sprintf("❌ %s", err.Error())
}

// NewValidationError creates a validation error
func NewValidationError(message string, suggestions ...string) *CustomError {
	return NewError(ErrorTypeValidation, "VALIDATION_ERROR", message).
		WithSuggestions(suggestions...)
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(message string, suggestions ...string) *CustomError {
	return NewError(ErrorTypeConfiguration, "CONFIGURATION_ERROR", message).
		WithSuggestions(suggestions...)
}

// NewAuthenticationError creates an authentication error
func NewAuthenticationError(message string, suggestions ...string) *CustomError {
	return NewError(ErrorTypeAuthentication, "AUTHENTICATION_ERROR", message).
		WithSuggestions(suggestions...)
}

// NewGCPError creates a GCP-related error
func NewGCPError(message string, suggestions ...string) *CustomError {
	return NewError(ErrorTypeGCP, "GCP_ERROR", message).
		WithSuggestions(suggestions...)
}

// NewGitHubError creates a GitHub-related error
func NewGitHubError(message string, suggestions ...string) *CustomError {
	return NewError(ErrorTypeGitHub, "GITHUB_ERROR", message).
		WithSuggestions(suggestions...)
}

// NewFileSystemError creates a file system error
func NewFileSystemError(message string, suggestions ...string) *CustomError {
	return NewError(ErrorTypeFileSystem, "FILESYSTEM_ERROR", message).
		WithSuggestions(suggestions...)
}

// NewInternalError creates an internal error
func NewInternalError(message string, cause error) *CustomError {
	return NewErrorWithCause(ErrorTypeInternal, "INTERNAL_ERROR", message, cause).
		WithSuggestions("Please report this issue to the developers")
}

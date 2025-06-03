// Package ui provides Bubble Tea interactive UI components
// for the GCP Workload Identity Federation CLI tool.
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Fordjour12/gcp-wif/internal/config"
	"github.com/Fordjour12/gcp-wif/internal/errors"
	"github.com/Fordjour12/gcp-wif/internal/logging"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Styles are now defined in styles.go to avoid duplication

// FieldType represents the type of form field
type FieldType int

const (
	FieldTypeText FieldType = iota
	FieldTypePassword
	FieldTypeOptional
	FieldTypeBoolean
)

// FieldValidationState represents the current validation state of a field
type FieldValidationState int

const (
	ValidationStateNone    FieldValidationState = iota // No validation performed yet
	ValidationStateValid                               // Field is valid
	ValidationStateInvalid                             // Field has validation errors
	ValidationStateWarning                             // Field has warnings but is valid
)

// FormField represents a single form field
type FormField struct {
	Key          string             // Configuration key
	Label        string             // Display label
	Description  string             // Field description
	Placeholder  string             // Placeholder text
	Required     bool               // Is field required
	Type         FieldType          // Field type
	Value        string             // Current value
	DefaultValue string             // Default value
	Validation   func(string) error // Validation function
	input        textinput.Model

	// Real-time validation fields
	validationState    FieldValidationState // Current validation state
	validationMessage  string               // Current validation message
	suggestionMessage  string               // Helpful suggestion for user
	lastValidationTime time.Time            // Last time validation was performed
	charCount          int                  // Current character count
	maxLength          int                  // Maximum allowed length (0 = no limit)
	minLength          int                  // Minimum required length (0 = no minimum)
}

// FormSection represents a group of related fields
type FormSection struct {
	Title       string
	Description string
	Fields      []FormField
}

// ValidationDebounceMsg is sent after debounce period for real-time validation
type ValidationDebounceMsg struct {
	fieldKey  string
	value     string
	timestamp time.Time
}

// InteractiveConfigForm represents the interactive configuration form
type InteractiveConfigForm struct {
	sections          []FormSection
	currentSectionIdx int
	currentFieldIdx   int
	complete          bool
	config            *config.Config
	validationErrors  map[string]string
	logger            *logging.Logger

	// Real-time validation
	validationDebounceInterval time.Duration
	lastKeypress               time.Time
}

// NewInteractiveConfigForm creates a new interactive configuration form
func NewInteractiveConfigForm(cfg *config.Config) *InteractiveConfigForm {
	form := &InteractiveConfigForm{
		sections:                   buildConfigSections(),
		config:                     cfg,
		validationErrors:           make(map[string]string),
		logger:                     logging.WithField("component", "interactive_form"),
		validationDebounceInterval: 300 * time.Millisecond, // 300ms debounce
	}

	// Initialize text inputs and set existing values
	form.initializeInputs()
	form.populateFromConfig()

	return form
}

// buildConfigSections creates the form sections based on the config structure
func buildConfigSections() []FormSection {
	return []FormSection{
		{
			Title:       "Project Configuration",
			Description: "Configure your Google Cloud Project settings",
			Fields: []FormField{
				{
					Key:         "project.id",
					Label:       "GCP Project ID",
					Description: "Your Google Cloud Project ID (6-30 chars, lowercase, digits, hyphens)",
					Placeholder: "my-gcp-project",
					Required:    true,
					Type:        FieldTypeText,
					Validation:  validateProjectID,
					minLength:   6,
					maxLength:   30,
				},
				{
					Key:          "project.region",
					Label:        "Default Region",
					Description:  "Default GCP region for resources",
					Placeholder:  "us-central1",
					Required:     false,
					Type:         FieldTypeOptional,
					DefaultValue: "us-central1",
				},
			},
		},
		{
			Title:       "GitHub Repository",
			Description: "Configure your GitHub repository settings",
			Fields: []FormField{
				{
					Key:         "repository.owner",
					Label:       "Repository Owner",
					Description: "GitHub username or organization name",
					Placeholder: "myusername",
					Required:    true,
					Type:        FieldTypeText,
					Validation:  validateGitHubOwner,
					maxLength:   39,
				},
				{
					Key:         "repository.name",
					Label:       "Repository Name",
					Description: "GitHub repository name",
					Placeholder: "my-app",
					Required:    true,
					Type:        FieldTypeText,
					Validation:  validateGitHubRepo,
					maxLength:   100,
				},
			},
		},
		{
			Title:       "Service Account",
			Description: "Configure the service account for GitHub Actions",
			Fields: []FormField{
				{
					Key:         "service_account.name",
					Label:       "Service Account Name",
					Description: "Name for the service account (6-30 chars, lowercase, digits, hyphens)",
					Placeholder: "github-actions-sa",
					Required:    true,
					Type:        FieldTypeText,
					Validation:  validateServiceAccountName,
					minLength:   6,
					maxLength:   30,
				},
				{
					Key:         "service_account.display_name",
					Label:       "Display Name",
					Description: "Human-readable display name",
					Placeholder: "GitHub Actions Service Account",
					Required:    false,
					Type:        FieldTypeOptional,
				},
			},
		},
		{
			Title:       "Workload Identity",
			Description: "Configure Workload Identity Federation settings",
			Fields: []FormField{
				{
					Key:         "workload_identity.pool_id",
					Label:       "Pool ID",
					Description: "Workload Identity Pool ID (3-32 chars, lowercase, digits, hyphens)",
					Placeholder: "github-pool",
					Required:    true,
					Type:        FieldTypeText,
					Validation:  validateWorkloadIdentityID,
					minLength:   3,
					maxLength:   32,
				},
				{
					Key:         "workload_identity.provider_id",
					Label:       "Provider ID",
					Description: "Workload Identity Provider ID (3-32 chars, lowercase, digits, hyphens)",
					Placeholder: "github-provider",
					Required:    true,
					Type:        FieldTypeText,
					Validation:  validateWorkloadIdentityID,
					minLength:   3,
					maxLength:   32,
				},
			},
		},
		{
			Title:       "Cloud Run (Optional)",
			Description: "Configure Cloud Run deployment settings",
			Fields: []FormField{
				{
					Key:         "cloud_run.service_name",
					Label:       "Service Name",
					Description: "Cloud Run service name (optional)",
					Placeholder: "my-app",
					Required:    false,
					Type:        FieldTypeOptional,
					Validation:  validateCloudRunServiceName,
				},
				{
					Key:          "cloud_run.region",
					Label:        "Region",
					Description:  "Cloud Run region",
					Placeholder:  "us-central1",
					Required:     false,
					Type:         FieldTypeOptional,
					DefaultValue: "us-central1",
				},
				{
					Key:         "cloud_run.registry",
					Label:       "Artifact Registry",
					Description: "Docker registry for container images (optional)",
					Placeholder: "us-central1-docker.pkg.dev/my-project/my-repo",
					Required:    false,
					Type:        FieldTypeOptional,
				},
			},
		},
		{
			Title:       "GitHub Actions Workflow",
			Description: "Configure the generated GitHub Actions workflow",
			Fields: []FormField{
				{
					Key:          "workflow.name",
					Label:        "Workflow Name",
					Description:  "Name for the GitHub Actions workflow",
					Placeholder:  "Deploy to Cloud Run",
					Required:     false,
					Type:         FieldTypeOptional,
					DefaultValue: "Deploy to Cloud Run",
				},
				{
					Key:          "workflow.filename",
					Label:        "Workflow Filename",
					Description:  "Workflow YAML filename (without .yml extension)",
					Placeholder:  "deploy",
					Required:     false,
					Type:         FieldTypeOptional,
					DefaultValue: "deploy",
				},
			},
		},
	}
}

// initializeInputs initializes all text inputs
func (m *InteractiveConfigForm) initializeInputs() {
	for sectionIdx := range m.sections {
		for fieldIdx := range m.sections[sectionIdx].Fields {
			field := &m.sections[sectionIdx].Fields[fieldIdx]

			ti := textinput.New()
			ti.Placeholder = field.Placeholder
			ti.CharLimit = 100
			ti.Width = 50

			if field.Type == FieldTypePassword {
				ti.EchoMode = textinput.EchoPassword
				ti.EchoCharacter = '*'
			}

			// Focus the first field
			if sectionIdx == 0 && fieldIdx == 0 {
				ti.Focus()
			}

			field.input = ti
		}
	}
}

// populateFromConfig populates form fields with existing config values
func (m *InteractiveConfigForm) populateFromConfig() {
	if m.config == nil {
		return
	}

	// Map config values to form fields
	fieldValues := map[string]string{
		"project.id":                    m.config.Project.ID,
		"project.region":                m.config.Project.Region,
		"repository.owner":              m.config.Repository.Owner,
		"repository.name":               m.config.Repository.Name,
		"service_account.name":          m.config.ServiceAccount.Name,
		"service_account.display_name":  m.config.ServiceAccount.DisplayName,
		"workload_identity.pool_id":     m.config.WorkloadIdentity.PoolID,
		"workload_identity.provider_id": m.config.WorkloadIdentity.ProviderID,
		"cloud_run.service_name":        m.config.CloudRun.ServiceName,
		"cloud_run.region":              m.config.CloudRun.Region,
		"cloud_run.registry":            m.config.CloudRun.Registry,
		"workflow.name":                 m.config.Workflow.Name,
		"workflow.filename":             m.config.Workflow.Filename,
	}

	// Set values in form fields
	for sectionIdx := range m.sections {
		for fieldIdx := range m.sections[sectionIdx].Fields {
			field := &m.sections[sectionIdx].Fields[fieldIdx]
			if value, exists := fieldValues[field.Key]; exists && value != "" {
				field.Value = value
				field.input.SetValue(value)
			} else if field.DefaultValue != "" {
				field.Value = field.DefaultValue
				field.input.SetValue(field.DefaultValue)
			}
		}
	}
}

// getCurrentField returns the currently active field
func (m *InteractiveConfigForm) getCurrentField() *FormField {
	if m.currentSectionIdx < len(m.sections) &&
		m.currentFieldIdx < len(m.sections[m.currentSectionIdx].Fields) {
		return &m.sections[m.currentSectionIdx].Fields[m.currentFieldIdx]
	}
	return nil
}

// Init initializes the form
func (m *InteractiveConfigForm) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the form state
func (m *InteractiveConfigForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			return m.handleEnter()

		case "tab":
			return m.handleTab()

		case "shift+tab":
			return m.handleShiftTab()
		}

		// Handle real-time validation for typing
		currentField := m.getCurrentField()
		if currentField != nil {
			// Update the input field
			currentField.input, cmd = currentField.input.Update(msg)
			oldValue := currentField.Value
			currentField.Value = currentField.input.Value()

			// Update character count
			currentField.charCount = len(currentField.Value)

			// Only trigger validation if the value actually changed
			if oldValue != currentField.Value {
				m.lastKeypress = time.Now()

				// Clear previous validation state for immediate feedback
				delete(m.validationErrors, currentField.Key)

				// Schedule debounced validation
				return m, tea.Batch(cmd, m.scheduleValidation(currentField.Key, currentField.Value))
			}
		}

	case ValidationDebounceMsg:
		// Handle debounced validation
		if currentField := m.getCurrentField(); currentField != nil {
			// Only validate if this is the most recent validation request
			if msg.fieldKey == currentField.Key && msg.value == currentField.Value &&
				time.Since(m.lastKeypress) >= m.validationDebounceInterval {
				m.performRealTimeValidation(currentField)
			}
		}
	}

	// Update the current input if not handled above
	currentField := m.getCurrentField()
	if currentField != nil && cmd == nil {
		currentField.input, cmd = currentField.input.Update(msg)
		currentField.Value = currentField.input.Value()
		currentField.charCount = len(currentField.Value)
	}

	return m, cmd
}

// handleEnter processes the Enter key
func (m *InteractiveConfigForm) handleEnter() (tea.Model, tea.Cmd) {
	currentField := m.getCurrentField()
	if currentField == nil {
		return m, nil
	}

	// Validate the current field
	if err := m.validateCurrentField(); err != nil {
		m.validationErrors[currentField.Key] = err.Error()
		return m, nil
	}

	// Move to next field or complete form
	return m.nextField()
}

// handleTab processes the Tab key (move to next field without validation)
func (m *InteractiveConfigForm) handleTab() (tea.Model, tea.Cmd) {
	currentField := m.getCurrentField()
	if currentField != nil {
		currentField.Value = currentField.input.Value()
	}
	return m.nextField()
}

// handleShiftTab processes Shift+Tab (move to previous field)
func (m *InteractiveConfigForm) handleShiftTab() (tea.Model, tea.Cmd) {
	currentField := m.getCurrentField()
	if currentField != nil {
		currentField.Value = currentField.input.Value()
	}
	return m.previousField()
}

// nextField moves to the next field or completes the form
func (m *InteractiveConfigForm) nextField() (tea.Model, tea.Cmd) {
	// Blur current field
	if currentField := m.getCurrentField(); currentField != nil {
		currentField.input.Blur()
	}

	// Move to next field
	m.currentFieldIdx++

	// Check if we need to move to next section
	if m.currentFieldIdx >= len(m.sections[m.currentSectionIdx].Fields) {
		m.currentSectionIdx++
		m.currentFieldIdx = 0

		// Check if form is complete
		if m.currentSectionIdx >= len(m.sections) {
			m.complete = true
			return m, tea.Quit
		}
	}

	// Focus new field
	if currentField := m.getCurrentField(); currentField != nil {
		currentField.input.Focus()
	}

	return m, textinput.Blink
}

// previousField moves to the previous field
func (m *InteractiveConfigForm) previousField() (tea.Model, tea.Cmd) {
	// Blur current field
	if currentField := m.getCurrentField(); currentField != nil {
		currentField.input.Blur()
	}

	// Move to previous field
	m.currentFieldIdx--

	// Check if we need to move to previous section
	if m.currentFieldIdx < 0 {
		if m.currentSectionIdx > 0 {
			m.currentSectionIdx--
			m.currentFieldIdx = len(m.sections[m.currentSectionIdx].Fields) - 1
		} else {
			m.currentFieldIdx = 0
		}
	}

	// Focus new field
	if currentField := m.getCurrentField(); currentField != nil {
		currentField.input.Focus()
	}

	return m, textinput.Blink
}

// validateCurrentField validates the current field
func (m *InteractiveConfigForm) validateCurrentField() error {
	currentField := m.getCurrentField()
	if currentField == nil {
		return nil
	}

	value := strings.TrimSpace(currentField.Value)

	// Check if required field is empty
	if currentField.Required && value == "" {
		return errors.NewValidationError(fmt.Sprintf("%s is required", currentField.Label))
	}

	// Run custom validation if provided
	if currentField.Validation != nil && value != "" {
		return currentField.Validation(value)
	}

	return nil
}

// scheduleValidation schedules a debounced validation for a field
func (m *InteractiveConfigForm) scheduleValidation(fieldKey, value string) tea.Cmd {
	return tea.Tick(m.validationDebounceInterval, func(t time.Time) tea.Msg {
		return ValidationDebounceMsg{
			fieldKey:  fieldKey,
			value:     value,
			timestamp: t,
		}
	})
}

// performRealTimeValidation performs real-time validation on a field
func (m *InteractiveConfigForm) performRealTimeValidation(field *FormField) {
	value := strings.TrimSpace(field.Value)
	field.lastValidationTime = time.Now()

	// Reset validation state
	field.validationState = ValidationStateNone
	field.validationMessage = ""
	field.suggestionMessage = ""

	// Check length constraints first
	if field.minLength > 0 && len(value) < field.minLength {
		field.validationState = ValidationStateInvalid
		field.validationMessage = fmt.Sprintf("Minimum %d characters required", field.minLength)
		field.suggestionMessage = fmt.Sprintf("Add %d more character(s)", field.minLength-len(value))
		return
	}

	if field.maxLength > 0 && len(value) > field.maxLength {
		field.validationState = ValidationStateInvalid
		field.validationMessage = fmt.Sprintf("Maximum %d characters allowed", field.maxLength)
		field.suggestionMessage = fmt.Sprintf("Remove %d character(s)", len(value)-field.maxLength)
		return
	}

	// Check if required field is empty
	if field.Required && value == "" {
		field.validationState = ValidationStateInvalid
		field.validationMessage = fmt.Sprintf("%s is required", field.Label)
		field.suggestionMessage = "This field cannot be empty"
		return
	}

	// Run custom validation if provided and field has content
	if field.Validation != nil && value != "" {
		if err := field.Validation(value); err != nil {
			field.validationState = ValidationStateInvalid
			field.validationMessage = err.Error()
			field.suggestionMessage = m.getValidationSuggestion(field.Key, value)
		} else {
			field.validationState = ValidationStateValid
			field.validationMessage = "Valid"
		}
	} else if value != "" {
		// Field has content and no validation errors
		field.validationState = ValidationStateValid
		field.validationMessage = "Valid"
	}

	// Add length hint as suggestion if approaching limits (80% of max length)
	if field.maxLength > 0 && len(value) > (field.maxLength*4)/5 {
		remaining := field.maxLength - len(value)
		if field.validationState == ValidationStateValid {
			field.validationState = ValidationStateWarning
		}
		if remaining > 0 {
			field.suggestionMessage = fmt.Sprintf("%d character(s) remaining", remaining)
		}
	}
}

// getValidationSuggestion provides helpful suggestions based on validation errors
func (m *InteractiveConfigForm) getValidationSuggestion(fieldKey, value string) string {
	switch fieldKey {
	case "project.id":
		if strings.Contains(value, "_") {
			return "Replace underscores with hyphens"
		}
		if strings.ToUpper(value) != strings.ToLower(value) {
			return "Use only lowercase letters"
		}
		if strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
			return "Remove leading/trailing hyphens"
		}
		if strings.Contains(value, "--") {
			return "Remove consecutive hyphens"
		}
		return "Must start with letter, use lowercase letters, digits, and hyphens"

	case "repository.owner", "repository.name":
		if strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
			return "Remove leading/trailing hyphens"
		}
		if strings.Contains(value, "--") {
			return "Avoid consecutive hyphens"
		}
		return "Use alphanumeric characters and hyphens only"

	case "service_account.name":
		if strings.Contains(value, "_") {
			return "Replace underscores with hyphens"
		}
		if strings.ToUpper(value) != strings.ToLower(value) {
			return "Use only lowercase letters"
		}
		return "Must start with letter, use lowercase letters, digits, and hyphens"

	case "workload_identity.pool_id", "workload_identity.provider_id":
		if strings.Contains(value, "_") {
			return "Replace underscores with hyphens"
		}
		if strings.ToUpper(value) != strings.ToLower(value) {
			return "Use only lowercase letters"
		}
		return "Must start with letter, use lowercase letters, digits, and hyphens"

	default:
		return "Check the format requirements"
	}
}

// View renders the form
func (m *InteractiveConfigForm) View() string {
	if m.complete {
		return m.renderComplete()
	}

	var b strings.Builder

	// Title
	b.WriteString(TitleStyle.Render("🚀 GCP Workload Identity Federation Setup"))
	b.WriteString("\n\n")

	// Current section info
	currentSection := m.sections[m.currentSectionIdx]
	b.WriteString(SectionStyle.Render(fmt.Sprintf("📋 %s", currentSection.Title)))
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render(currentSection.Description))
	b.WriteString("\n\n")

	// Progress indicator
	totalFields := 0
	currentFieldGlobal := 0
	for i, section := range m.sections {
		if i < m.currentSectionIdx {
			currentFieldGlobal += len(section.Fields)
		} else if i == m.currentSectionIdx {
			currentFieldGlobal += m.currentFieldIdx
		}
		totalFields += len(section.Fields)
	}
	progress := fmt.Sprintf("Progress: %d/%d fields (%d/%d sections)",
		currentFieldGlobal+1, totalFields, m.currentSectionIdx+1, len(m.sections))
	b.WriteString(HelpStyle.Render(progress))
	b.WriteString("\n\n")

	// Current field
	currentField := m.getCurrentField()
	if currentField != nil {
		b.WriteString(m.renderField(currentField))
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("Navigation: Enter=Next • Tab=Skip • Shift+Tab=Previous • Ctrl+C=Quit"))

	return b.String()
}

// renderField renders a single field with real-time validation feedback
func (m *InteractiveConfigForm) renderField(field *FormField) string {
	var b strings.Builder

	// Field label with validation indicator
	label := field.Label
	if field.Required {
		label += " *"
	}

	// Add validation state icon to label
	switch field.validationState {
	case ValidationStateValid:
		label = "✅ " + label
	case ValidationStateInvalid:
		label = "❌ " + label
	case ValidationStateWarning:
		label = "⚠️ " + label
	}

	b.WriteString(InputStyle.Render(label))
	b.WriteString("\n")

	// Field description
	if field.Description != "" {
		b.WriteString(HelpStyle.Render(field.Description))
		b.WriteString("\n")
	}

	// Character count and length limits
	if field.maxLength > 0 || field.minLength > 0 {
		var lengthInfo strings.Builder
		lengthInfo.WriteString(fmt.Sprintf("Length: %d", field.charCount))

		if field.minLength > 0 && field.maxLength > 0 {
			lengthInfo.WriteString(fmt.Sprintf(" (min: %d, max: %d)", field.minLength, field.maxLength))
		} else if field.minLength > 0 {
			lengthInfo.WriteString(fmt.Sprintf(" (min: %d)", field.minLength))
		} else if field.maxLength > 0 {
			lengthInfo.WriteString(fmt.Sprintf(" (max: %d)", field.maxLength))
		}

		// Color the length info based on validation state
		switch field.validationState {
		case ValidationStateValid:
			b.WriteString(SuccessStyle.Render(lengthInfo.String()))
		case ValidationStateInvalid:
			b.WriteString(ErrorStyle.Render(lengthInfo.String()))
		case ValidationStateWarning:
			b.WriteString(WarningStyle.Render(lengthInfo.String()))
		default:
			b.WriteString(HelpStyle.Render(lengthInfo.String()))
		}
		b.WriteString("\n")
	}

	// Input field
	b.WriteString(field.input.View())
	b.WriteString("\n")

	// Real-time validation message
	if field.validationMessage != "" {
		switch field.validationState {
		case ValidationStateValid:
			b.WriteString(SuccessStyle.Render(fmt.Sprintf("✅ %s", field.validationMessage)))
		case ValidationStateInvalid:
			b.WriteString(ErrorStyle.Render(fmt.Sprintf("❌ %s", field.validationMessage)))
		case ValidationStateWarning:
			b.WriteString(WarningStyle.Render(fmt.Sprintf("⚠️ %s", field.validationMessage)))
		}
		b.WriteString("\n")
	}

	// Helpful suggestion
	if field.suggestionMessage != "" {
		b.WriteString(HelpStyle.Render(fmt.Sprintf("💡 %s", field.suggestionMessage)))
		b.WriteString("\n")
	}

	// Legacy validation error (for backwards compatibility)
	if err, exists := m.validationErrors[field.Key]; exists {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("❌ %s", err)))
		b.WriteString("\n")
	}

	return b.String()
}

// renderComplete renders the completion screen
func (m *InteractiveConfigForm) renderComplete() string {
	var b strings.Builder

	b.WriteString(SuccessStyle.Render("✅ Configuration Complete!"))
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("All configuration fields have been collected."))
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("Your Workload Identity Federation setup will now proceed..."))

	return b.String()
}

// GetConfig returns the updated configuration
func (m *InteractiveConfigForm) GetConfig() *config.Config {
	// Update config with form values
	for _, section := range m.sections {
		for _, field := range section.Fields {
			m.setConfigValue(field.Key, field.Value)
		}
	}

	// Apply defaults and return
	m.config.SetDefaults()
	return m.config
}

// setConfigValue sets a configuration value based on the field key
func (m *InteractiveConfigForm) setConfigValue(key, value string) {
	if value == "" {
		return
	}

	switch key {
	case "project.id":
		m.config.Project.ID = value
	case "project.region":
		m.config.Project.Region = value
	case "repository.owner":
		m.config.Repository.Owner = value
	case "repository.name":
		m.config.Repository.Name = value
	case "service_account.name":
		m.config.ServiceAccount.Name = value
	case "service_account.display_name":
		m.config.ServiceAccount.DisplayName = value
	case "workload_identity.pool_id":
		m.config.WorkloadIdentity.PoolID = value
	case "workload_identity.provider_id":
		m.config.WorkloadIdentity.ProviderID = value
	case "cloud_run.service_name":
		m.config.CloudRun.ServiceName = value
	case "cloud_run.region":
		m.config.CloudRun.Region = value
	case "cloud_run.registry":
		m.config.CloudRun.Registry = value
	case "workflow.name":
		m.config.Workflow.Name = value
	case "workflow.filename":
		m.config.Workflow.Filename = value
	}
}

// IsComplete returns whether the form is complete
func (m *InteractiveConfigForm) IsComplete() bool {
	return m.complete
}

// RunInteractiveConfig runs the interactive configuration form
func RunInteractiveConfig(cfg *config.Config) (*config.Config, error) {
	logger := logging.WithField("function", "RunInteractiveConfig")
	logger.Info("Starting interactive configuration")

	form := NewInteractiveConfigForm(cfg)

	p := tea.NewProgram(form, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeInternal, "INTERACTIVE_FORM_FAILED",
			"Failed to run interactive configuration form")
	}

	if formModel, ok := finalModel.(*InteractiveConfigForm); ok {
		finalConfig := formModel.GetConfig()

		// Validate the final configuration
		result := finalConfig.ValidateSchema()
		if !result.Valid {
			logger.Error("Configuration validation failed", "errors", len(result.Errors))
			var errorMessages []string
			for _, valErr := range result.Errors {
				errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", valErr.Field, valErr.Message))
			}
			return nil, errors.NewValidationError(
				"Configuration validation failed",
				errorMessages...)
		}

		logger.Info("Interactive configuration completed successfully")
		return finalConfig, nil
	}

	return nil, errors.NewInternalError("Unexpected model type returned from interactive form", nil)
}

// Validation functions
func validateProjectID(value string) error {
	if len(value) < 6 || len(value) > 30 {
		return errors.NewValidationError("Project ID must be 6-30 characters long")
	}
	// Add more specific validation here
	return nil
}

func validateGitHubOwner(value string) error {
	if len(value) > 39 {
		return errors.NewValidationError("Repository owner cannot be longer than 39 characters")
	}
	// Add more specific validation here
	return nil
}

func validateGitHubRepo(value string) error {
	if len(value) > 100 {
		return errors.NewValidationError("Repository name cannot be longer than 100 characters")
	}
	// Add more specific validation here
	return nil
}

func validateServiceAccountName(value string) error {
	if len(value) < 6 || len(value) > 30 {
		return errors.NewValidationError("Service account name must be 6-30 characters long")
	}
	// Add more specific validation here
	return nil
}

func validateWorkloadIdentityID(value string) error {
	if len(value) < 3 || len(value) > 32 {
		return errors.NewValidationError("Workload Identity ID must be 3-32 characters long")
	}
	// Add more specific validation here
	return nil
}

func validateCloudRunServiceName(value string) error {
	if value != "" && len(value) > 63 {
		return errors.NewValidationError("Cloud Run service name cannot be longer than 63 characters")
	}
	// Add more specific validation here
	return nil
}

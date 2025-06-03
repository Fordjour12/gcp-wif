package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles are now defined in styles.go to avoid duplication

// ProgressStep represents a single step in the process
type ProgressStep struct {
	Name        string
	Description string
	Status      StepStatus
	Error       error
}

// StepStatus represents the status of a step
type StepStatus int

const (
	StepPending StepStatus = iota
	StepInProgress
	StepCompleted
	StepFailed
	StepSkipped
)

// ProgressModel represents the progress display model
type ProgressModel struct {
	steps    []ProgressStep
	current  int
	spinner  spinner.Model
	progress progress.Model
	complete bool
	failed   bool
}

// NewProgressModel creates a new progress model
func NewProgressModel(steps []ProgressStep) *ProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	p := progress.New(progress.WithDefaultGradient())
	p.Width = 50

	return &ProgressModel{
		steps:    steps,
		current:  0,
		spinner:  s,
		progress: p,
	}
}

// Init initializes the progress model
func (m *ProgressModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.progress.Init())
}

// Update handles messages and updates the progress state
func (m *ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if m.complete || m.failed {
				return m, tea.Quit
			}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case StepCompleteMsg:
		if m.current < len(m.steps) {
			m.steps[m.current].Status = StepCompleted
			m.current++
			if m.current >= len(m.steps) {
				m.complete = true
				return m, nil
			}
			m.steps[m.current].Status = StepInProgress
		}
		return m, m.spinner.Tick
	case StepFailedMsg:
		if m.current < len(m.steps) {
			m.steps[m.current].Status = StepFailed
			m.steps[m.current].Error = msg.Error
			m.failed = true
		}
		return m, nil
	case StepSkippedMsg:
		if m.current < len(m.steps) {
			m.steps[m.current].Status = StepSkipped
			m.current++
			if m.current >= len(m.steps) {
				m.complete = true
				return m, nil
			}
			m.steps[m.current].Status = StepInProgress
		}
		return m, m.spinner.Tick
	}

	return m, nil
}

// View renders the progress display
func (m *ProgressModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(TitleStyle.Render("Setting up Workload Identity Federation"))
	b.WriteString("\n\n")

	// Progress bar
	percent := float64(m.current) / float64(len(m.steps))
	if m.complete {
		percent = 1.0
	}
	b.WriteString(m.progress.ViewAs(percent))
	b.WriteString("\n\n")

	// Steps
	for _, step := range m.steps {
		var icon string
		var styleFunc func(...string) string
		switch step.Status {
		case StepPending:
			icon = "⏸"
			styleFunc = HelpStyle.Render
		case StepInProgress:
			icon = m.spinner.View()
			styleFunc = ProgressStyle.Render
		case StepCompleted:
			icon = "✓"
			styleFunc = StatusStyle.Render
		case StepFailed:
			icon = "✗"
			styleFunc = ErrorStyle.Render
		case StepSkipped:
			icon = "⊘"
			styleFunc = WarningStyle.Render
		}

		stepText := fmt.Sprintf("%s %s", icon, step.Name)
		if step.Description != "" && step.Status == StepInProgress {
			stepText += fmt.Sprintf(" - %s", step.Description)
		}

		b.WriteString(styleFunc(stepText))

		if step.Error != nil {
			b.WriteString("\n")
			b.WriteString(ErrorStyle.Render(fmt.Sprintf("   Error: %s", step.Error.Error())))
		}

		b.WriteString("\n")
	}

	// Status message
	if m.complete {
		b.WriteString("\n")
		b.WriteString(StatusStyle.Render("✓ Workload Identity Federation setup complete!"))
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("Press Enter to continue or Ctrl+C to quit"))
	} else if m.failed {
		b.WriteString("\n")
		b.WriteString(ErrorStyle.Render("✗ Setup failed. Please check the errors above."))
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("Press Enter to continue or Ctrl+C to quit"))
	}

	return b.String()
}

// IsComplete returns whether the progress is complete
func (m *ProgressModel) IsComplete() bool {
	return m.complete
}

// IsFailed returns whether the progress has failed
func (m *ProgressModel) IsFailed() bool {
	return m.failed
}

// StepCompleteMsg indicates a step has completed
type StepCompleteMsg struct{}

// StepFailedMsg indicates a step has failed
type StepFailedMsg struct {
	Error error
}

// StepSkippedMsg indicates a step was skipped
type StepSkippedMsg struct{}

// RunProgressDisplay runs the progress display
func RunProgressDisplay(steps []ProgressStep) (*ProgressModel, error) {
	model := NewProgressModel(steps)
	if len(steps) > 0 {
		steps[0].Status = StepInProgress
	}

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run progress display: %w", err)
	}

	if progressModel, ok := finalModel.(*ProgressModel); ok {
		return progressModel, nil
	}

	return nil, fmt.Errorf("unexpected model type")
}

// DefaultWIFSteps returns the default steps for WIF setup
func DefaultWIFSteps() []ProgressStep {
	return []ProgressStep{
		{
			Name:        "Validate Configuration",
			Description: "Checking configuration parameters",
			Status:      StepPending,
		},
		{
			Name:        "Authenticate with GCP",
			Description: "Verifying gcloud authentication",
			Status:      StepPending,
		},
		{
			Name:        "Create Service Account",
			Description: "Creating service account for GitHub Actions",
			Status:      StepPending,
		},
		{
			Name:        "Grant IAM Roles",
			Description: "Assigning necessary permissions",
			Status:      StepPending,
		},
		{
			Name:        "Create Workload Identity Pool",
			Description: "Setting up workload identity pool",
			Status:      StepPending,
		},
		{
			Name:        "Create Workload Identity Provider",
			Description: "Configuring GitHub OIDC provider",
			Status:      StepPending,
		},
		{
			Name:        "Bind Service Account",
			Description: "Linking service account to workload identity",
			Status:      StepPending,
		},
		{
			Name:        "Generate GitHub Workflow",
			Description: "Creating GitHub Actions workflow file",
			Status:      StepPending,
		},
	}
}

// SimpleProgress displays a simple progress message
func SimpleProgress(message string) {
	fmt.Printf("⏳ %s...\n", message)
}

// SimpleSuccess displays a simple success message
func SimpleSuccess(message string) {
	fmt.Printf("✓ %s\n", StatusStyle.Render(message))
}

// SimpleError displays a simple error message
func SimpleError(message string) {
	fmt.Printf("✗ %s\n", ErrorStyle.Render(message))
}

// SimpleWarning displays a simple warning message
func SimpleWarning(message string) {
	fmt.Printf("⚠ %s\n", WarningStyle.Render(message))
}

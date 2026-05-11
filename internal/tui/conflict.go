package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skillops/internal/config"
	"skillops/internal/skills"
	"skillops/internal/utils"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// Conflict represents a symlink name collision between multiple skill identities.
type Conflict struct {
	SymlinkName string   // The symlink name that conflicts
	Identities  []string // The identities that share this symlink name
}

// DetectConflicts checks a list of skill identities for symlink name collisions.
// It considers custom names from localConfig.SymlinkNames — if an identity has a
// custom name, that name is used instead of the default short name.
// Returns all conflicts (not just the first one).
func DetectConflicts(identities []string, localConfig config.LocalConfig) []Conflict {
	// Build symlink name map: symlinkName -> []identity
	symlinkMap := map[string][]string{}

	for _, identity := range identities {
		parsed, err := skills.ParseIdentity(identity)
		if err != nil {
			// Skip invalid identities
			continue
		}

		// Check if has custom name
		symlinkName := localConfig.SymlinkNames[identity]
		if symlinkName == "" {
			symlinkName = parsed.ShortName
		}

		symlinkMap[symlinkName] = append(symlinkMap[symlinkName], identity)
	}

	// Identify conflicts (entries with more than one identity)
	var conflicts []Conflict
	for symlinkName, identityList := range symlinkMap {
		if len(identityList) > 1 {
			conflicts = append(conflicts, Conflict{
				SymlinkName: symlinkName,
				Identities:  identityList,
			})
		}
	}

	return conflicts
}

// ConflictResolutionModel is a bubbletea model for resolving symlink name conflicts.
// It displays conflicting identities and provides input fields for custom names.
type ConflictResolutionModel struct {
	conflicts     []Conflict
	inputs        []textinput.Model // one input per conflicting identity
	identities    []string          // flat list of all conflicting identities (in order)
	focusIndex    int               // which input is focused
	validationErr string            // current validation error message
	customNames   map[string]string // result: identity -> custom name
	quitting      bool
	cancelled     bool
}

// NewConflictResolutionModel creates a new conflict resolution TUI model.
// It flattens all conflicts into a list of identities and creates one text input per identity.
func NewConflictResolutionModel(conflicts []Conflict) ConflictResolutionModel {
	var identities []string
	var inputs []textinput.Model

	counter := map[string]int{} // track numbering per symlink name

	for _, c := range conflicts {
		for _, identity := range c.Identities {
			identities = append(identities, identity)

			ti := textinput.New()
			// Suggest a name based on short name + counter
			counter[c.SymlinkName]++
			suggested := fmt.Sprintf("%s-%d", c.SymlinkName, counter[c.SymlinkName])
			ti.Placeholder = suggested
			ti.CharLimit = 64
			ti.Width = 40

			inputs = append(inputs, ti)
		}
	}

	// Focus the first input
	if len(inputs) > 0 {
		inputs[0].Focus()
	}

	return ConflictResolutionModel{
		conflicts:   conflicts,
		inputs:      inputs,
		identities:  identities,
		focusIndex:  0,
		customNames: make(map[string]string),
	}
}

// Init returns the initial command (textinput blink).
func (m ConflictResolutionModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the conflict resolution model.
func (m ConflictResolutionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.quitting = true
			return m, tea.Quit

		case "tab", "down":
			// Focus next input
			m.inputs[m.focusIndex].Blur()
			m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
			m.inputs[m.focusIndex].Focus()
			return m, textinput.Blink

		case "shift+tab", "up":
			// Focus previous input
			m.inputs[m.focusIndex].Blur()
			m.focusIndex = (m.focusIndex - 1 + len(m.inputs)) % len(m.inputs)
			m.inputs[m.focusIndex].Focus()
			return m, textinput.Blink

		case "enter":
			// Validate all inputs and submit
			if err := m.validate(); err != "" {
				m.validationErr = err
				return m, nil
			}
			// Build custom names map
			for i, identity := range m.identities {
				name := strings.TrimSpace(m.inputs[i].Value())
				if name == "" {
					name = m.inputs[i].Placeholder
				}
				m.customNames[identity] = name
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	// Forward to focused input
	var cmd tea.Cmd
	m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)

	// Clear validation error on typing
	m.validationErr = ""

	return m, cmd
}

// validate checks all input values and returns an error message or empty string.
func (m *ConflictResolutionModel) validate() string {
	seen := map[string]bool{}

	for i, input := range m.inputs {
		name := strings.TrimSpace(input.Value())
		if name == "" {
			name = input.Placeholder
		}

		// Check basic validation (path separators, empty, dots)
		if err := validateCustomName(name); err != "" {
			return fmt.Sprintf("Field %d (%s): %s", i+1, m.identities[i], err)
		}

		// Check for duplicates
		if seen[name] {
			return fmt.Sprintf("Duplicate name: %q is used more than once", name)
		}
		seen[name] = true
	}

	return ""
}

// validateCustomName checks a single custom name for validity.
func validateCustomName(name string) string {
	if name == "" {
		return "name cannot be empty"
	}
	if name == "." || name == ".." {
		return "name cannot be \".\" or \"..\""
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "name cannot contain path separators (/ or \\)"
	}
	// Also use the shared validator for additional checks
	if err := utils.ValidateName(name); err != nil {
		return err.Error()
	}
	return ""
}

// View renders the conflict resolution TUI.
func (m ConflictResolutionModel) View() string {
	if m.quitting {
		return ""
	}

	content := TitleStyle.Render(" SYMLINK CONFLICT DETECTED ") + "\n\n"

	inputIdx := 0
	for _, c := range m.conflicts {
		content += InfoStyle.Render(fmt.Sprintf("  Multiple skills resolve to the same symlink name: %s", c.SymlinkName)) + "\n"
		content += InfoStyle.Render("  Please provide custom names for each skill:") + "\n\n"

		for _, identity := range c.Identities {
			// Show identity
			content += DimStyle.Render(fmt.Sprintf("  %s", identity)) + "\n"

			// Show input field with label
			prefix := "  Symlink name: "
			if inputIdx == m.focusIndex {
				prefix = "> Symlink name: "
			}
			content += fmt.Sprintf("%s%s\n\n", prefix, m.inputs[inputIdx].View())
			inputIdx++
		}
	}

	// Show validation error
	if m.validationErr != "" {
		content += "\n" + SuccessStyle.Copy().Foreground(Pink).Render("  ⚠ "+m.validationErr) + "\n"
	}

	content += HelpStyle.Render("\n  [Tab] Next field  [Shift+Tab] Previous  [Enter] Confirm  [Esc] Cancel")

	return BorderStyle.Render(content) + "\n"
}

// CustomNames returns the resolved custom names map (identity -> custom name).
func (m ConflictResolutionModel) CustomNames() map[string]string {
	return m.customNames
}

// Cancelled returns true if the user cancelled the conflict resolution.
func (m ConflictResolutionModel) Cancelled() bool {
	return m.cancelled
}

// RunConflictResolution launches the conflict resolution TUI and returns
// the custom names map or an error if the user cancelled.
func RunConflictResolution(conflicts []Conflict) (map[string]string, error) {
	model := NewConflictResolutionModel(conflicts)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("conflict resolution TUI error: %w", err)
	}

	result := finalModel.(ConflictResolutionModel)
	if result.Cancelled() {
		return nil, fmt.Errorf("conflict resolution cancelled by user")
	}

	return result.CustomNames(), nil
}

// IsTerminal checks if stdin is a terminal (TTY).
// Returns false in CI, SSH piped, or non-interactive environments.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// FormatConflictError formats a descriptive error message for non-TTY environments
// when symlink conflicts are detected. It lists all conflicts and suggests manual resolution.
func FormatConflictError(conflicts []Conflict) error {
	var b strings.Builder

	b.WriteString("Error: Symlink conflicts detected (non-interactive mode)\n")
	b.WriteString("\nThe following skills have conflicting symlink names:\n")

	for _, c := range conflicts {
		b.WriteString(fmt.Sprintf("\nSymlink name: %s\n", c.SymlinkName))
		for _, identity := range c.Identities {
			b.WriteString(fmt.Sprintf("  - %s\n", identity))
		}
	}

	b.WriteString("\nTo resolve, add custom symlink names to .skillops/config.json:\n\n")

	// Build example config snippet
	b.WriteString("{\n")
	b.WriteString("  \"symlink_names\": {\n")

	// Collect all entries for the example snippet
	type entry struct {
		identity string
		name     string
	}
	var entries []entry
	for _, c := range conflicts {
		for _, identity := range c.Identities {
			suggestedName := suggestCustomName(identity, c.SymlinkName)
			entries = append(entries, entry{identity: identity, name: suggestedName})
		}
	}

	for i, e := range entries {
		comma := ","
		if i == len(entries)-1 {
			comma = ""
		}
		b.WriteString(fmt.Sprintf("    %q: %q%s\n", e.identity, e.name, comma))
	}

	b.WriteString("  }\n")
	b.WriteString("}\n")

	b.WriteString("\nThen run: skillops sync\n")

	return fmt.Errorf("%s", b.String())
}

// suggestCustomName generates a suggested custom symlink name for a conflicting identity.
// It uses the short name combined with a distinguishing path component.
func suggestCustomName(identity, shortName string) string {
	parts := strings.Split(identity, "/")
	if len(parts) < 3 {
		return shortName + "-custom"
	}

	// Use the component just before the short name as a distinguishing suffix
	// e.g., "github.com/company-a/utils/tools/logger" → "logger-tools"
	// e.g., "github.com/company-b/helpers/services/logger" → "logger-services"
	preceding := parts[len(parts)-2]

	// If the preceding component is generic (like "skills"), try the one before that
	genericNames := map[string]bool{"skills": true, "tools": true, "services": true, "src": true, "lib": true}
	if genericNames[preceding] && len(parts) >= 4 {
		preceding = parts[len(parts)-3]
	}

	suggested := shortName + "-" + preceding
	// Validate the suggested name
	if err := utils.ValidateName(suggested); err != nil {
		// Fallback to a simple suffix
		return shortName + "-" + filepath.Base(parts[1])
	}
	return suggested
}

// HandleConflicts handles symlink conflicts based on the environment.
// In TTY mode, it launches the interactive conflict resolution TUI.
// In non-TTY mode, it returns a descriptive error with manual resolution instructions.
func HandleConflicts(conflicts []Conflict) (map[string]string, error) {
	if !IsTerminal() {
		return nil, FormatConflictError(conflicts)
	}
	return RunConflictResolution(conflicts)
}

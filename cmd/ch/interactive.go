package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type step int

const (
	stepType step = iota
	stepScope
	stepSubject
	stepBody
	stepIsBreaking
	stepBreakingBody
	stepConfirm
)

type model struct {
	currentStep     step
	inputs          []textinput.Model
	commitType      string
	commitScope     string
	commitSubject   string
	commitBody      string
	isBreaking      bool
	breakingMessage string
	quitting        bool
	confirmed       bool
	err             error
	helpVisible     bool
}

func initialModel() model {
	// We'll start with 5 inputs, and add the 6th one dynamically if needed.
	inputs := make([]textinput.Model, 5)

	inputs[stepType] = textinput.New()
	inputs[stepType].Placeholder = "feat"
	inputs[stepType].Focus()
	inputs[stepType].Prompt = "│ "
	inputs[stepType].CharLimit = 20

	inputs[stepScope] = textinput.New()
	inputs[stepScope].Placeholder = "api, auth, db..."
	inputs[stepScope].CharLimit = 50

	inputs[stepSubject] = textinput.New()
	inputs[stepSubject].Placeholder = "A short, imperative tense description"
	inputs[stepSubject].CharLimit = 100

	inputs[stepBody] = textinput.New()
	inputs[stepBody].Placeholder = "Provide a longer description of the change (optional)"

	inputs[stepIsBreaking] = textinput.New()
	inputs[stepIsBreaking].Placeholder = "y/N"
	inputs[stepIsBreaking].CharLimit = 1

	return model{
		currentStep: stepType,
		inputs:      inputs,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.Type == tea.KeyCtrlC {
			m.confirmed = false
			m.quitting = true
			return m, tea.Quit
		}
	}

	if m.helpVisible {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "?", "esc":
				m.helpVisible = false
			}
		}
		return m, nil
	}

	switch m.currentStep {
	case stepConfirm:
		return m.updateConfirm(msg)
	default:
		return m.updateInputs(msg)
	}
}

func (m *model) updateInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "?" {
			m.helpVisible = true
			return m, nil
		}

		switch key.Type {
		case tea.KeyEnter:
			if err := m.validateCurrentStep(); err != nil {
				m.err = err
				return m, nil
			}
			m.err = nil

			m.saveCurrentStep()

			// Special logic for breaking change step
			if m.currentStep == stepIsBreaking {
				if m.isBreaking {
					m.currentStep = stepBreakingBody
					// Lazily create the breaking body input only when needed
					if len(m.inputs) <= int(stepBreakingBody) {
						ti := textinput.New()
						ti.Placeholder = "Describe the breaking change (optional)"
						m.inputs = append(m.inputs, ti)
					}
				} else {
					m.currentStep = stepConfirm // Skip breaking body
				}
			} else {
				m.currentStep++
			}

			if m.currentStep >= stepConfirm {
				m.currentStep = stepConfirm
				return m, nil
			}

			cmd := m.inputs[m.currentStep].Focus()
			cmds = append(cmds, cmd)

		case tea.KeyEsc:
			m.err = nil
			if m.currentStep > 0 {
				// Special case to jump back from breaking body to breaking question
				if m.currentStep == stepBreakingBody {
					m.currentStep = stepIsBreaking
				} else {
					m.currentStep--
				}
				cmd := m.inputs[m.currentStep].Focus()
				cmds = append(cmds, cmd)
			}
		}
	}

	var cmd tea.Cmd
	m.inputs[m.currentStep], cmd = m.inputs[m.currentStep].Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) validateCurrentStep() error {
	val := m.inputs[m.currentStep].Value()
	switch m.currentStep {
	case stepType:
		if !isAllowedType(val, allowedTypes) {
			return fmt.Errorf("invalid type. Choose from: %s", strings.Join(allowedTypes, ", "))
		}
	case stepSubject:
		if strings.TrimSpace(val) == "" {
			return fmt.Errorf("subject cannot be empty")
		}
	case stepIsBreaking:
		v := strings.ToLower(val)
		if v != "y" && v != "n" && v != "" {
			return fmt.Errorf("please enter 'y' or 'n'")
		}
	}
	return nil
}

func (m *model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "y", "Y":
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		case "n", "N", "esc":
			m.currentStep = stepType
			m.err = nil
			return m, m.inputs[stepType].Focus()
		}
	}
	return m, nil
}

func (m *model) saveCurrentStep() {
	switch m.currentStep {
	case stepType:
		m.commitType = m.inputs[stepType].Value()
	case stepScope:
		m.commitScope = m.inputs[stepScope].Value()
	case stepSubject:
		m.commitSubject = m.inputs[stepSubject].Value()
	case stepBody:
		m.commitBody = m.inputs[stepBody].Value()
	case stepIsBreaking:
		m.isBreaking = strings.ToLower(m.inputs[stepIsBreaking].Value()) == "y"
	case stepBreakingBody:
		m.breakingMessage = m.inputs[stepBreakingBody].Value()
	}
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Conventional Commit Builder") + "\n\n")

	if m.helpVisible {
		b.WriteString(helpBoxStyle.Render(getHelpText(m.currentStep)))
		b.WriteString("\n" + faintStyle.Render("Press '?' or 'esc' to return to the form."))
		return b.String()
	}

	fmt.Fprintf(&b, "1. Type:             %s\n", m.inputs[stepType].View())
	fmt.Fprintf(&b, "2. Scope:            %s\n", m.inputs[stepScope].View())
	fmt.Fprintf(&b, "3. Subject:          %s\n", m.inputs[stepSubject].View())
	fmt.Fprintf(&b, "4. Body:             %s\n", m.inputs[stepBody].View())
	fmt.Fprintf(&b, "5. Breaking Change?   %s\n", m.inputs[stepIsBreaking].View())

	if m.isBreaking && m.currentStep >= stepBreakingBody {
		// Ensure the input field exists before trying to view it
		if len(m.inputs) > int(stepBreakingBody) {
			fmt.Fprintf(&b, "6. Breaking Description: %s\n", m.inputs[stepBreakingBody].View())
		}
	}

	if m.currentStep == stepConfirm {
		finalMessage := m.constructCommitMessage()
		b.WriteString("\n" + guidelineBoxStyle.Render(finalMessage))
		b.WriteString("\n\n" + successStyle.Render("Write this commit message? (Y/n)"))
	} else {
		if m.err != nil {
			b.WriteString("\n" + errorStyle.Render(m.err.Error()))
		}
		b.WriteString("\n" + faintStyle.Render("?: help") + " · " + faintStyle.Render("Enter: next") + " · " + faintStyle.Render("Esc: back") + " · " + faintStyle.Render("Ctrl+C: quit"))
	}

	return b.String()
}

func getHelpText(s step) string {
	switch s {
	case stepType:
		return "The 'type' describes the kind of change you're making.\n\n" +
			"Common types:\n" +
			"  • " + codeStyle.Render("feat") + ": A new feature for the user.\n" +
			"  • " + codeStyle.Render("fix") + ": A bug fix for the user.\n" +
			"  • " + codeStyle.Render("chore") + ": Routine tasks, maintenance, or dependency updates."
	case stepScope:
		return "The 'scope' provides context for the change.\n\n" +
			"It's an optional noun describing the section of the codebase affected."
	case stepSubject:
		return "The 'subject' is a short, imperative summary of the change.\n\n" +
			"Rules:\n" +
			"  • Use the imperative, present tense: \"add\" not \"added\" or \"adds\".\n" +
			"  • Don't capitalize the first letter.\n" +
			"  • Don't end with a period."
	case stepBody:
		return "The 'body' provides additional context and details.\n\n" +
			"Use it to explain *what* and *why* vs. *how*."
	case stepIsBreaking:
		return "Mark a commit as a 'breaking change' if it introduces a change that is not backward-compatible.\n\n" +
			"This will add a '!' to the commit header and optionally a 'BREAKING CHANGE:' footer."
	case stepBreakingBody:
		return "Provide a clear description of the breaking change.\n\n" +
			"This is optional. If you leave it blank, the '!' in the header will still mark this as a breaking change."
	default:
		return "No help available for this step."
	}
}

func (m *model) constructCommitMessage() string {
	var msg strings.Builder
	msg.WriteString(m.commitType)
	if m.commitScope != "" {
		msg.WriteString(fmt.Sprintf("(%s)", m.commitScope))
	}
	if m.isBreaking {
		msg.WriteString("!")
	}
	msg.WriteString(": ")
	msg.WriteString(m.commitSubject)

	if m.commitBody != "" {
		msg.WriteString("\n\n")
		msg.WriteString(m.commitBody)
	}

	if m.isBreaking && m.breakingMessage != "" {
		msg.WriteString("\n\nBREAKING CHANGE: ")
		msg.WriteString(m.breakingMessage)
	}
	return msg.String()
}

func runInteractiveMode() (string, error) {
	m := initialModel()
	p := tea.NewProgram(&m, tea.WithOutput(os.Stderr))

	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running program: %w", err)
	}

	finalM, ok := finalModel.(*model)
	if !ok {
		return "", fmt.Errorf("could not cast final model")
	}

	if finalM.confirmed {
		return finalM.constructCommitMessage(), nil
	}

	return "", fmt.Errorf("interactive mode cancelled by user")
}

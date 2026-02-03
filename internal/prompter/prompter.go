package prompter

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/triptechtravel/clickup-cli/internal/iostreams"
)

// Prompter provides interactive prompts.
type Prompter struct {
	ios *iostreams.IOStreams
}

// New creates a new Prompter.
func New(ios *iostreams.IOStreams) *Prompter {
	return &Prompter{ios: ios}
}

// Input prompts for text input.
func (p *Prompter) Input(message, defaultValue string) (string, error) {
	if !p.ios.IsTerminal() {
		return "", fmt.Errorf("cannot prompt in non-interactive mode")
	}
	var result string
	err := survey.AskOne(&survey.Input{
		Message: message,
		Default: defaultValue,
	}, &result)
	return result, err
}

// Password prompts for secret input (hidden).
func (p *Prompter) Password(message string) (string, error) {
	if !p.ios.IsTerminal() {
		return "", fmt.Errorf("cannot prompt in non-interactive mode")
	}
	var result string
	err := survey.AskOne(&survey.Password{
		Message: message,
	}, &result)
	return result, err
}

// Select prompts the user to choose from a list.
func (p *Prompter) Select(message string, options []string) (int, error) {
	if !p.ios.IsTerminal() {
		return -1, fmt.Errorf("cannot prompt in non-interactive mode")
	}
	var idx int
	err := survey.AskOne(&survey.Select{
		Message: message,
		Options: options,
	}, &idx)
	return idx, err
}

// Confirm prompts for a yes/no answer.
func (p *Prompter) Confirm(message string, defaultValue bool) (bool, error) {
	if !p.ios.IsTerminal() {
		return false, fmt.Errorf("cannot prompt in non-interactive mode")
	}
	var result bool
	err := survey.AskOne(&survey.Confirm{
		Message: message,
		Default: defaultValue,
	}, &result)
	return result, err
}

// Editor opens the user's editor for multi-line input.
func (p *Prompter) Editor(message, defaultValue, filename string) (string, error) {
	if !p.ios.IsTerminal() {
		return "", fmt.Errorf("cannot prompt in non-interactive mode")
	}
	var result string
	err := survey.AskOne(&survey.Editor{
		Message:       message,
		Default:       defaultValue,
		FileName:      filename,
		HideDefault:   true,
		AppendDefault: true,
	}, &result)
	return result, err
}

// MultiSelect prompts the user to choose multiple items from a list.
func (p *Prompter) MultiSelect(message string, options []string) ([]int, error) {
	if !p.ios.IsTerminal() {
		return nil, fmt.Errorf("cannot prompt in non-interactive mode")
	}
	var indices []int
	err := survey.AskOne(&survey.MultiSelect{
		Message: message,
		Options: options,
	}, &indices)
	return indices, err
}

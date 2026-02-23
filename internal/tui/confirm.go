package tui

import "github.com/charmbracelet/huh"

// ConfirmDestructive prompts the user for confirmation of a destructive action.
// Returns true in non-TTY environments (don't block CI).
func ConfirmDestructive(message string) bool {
	if !IsTTY() {
		return true
	}

	var confirmed bool
	err := huh.NewConfirm().
		Title(message).
		Affirmative("Yes").
		Negative("No").
		Value(&confirmed).
		WithTheme(huh.ThemeDracula()).
		Run()
	if err != nil {
		return false
	}
	return confirmed
}

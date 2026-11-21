// Package ui wraps huh and lipgloss into the small set of interactions this
// tool needs: styled status lines, a bordered "go do this in Buzz Desktop"
// pause, and confirmations that abort the whole run on Ctrl-C or "no".
package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	dimStyle   = lipgloss.NewStyle().Faint(true)
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)

	actionBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")).
			Padding(1, 2).
			MarginTop(1)

	actionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("214"))

	// Blue, not amber: a read-only screen must be distinguishable from one
	// that is waiting on the user without reading a word of either.
	infoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(1, 2).
			MarginTop(1)

	infoTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))
)

// Screen badges. Every screen carries one so the user always knows, without
// reading the body, whether it is telling them something or waiting on them.
// The single most confusing thing about a guided CLI is not knowing which.
const (
	BadgeRead    = "READ  ·  nothing to do yet"
	BadgeAction  = "ACTION NEEDED  ·  do this in Buzz Desktop, then come back"
	BadgeInput   = "YOUR INPUT  ·  type or paste below"
	BadgeConfirm = "CONFIRM  ·  this one writes to disk"
)

// Aborted is returned by any step the user declined or Ctrl-C'd out of.
// Callers should treat it as "stop, nothing was changed" rather than a bug.
type Aborted struct{ Reason string }

func (a Aborted) Error() string { return a.Reason }

// Heading prints the top-of-run banner.
func Heading(title string) {
	fmt.Println()
	fmt.Println(titleStyle.Render(title))
}

// Step prints a numbered section header within a multi-step flow.
func Step(n, total int, title string) {
	fmt.Println()
	fmt.Println(titleStyle.Render(fmt.Sprintf("── STEP %d/%d · %s", n, total, title)))
}

// Ok prints a success line.
func Ok(msg string) { fmt.Println("  " + okStyle.Render("✓") + " " + msg) }

// Info prints a plain indented line.
func Info(msg string) { fmt.Println("  " + msg) }

// Dim renders text faint, for inline use inside an Info/Ok line.
func Dim(msg string) string { return dimStyle.Render(msg) }

// Bold renders text bold, for inline use inside an Info/Ok line.
func Bold(msg string) string { return titleStyle.Render(msg) }

// Warn prints a warning line.
func Warn(msg string) { fmt.Println("  " + warnStyle.Render("!") + " " + msg) }

// Die prints a fatal message to stderr and exits 1.
func Die(msg string) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, errStyle.Render("STOPPED:")+" "+msg)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

// ActionPause shows a bordered "your turn" block describing what the human
// must do in Buzz Desktop right now, then blocks until they confirm it's
// done. Mirrors the original script's "press ENTER when done" pause: the
// continue button is pre-selected, so a bare Enter advances. Cancel is still
// one arrow-key away. Returns Aborted if they cancel or Ctrl-C.
func ActionPause(lines []string) error {
	body := strings.Join(lines, "\n")
	box := actionBoxStyle.Render(actionTitleStyle.Render(BadgeAction) + "\n\n" + body)
	fmt.Println(box)
	return confirm("Done with that?", true)
}

// InfoBox shows something the user only needs to read. Styled distinctly from
// ActionPause so "read this" and "do this" are never confused at a glance.
func InfoBox(lines []string) {
	body := strings.Join(lines, "\n")
	fmt.Println(infoBoxStyle.Render(infoTitleStyle.Render(BadgeRead) + "\n\n" + body))
}

// Confirm asks a yes/no question that gates a real write (registering an
// agent, restoring a backup). Defaults to "no", matching the original
// script's [y/N] prompts — a bare Enter here must never take the
// destructive path. Returns Aborted on "no" or Ctrl-C.
func Confirm(question string) error {
	return confirm(question, false)
}

func confirm(question string, defaultYes bool) error {
	proceed := defaultYes
	err := huh.NewConfirm().
		Title(question).
		Affirmative("Yes, continue").
		Negative("Cancel").
		Value(&proceed).
		Run()
	if err != nil {
		return Aborted{Reason: "aborted — nothing was changed"}
	}
	if !proceed {
		return Aborted{Reason: "declined — nothing was changed"}
	}
	return nil
}

// Select asks the user to pick one of several options, returning the chosen
// value. Aborted on Ctrl-C.
func Select[T comparable](title string, options ...huh.Option[T]) (T, error) {
	var choice T
	err := huh.NewSelect[T]().
		Title(title).
		Options(options...).
		Value(&choice).
		Run()
	if err != nil {
		return choice, Aborted{Reason: "aborted — nothing was changed"}
	}
	return choice, nil
}

// Input asks for one line of text. Aborted on Ctrl-C.
func Input(title, placeholder string) (string, error) {
	var value string
	err := huh.NewInput().
		Title(BadgeInput + "\n\n" + title).
		Placeholder(placeholder).
		Value(&value).
		Validate(huh.ValidateNotEmpty()).
		Run()
	if err != nil {
		return "", Aborted{Reason: "aborted — nothing was changed"}
	}
	return value, nil
}

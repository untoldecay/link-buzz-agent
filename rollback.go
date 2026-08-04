package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/untoldecay/link-buzz-agent/internal/registry"
	"github.com/untoldecay/link-buzz-agent/internal/ui"
)

func runRollback(path, which string) error {
	backups := registry.BackupsFor(path)
	if len(backups) == 0 {
		return fmt.Errorf("no backups found next to %s", path)
	}

	var chosen string
	switch {
	case which != "":
		chosen = filepath.Join(filepath.Dir(path), which)
		if !contains(backups, chosen) {
			fmt.Println()
			fmt.Println(ui.Bold("Available backups:"))
			for _, b := range backups {
				fmt.Println("  " + filepath.Base(b))
			}
			return fmt.Errorf("%s is not one of them", which)
		}
	case len(backups) == 1:
		chosen = backups[0]
	default:
		var options []huh.Option[string]
		for i, b := range backups {
			label := filepath.Base(b) + "  →  " + strings.Join(summarize(b), ", ")
			if i == 0 {
				label += "  (newest)"
			}
			options = append(options, huh.NewOption(label, b))
		}
		selected, err := ui.Select("Multiple backups exist — restore which one?", options...)
		if err != nil {
			return err
		}
		chosen = selected
	}

	nowRows, err := registry.LoadStore(path)
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Println(ui.Bold("Now:") + "      " + strings.Join(registry.NamesIn(nowRows), ", "))
	fmt.Println(ui.Bold("Restore:") + "  " + strings.Join(summarize(chosen), ", "))
	fmt.Println("          " + ui.Dim("from "+filepath.Base(chosen)))

	if err := ui.ActionPause("YOUR TURN", []string{
		ui.Bold("QUIT Buzz Desktop completely."),
		"Not just the window — quit the application, so it stops",
		"holding and rewriting this file.",
	}); err != nil {
		return err
	}
	if err := ui.Confirm("Restore that backup now?"); err != nil {
		return err
	}

	if err := registry.CopyFile(chosen, path); err != nil {
		return err
	}
	ui.Ok("restored from " + filepath.Base(chosen))
	ui.Info("Start Buzz Desktop again; the registry is back to its previous state.")
	return nil
}

func summarize(path string) []string {
	rows, err := registry.LoadStore(path)
	if err != nil {
		return []string{"(unreadable)"}
	}
	return registry.NamesIn(rows)
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

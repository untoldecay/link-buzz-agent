package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/untoldecay/link-buzz-agent/internal/registry"
	"github.com/untoldecay/link-buzz-agent/internal/ui"
)

const probeName = "ZZ-TEMPLATE-PROBE"

// runLink walks the human through linking remote agents into Buzz Desktop's
// local registry: back up, capture the row shape the app itself just wrote
// for a throwaway probe agent, substitute each remote identity into it, then
// write them back once the probe is gone and Desktop is closed. One probe,
// one backup and one write cover any number of agents.
func runLink(path string, specs []registry.RowSpec) error {
	const total = 6

	fmt.Println()
	if len(specs) == 1 {
		fmt.Println(ui.Bold("Link an externally-hosted agent into Buzz Desktop"))
	} else {
		fmt.Println(ui.Bold(fmt.Sprintf("Link %d externally-hosted agents into Buzz Desktop", len(specs))))
	}
	ui.Info("registry : " + path)
	for _, spec := range specs {
		ui.Info(fmt.Sprintf("agent    : %-24s %s…", spec.Name, shortKey(spec.Pubkey)))
	}
	ui.Info(ui.Dim("No private key is copied. Desktop will show the agents and allow"))
	ui.Info(ui.Dim("@-mentions, but cannot start them — which is what stops it launching"))
	ui.Info(ui.Dim("a second copy against the same identity."))

	ui.Step(1, total, "Back up the registry")
	data, err := registry.LoadStore(path)
	if err != nil {
		return err
	}

	// Check BEFORE backing up. A no-op re-run must not write a backup, because
	// that backup would contain the already-linked state and would then be
	// the newest — making rollback restore the very thing you wanted undone.
	var todo []registry.RowSpec
	for _, spec := range specs {
		if row := rowByPubkey(data, spec.Pubkey); row != nil {
			ui.Info(fmt.Sprintf("%s is already registered — skipping", spec.Name))
			if b, _ := row["backend"].(map[string]any); b["type"] == "local" {
				ui.Warn("  its row predates the remote marker; if a mention says 'Could not start agent':")
				ui.Warn(fmt.Sprintf("  link-buzz-agent --repair --pubkey %s --name %q", spec.Pubkey, spec.Name))
			}
			continue
		}
		todo = append(todo, spec)
	}
	if len(todo) == 0 {
		return fmt.Errorf(
			"%s already in the registry.\n"+
				"  Nothing to do, and no backup was written — an extra backup here\n"+
				"  would overwrite the one that can actually undo the original link.\n"+
				"  Inspect with --status, or undo with --rollback",
			plural(len(specs), specs[0].Name+" is", "every listed agent is"),
		)
	}
	specs = todo
	label := names(specs)

	stamp := time.Now().Format("20060102T150405")
	backup := registry.BackupPath(path, stamp)
	if err := registry.CopyFile(path, backup); err != nil {
		return err
	}
	ui.Ok(fmt.Sprintf("%d entries backed up", len(data)))
	ui.Info(ui.Dim(backup))
	ui.Info("Undo at any time with: " + ui.Bold("link-buzz-agent --rollback"))

	before := make(map[string]bool)
	for _, r := range data {
		if pk, _ := r["pubkey"].(string); pk != "" {
			before[strings.ToLower(pk)] = true
		}
	}

	ui.Step(2, total, "Let the app write a row for us to copy")
	ui.Info("Rather than guess the file format, we copy the shape of a row that")
	ui.Info("your own build just wrote. That way it matches your version exactly.")
	if err := ui.ActionPause([]string{
		"In Buzz Desktop, go to the " + ui.Bold("Agents") + " view.",
		"",
		"1. Create a new agent. Name it exactly:  " + ui.Bold(probeName),
		"   Any runtime/harness is fine — it will never be used.",
		"",
		"2. If it starts running, " + ui.Bold("stop it") + ".",
		"",
		"3. " + ui.Bold("Leave Buzz Desktop open."),
	}); err != nil {
		return err
	}

	after, err := registry.LoadStore(path)
	if err != nil {
		return err
	}
	var newRows []registry.Row
	for _, r := range after {
		pk, _ := r["pubkey"].(string)
		if pk != "" && !before[strings.ToLower(pk)] {
			newRows = append(newRows, r)
		}
	}
	if len(newRows) == 0 {
		return fmt.Errorf(
			"no new agent appeared in the registry.\n"+
				"  Was %s actually created? Check the Agents view and re-run", probeName,
		)
	}
	if len(newRows) > 1 {
		var named []registry.Row
		for _, r := range newRows {
			if name, _ := r["name"].(string); name == probeName {
				named = append(named, r)
			}
		}
		if len(named) != 1 {
			return fmt.Errorf(
				"%d new agents appeared; cannot tell which is the probe.\n"+
					"  Delete the extras and re-run", len(newRows),
			)
		}
		newRows = named
	}
	template := newRows[0]
	ui.Ok(fmt.Sprintf("captured the shape of '%v' (%d fields)", template["name"], len(template)))

	ui.Step(3, total, "Build "+plural(len(specs), "the "+label+" row", "the rows for "+label))
	var rows []registry.Row
	for _, spec := range specs {
		row, err := registry.BuildRow(template, spec)
		if err != nil {
			return err
		}
		rows = append(rows, row)
		ui.Info(fmt.Sprintf("%-24s %s…  key: %s  autostart: %v  respond_to: %v",
			row["name"], shortKey(fmt.Sprint(row["pubkey"])), ui.Dim("(empty)"),
			row["start_on_app_launch"], row["respond_to"]))
	}
	ui.Ok(plural(len(specs), "Desktop cannot start this agent", "Desktop cannot start these agents") +
		", by construction (no private key copied)")

	ui.Step(4, total, "Remove the probe and close the app")
	if err := ui.ActionPause([]string{
		"1. In Buzz Desktop, " + ui.Bold("delete the agent "+probeName) + ".",
		"   (Agents view -> its card -> delete.)",
		"",
		"2. Then " + ui.Bold("QUIT Buzz Desktop completely") + " — not just the window.",
		"   The app rewrites this file while running, so it must be closed",
		"   before we write, or our change will be lost.",
	}); err != nil {
		return err
	}

	final, err := registry.LoadStore(path)
	if err != nil {
		return err
	}
	var cleaned []registry.Row
	probePubkey := strings.ToLower(fmt.Sprint(template["pubkey"]))
	for _, r := range final {
		name, _ := r["name"].(string)
		pk, _ := r["pubkey"].(string)
		if name == probeName {
			ui.Warn(fmt.Sprintf("%s is still in the registry — deleting it from the file too", probeName))
			continue
		}
		if pk != "" && strings.ToLower(pk) == probePubkey {
			continue
		}
		cleaned = append(cleaned, r)
	}

	ui.Step(5, total, "Write the registry")
	cleaned = append(cleaned, rows...)
	if err := ui.Confirm(fmt.Sprintf("Add %s to the registry now?", label)); err != nil {
		return err
	}
	if err := registry.WriteStore(path, cleaned); err != nil {
		return err
	}
	verify, err := registry.LoadStore(path)
	if err != nil {
		return err
	}
	var missing []registry.RowSpec
	for _, spec := range specs {
		if rowByPubkey(verify, spec.Pubkey) == nil {
			missing = append(missing, spec)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("wrote the file but %s not there — restore with --rollback", names(missing))
	}
	ui.Ok(fmt.Sprintf("%s registered (%d entries, mode 0600)", label, len(verify)))

	ui.Step(6, total, "Check it worked")
	if err := ui.ActionPause([]string{
		"1. " + ui.Bold("Start Buzz Desktop."),
		"",
		"2. " + ui.Bold("Agents view") + " — " + plural(len(specs), label+" should have a card.", "each of "+label+" should have a card."),
		"   No start/stop is expected: this machine holds no key for it.",
		"",
		"3. In a shared channel, type " + ui.Bold("@"+strings.ToLower(specs[0].Name)) + " —",
		"   it should appear in the autocomplete.",
	}); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(ui.Bold("Done."))
	ui.Info("Didn't work?  link-buzz-agent --rollback")
	ui.Info("Backup kept:  " + ui.Dim(backup))
	return nil
}

func rowByPubkey(rows []registry.Row, pubkey string) registry.Row {
	target := strings.ToLower(pubkey)
	for _, r := range rows {
		if pk, _ := r["pubkey"].(string); strings.ToLower(pk) == target {
			return r
		}
	}
	return nil
}

func shortKey(pubkey string) string {
	if len(pubkey) > 16 {
		return pubkey[:16]
	}
	return pubkey
}

func names(specs []registry.RowSpec) string {
	list := make([]string, len(specs))
	for i, s := range specs {
		list[i] = s.Name
	}
	return strings.Join(list, ", ")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/untoldecay/link-buzz-agent/internal/registry"
	"github.com/untoldecay/link-buzz-agent/internal/ui"
)

// runUnlink removes a linked agent's row, for an agent that was retired or
// moved: its card and @-mention entry go away. Rows that hold a private key
// were made by Desktop itself and are left to Desktop's own delete button.
func runUnlink(path, pubkey string) error {
	data, err := registry.LoadStore(path)
	if err != nil {
		return err
	}
	row := rowByPubkey(data, pubkey)
	if row == nil {
		return fmt.Errorf("no agent with pubkey %s… in the registry. Nothing to remove", shortKey(pubkey))
	}
	name := fmt.Sprint(row["name"])
	if nsec, _ := row["private_key_nsec"].(string); nsec != "" {
		return fmt.Errorf(
			"%s holds a private key, so Desktop created it: delete it in the Agents view instead", name,
		)
	}

	fmt.Println()
	fmt.Println(ui.Bold("Unlink") + "  " + name + "  " + shortKey(pubkey) + "…")
	ui.Info("Removes its card and @-mention entry from this Desktop only.")
	ui.Info("The agent itself, its key and its relay profile are not touched.")

	if err := ui.ActionPause([]string{
		ui.Bold("QUIT Buzz Desktop completely."),
		"Not just the window — the app rewrites this file while running,",
		"so it must be closed or the change will be lost.",
	}); err != nil {
		return err
	}
	if err := ui.Confirm(fmt.Sprintf("Remove %s from the registry now?", name)); err != nil {
		return err
	}

	backup := registry.BackupPath(path, time.Now().Format("20060102T150405"))
	if err := registry.CopyFile(path, backup); err != nil {
		return err
	}
	ui.Info(ui.Dim("backed up to " + backup))

	data, err = registry.LoadStore(path)
	if err != nil {
		return err
	}
	var kept []registry.Row
	for _, r := range data {
		if pk, _ := r["pubkey"].(string); strings.EqualFold(pk, pubkey) {
			continue
		}
		kept = append(kept, r)
	}
	if err := registry.WriteStore(path, kept); err != nil {
		return err
	}
	ui.Ok(fmt.Sprintf("%s removed (%d entries left)", name, len(kept)))
	fmt.Println()
	fmt.Println(ui.Bold("Done.") + "  Undo with: link-buzz-agent --rollback")
	return nil
}

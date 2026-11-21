package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/untoldecay/link-buzz-agent/internal/registry"
	"github.com/untoldecay/link-buzz-agent/internal/ui"
)

// runRepair fixes a row that's already in the registry but that Desktop
// still tries to start on mention — the fix link.go's build already applies
// to a fresh row, reapplied here for one that predates it.
func runRepair(path, pubkey, name string) error {
	data, err := registry.LoadStore(path)
	if err != nil {
		return err
	}
	row := rowByPubkey(data, pubkey)
	if row == nil {
		return fmt.Errorf(
			"%s (%s…) is not in the registry.\n"+
				"  Nothing to repair — link it first", name, shortKey(pubkey),
		)
	}

	fmt.Println()
	fmt.Println(ui.Bold("Repair") + "  " + fmt.Sprint(row["name"]) + "  " + shortKey(pubkey) + "…")
	backendJSON, _ := json.Marshal(row["backend"])
	ui.Info("backend now      : " + string(backendJSON))
	ui.Info(fmt.Sprintf("backend_agent_id : %v", row["backend_agent_id"]))
	fmt.Println()
	ui.Info("Mentioning a managed agent makes Desktop start it first. A 'local'")
	ui.Info("backend is never 'running' here, so every mention tried to spawn a")
	ui.Info("process and hit the no-key refusal.")
	ui.Info("Marking it as a remotely-deployed agent skips the start entirely —")
	ui.Info("and is the accurate description of an agent supervised elsewhere.")

	if nsec, _ := row["private_key_nsec"].(string); nsec != "" {
		ui.Warn("this row HAS a private key — leaving it alone, but note that")
		ui.Warn("Desktop could then start a second copy of the agent.")
	}

	stamp := time.Now().Format("20060102T150405")
	backup := registry.BackupPath(path, stamp)

	if err := ui.ActionPause([]string{
		ui.Bold("QUIT Buzz Desktop completely."),
		"Not just the window — the app rewrites this file while running,",
		"so it must be closed or the change will be lost.",
	}); err != nil {
		return err
	}
	if err := ui.Confirm("Apply the repair now?"); err != nil {
		return err
	}

	if err := registry.CopyFile(path, backup); err != nil {
		return err
	}
	ui.Info(ui.Dim("backed up to " + backup))

	data, err = registry.LoadStore(path)
	if err != nil {
		return err
	}
	row = rowByPubkey(data, pubkey)
	if row == nil {
		return fmt.Errorf("the row disappeared while the app was closing — re-run repair")
	}
	registry.ApplyRemoteBackend(row)
	row["start_on_app_launch"] = false
	row["auto_restart_on_config_change"] = false
	if err := registry.WriteStore(path, data); err != nil {
		return err
	}

	ui.Ok("marked as remotely deployed")
	backendJSON, _ = json.Marshal(row["backend"])
	ui.Info("backend          : " + string(backendJSON))
	ui.Info(fmt.Sprintf("backend_agent_id : %v", row["backend_agent_id"]))

	displayName := fmt.Sprint(row["name"])
	if err := ui.ActionPause([]string{
		"1. " + ui.Bold("Start Buzz Desktop."),
		"",
		"2. In a shared channel, type " + ui.Bold("@"+strings.ToLower(displayName)) + " and send.",
		"   It should send with no 'Could not start agent' error.",
		"",
		"3. The Agents card should now read " + ui.Bold("Remote") + " instead of Local.",
	}); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(ui.Bold("Done.") + "  Undo with: link-buzz-agent --rollback")
	return nil
}

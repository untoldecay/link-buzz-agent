// Command link-buzz-agent registers an externally-hosted agent in Buzz
// Desktop's local registry, so it gets an Agents-view card and stays
// @-mentionable, without ever copying its private key to this machine.
//
// Run with no flags for a guided, interactive session. Flags are available
// for scripted use.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/untoldecay/link-buzz-agent/internal/registry"
	"github.com/untoldecay/link-buzz-agent/internal/ui"
)

func main() {
	var (
		storeFlag        string
		pubkeyFlag       string
		nameFlag         string
		agentCommandFlag string
		agentArgsFlag    string
		respondToFlag    string
		rollbackFlag     bool
		backupFlag       string
		statusFlag       bool
		repairFlag       bool
	)

	flag.StringVar(&storeFlag, "store", "", "path to managed-agents.json (auto-detected by default)")
	flag.StringVar(&pubkeyFlag, "pubkey", "", "agent pubkey, 64-char hex")
	flag.StringVar(&nameFlag, "name", "", "display name")
	flag.StringVar(&agentCommandFlag, "agent-command", "", "informational only")
	flag.StringVar(&agentArgsFlag, "agent-args", "", "space-separated, informational only")
	flag.StringVar(&respondToFlag, "respond-to", "owner-only", "owner-only | allowlist | anyone")
	flag.BoolVar(&rollbackFlag, "rollback", false, "restore a backup (newest by default)")
	flag.StringVar(&backupFlag, "backup", "", "with --rollback: restore this backup file name instead of the newest")
	flag.BoolVar(&statusFlag, "status", false, "show what is registered")
	flag.BoolVar(&repairFlag, "repair", false, "fix an already-linked agent Desktop still tries to start on mention")
	flag.Parse()

	path, err := registry.FindStore(storeFlag)
	if err != nil {
		ui.Die(err.Error())
	}

	switch {
	case statusFlag:
		run(runStatus(path))

	case rollbackFlag:
		run(runRollback(path, backupFlag))

	case repairFlag:
		pubkey, name := pubkeyFlag, nameFlag
		if pubkey == "" || name == "" {
			var promptErr error
			pubkey, name, promptErr = promptRepairTarget()
			if promptErr != nil {
				run(promptErr)
			}
		}
		run(runRepair(path, pubkey, name))

	case pubkeyFlag != "" || nameFlag != "":
		spec := registry.RowSpec{
			Pubkey:       strings.ToLower(strings.TrimSpace(pubkeyFlag)),
			Name:         strings.TrimSpace(nameFlag),
			AgentCommand: agentCommandFlag,
			AgentArgs:    strings.Fields(agentArgsFlag),
			RespondTo:    respondToFlag,
		}
		if err := validatePubkeyHex(spec.Pubkey); err != nil {
			ui.Die("--pubkey " + err.Error())
		}
		run(runLink(path, spec))

	default:
		run(runFromMenu(path))
	}
}

// run exits 1 with a styled message on error, 0 otherwise. Kept as one
// choke point so every command path reports failures identically.
func run(err error) {
	if err != nil {
		ui.Die(err.Error())
	}
	os.Exit(0)
}

func runFromMenu(path string) error {
	choice, err := ui.Select("What would you like to do?",
		huh.NewOption("Link a new agent", "link"),
		huh.NewOption("Check what's registered", "status"),
		huh.NewOption("Repair an already-linked agent", "repair"),
		huh.NewOption("Rollback a previous change", "rollback"),
		huh.NewOption("Quit", "quit"),
	)
	if err != nil {
		return err
	}

	switch choice {
	case "quit":
		fmt.Println("Nothing changed.")
		return nil
	case "link":
		spec, err := promptLinkSpec(registry.RowSpec{RespondTo: "owner-only"})
		if err != nil {
			return err
		}
		return runLink(path, spec)
	case "status":
		return runStatus(path)
	case "repair":
		pubkey, name, err := promptRepairTarget()
		if err != nil {
			return err
		}
		return runRepair(path, pubkey, name)
	case "rollback":
		return runRollback(path, "")
	}
	return nil
}

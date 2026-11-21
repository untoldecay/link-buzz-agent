// Command link-buzz-agent registers an externally-hosted agent in Buzz
// Desktop's local registry, so it gets an Agents-view card and stays
// @-mentionable, without ever copying its private key to this machine.
//
// Run with no flags for a guided, interactive session. Flags are available
// for scripted use; --all --from <file> links a whole list of agents in one
// pass.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
		allFlag          bool
		fromFlag         string
		versionFlag      bool
		unlinkFlag       bool
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
	flag.BoolVar(&allFlag, "all", false, "link every agent in the --from list this Desktop doesn't have yet")
	flag.StringVar(&fromFlag, "from", "", "with --all: agent list file (JSON array or \"<pubkey> <name>\" lines)")
	flag.BoolVar(&unlinkFlag, "unlink", false, "with --pubkey: remove a linked agent's row (retired or moved agents)")
	flag.BoolVar(&versionFlag, "version", false, "print the version and exit")
	flag.Parse()

	if versionFlag {
		fmt.Println("link-buzz-agent", version)
		return
	}
	if err := registry.ValidateRespondTo(respondToFlag); err != nil {
		ui.Die("--" + err.Error())
	}
	if fromFlag != "" && !allFlag {
		ui.Die("--from goes with --all")
	}

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

	case unlinkFlag:
		pubkey := strings.ToLower(strings.TrimSpace(pubkeyFlag))
		if err := validatePubkeyHex(pubkey); err != nil {
			ui.Die("--unlink needs --pubkey: " + err.Error())
		}
		run(runUnlink(path, pubkey))

	case allFlag:
		if fromFlag == "" {
			ui.Die("--all needs --from <agent list file>")
		}
		specs, err := loadAgentList(fromFlag, registry.RowSpec{
			AgentCommand: agentCommandFlag,
			AgentArgs:    strings.Fields(agentArgsFlag),
			RespondTo:    respondToFlag,
		})
		if err != nil {
			ui.Die(err.Error())
		}
		run(runLink(path, specs))

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
		run(runLink(path, []registry.RowSpec{spec}))

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
		huh.NewOption("Link every agent in a list file", "all"),
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
		return runLink(path, []registry.RowSpec{spec})
	case "all":
		file, err := ui.Input("Path to the agent list file", "e.g. ~/outgrey-agents.json")
		if err != nil {
			return err
		}
		specs, err := loadAgentList(file, registry.RowSpec{RespondTo: "owner-only"})
		if err != nil {
			return err
		}
		return runLink(path, specs)
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

// version is set at release time with -ldflags "-X main.version=…".
var version = "dev"

func loadAgentList(file string, defaults registry.RowSpec) ([]registry.RowSpec, error) {
	file = strings.TrimSpace(file)
	if strings.HasPrefix(file, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			file = filepath.Join(home, file[2:])
		}
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("agent list: %w", err)
	}
	specs, err := registry.ParseAgentList(data, defaults)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return specs, nil
}

package main

import (
	"fmt"

	"github.com/decaylab/link-buzz-agent/internal/registry"
	"github.com/decaylab/link-buzz-agent/internal/ui"
)

func runStatus(path string) error {
	data, err := registry.LoadStore(path)
	if err != nil {
		return err
	}

	var keyed, defs []registry.Row
	for _, r := range data {
		if pk, _ := r["pubkey"].(string); pk != "" {
			keyed = append(keyed, r)
		} else {
			defs = append(defs, r)
		}
	}

	fmt.Println()
	fmt.Println(ui.Bold("Registry:") + " " + path)
	fmt.Println(ui.Bold("Backups:") + "  " + fmt.Sprint(len(registry.BackupsFor(path))))
	fmt.Println()
	fmt.Println(ui.Bold(fmt.Sprintf("Agents (%d):", len(keyed))))
	for _, r := range keyed {
		name, _ := r["name"].(string)
		if name == "" {
			name = "(unnamed)"
		}
		pubkey, _ := r["pubkey"].(string)
		hasKey := "NO KEY"
		if nsec, _ := r["private_key_nsec"].(string); nsec != "" {
			hasKey = "key"
		}
		boot := "no autostart"
		if v, _ := r["start_on_app_launch"].(bool); v {
			boot = "autostart"
		}
		fmt.Printf("  %-24s %s…  [%s, %s]\n", name, shortKey(pubkey), hasKey, boot)
	}
	fmt.Println()
	fmt.Println(ui.Bold("Definitions:") + " " + fmt.Sprint(len(defs)))
	return nil
}

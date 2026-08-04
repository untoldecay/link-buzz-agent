package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/decaylab/link-buzz-agent/internal/registry"
	"github.com/decaylab/link-buzz-agent/internal/ui"
)

// promptLinkSpec interactively gathers the remote agent's identity and
// behavior. defaults pre-fills anything already supplied via flags.
func promptLinkSpec(defaults registry.RowSpec) (registry.RowSpec, error) {
	spec := defaults
	if spec.RespondTo == "" {
		spec.RespondTo = "owner-only"
	}
	argsLine := strings.Join(spec.AgentArgs, " ")

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Link a remote agent").
				Description("No private key is copied — Desktop will show a card and allow\n@-mentions, but structurally cannot start this agent."),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Please paste the remote agent's public key").
				Description("Not a private key — this is public information, safe to paste. 64 hex characters.").
				Placeholder("e.g. ca1b30626b8a09764c7ff33170c6f1c5c4ad985752231d206d169b2565ba2d2b").
				Value(&spec.Pubkey).
				Validate(validatePubkeyHex),
			huh.NewInput().
				Title("Display name").
				Description("Shown on the Agents card.").
				Placeholder("e.g. Hermes").
				Value(&spec.Name).
				Validate(huh.ValidateNotEmpty()),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Agent command (informational only)").
				Description("Desktop never runs this for a remote agent — it's just descriptive.").
				Placeholder("e.g. hermes").
				Value(&spec.AgentCommand),
			huh.NewInput().
				Title("Agent args (space-separated, informational only)").
				Placeholder("e.g. acp").
				Value(&argsLine),
			huh.NewSelect[string]().
				Title("Who should this agent respond to?").
				Options(
					huh.NewOption("Owner only (recommended)", "owner-only"),
					huh.NewOption("An allowlist of pubkeys", "allowlist"),
					huh.NewOption("Anyone", "anyone"),
				).
				Value(&spec.RespondTo),
		),
	).WithTheme(huh.ThemeCharm())

	if err := form.Run(); err != nil {
		return spec, ui.Aborted{Reason: "aborted — nothing was changed"}
	}

	spec.Pubkey = strings.ToLower(strings.TrimSpace(spec.Pubkey))
	spec.Name = strings.TrimSpace(spec.Name)
	spec.AgentArgs = strings.Fields(argsLine)
	return spec, nil
}

// promptRepairTarget gathers just enough to find the row: its pubkey (the
// only thing repair keys off) and a display name for the follow-up messages.
func promptRepairTarget() (pubkey, name string, err error) {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Please paste the remote agent's public key").
				Description("Whichever agent is already linked but won't start on mention. 64 hex characters.").
				Value(&pubkey).
				Validate(validatePubkeyHex),
			huh.NewInput().
				Title("Display name (for the messages below only)").
				Value(&name).
				Validate(huh.ValidateNotEmpty()),
		),
	).WithTheme(huh.ThemeCharm())

	if err := form.Run(); err != nil {
		return "", "", ui.Aborted{Reason: "aborted — nothing was changed"}
	}
	return strings.ToLower(strings.TrimSpace(pubkey)), strings.TrimSpace(name), nil
}

func validatePubkeyHex(s string) error {
	s = strings.TrimSpace(s)
	if len(s) != 64 {
		return fmt.Errorf("must be exactly 64 hex characters, got %d", len(s))
	}
	for _, c := range s {
		if !isHexChar(c) {
			return fmt.Errorf("must be hex (0-9, a-f) — %q is not valid", string(c))
		}
	}
	return nil
}

func isHexChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

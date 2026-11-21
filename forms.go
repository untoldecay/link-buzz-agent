package main

import (
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/untoldecay/link-buzz-agent/internal/registry"
	"github.com/untoldecay/link-buzz-agent/internal/ui"
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
				Title(ui.BadgeRead+"\n\nLink a remote agent").
				Description(
					"Your agent runs on its own server, somewhere else. Buzz Desktop\n"+
						"doesn't know it exists yet. This adds one entry to a settings file\n"+
						"on THIS computer so Buzz can see it.\n\n"+
						"What you get\n"+
						"  • The agent appears in your Agents list\n"+
						"  • You can @-mention it in channels\n\n"+
						"What gets copied\n"+
						"  • Only its public ID — the equivalent of a phone number\n"+
						"  • No password, no secret key. Nothing sensitive leaves the server\n\n"+
						"What this deliberately cannot do\n"+
						"  • Start, stop or control the agent from here\n"+
						"  • That's on purpose: it stops a second copy of your agent from\n"+
						"    running against the same identity, which would make it reply twice\n\n"+
						"Nothing is written until you confirm, a backup is taken first, and\n"+
						"any change can be undone later with:  link-buzz-agent --rollback",
				),
		),
		huh.NewGroup(
			huh.NewInput().
				Title(ui.BadgeInput+"\n\nPaste the remote agent's public key").
				Description("Not a private key — this is public information, safe to paste. 64 hex characters.").
				Placeholder("e.g. ca1b30626b8a09764c7ff33170c6f1c5c4ad985752231d206d169b2565ba2d2b").
				Value(&spec.Pubkey).
				Validate(validatePubkeyHex),
			huh.NewInput().
				Title(ui.BadgeInput+"\n\nDisplay name").
				Description("Shown on the Agents card.").
				Placeholder("e.g. Hermes").
				Value(&spec.Name).
				Validate(huh.ValidateNotEmpty()),
		),
		huh.NewGroup(
			huh.NewInput().
				Title(ui.BadgeInput+"\n\nAgent command (informational only)").
				Description("Desktop never runs this for a remote agent — it's just descriptive.").
				Placeholder("e.g. hermes").
				Value(&spec.AgentCommand),
			huh.NewInput().
				Title(ui.BadgeInput+"\n\nAgent args (space-separated, informational only)").
				Placeholder("e.g. acp").
				Value(&argsLine),
			huh.NewSelect[string]().
				Title(ui.BadgeInput+"\n\nWho should this agent respond to?").
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
				Title(ui.BadgeInput+"\n\nPaste the remote agent's public key").
				Description("Whichever agent is already linked but won't start on mention. 64 hex characters.").
				Value(&pubkey).
				Validate(validatePubkeyHex),
			huh.NewInput().
				Title(ui.BadgeInput+"\n\nDisplay name (for the messages below only)").
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
	return registry.ValidatePubkey(strings.TrimSpace(s))
}

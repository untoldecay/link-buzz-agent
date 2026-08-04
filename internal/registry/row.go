package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// RemoteBackendID marks a row as backed by a provider-style backend rather
// than a local process.
const RemoteBackendID = "external"

// ApplyRemoteBackend marks the agent as remotely deployed so Desktop never
// tries to start it.
//
// Mentioning a managed agent runs ensureManagedAgentMentionsReady
// (desktop/src/features/messages/ui/useMentionSendFlow.ts:233), which starts
// it first unless it already reads as deployed. build_managed_agent_summary
// (managed_agents/runtime.rs:146) computes that status purely from whether
// backend_agent_id is set — so a provider backend with an id reads as
// "deployed" and the start is skipped. That's also the accurate description:
// the code's own comment defines "deployed" as "infrastructure exists, not
// whether the process is currently running" — exactly an agent supervised by
// systemd (or anything else) on another host. Desktop additionally refuses to
// stop a non-local agent (commands/agents.rs:1288) and labels it "Remote".
func ApplyRemoteBackend(row Row) {
	pubkey, _ := row["pubkey"].(string)
	idSuffix := pubkey
	if len(idSuffix) > 16 {
		idSuffix = idSuffix[:16]
	}
	row["backend"] = map[string]any{
		"type":   "provider",
		"id":     RemoteBackendID,
		"config": map[string]any{},
	}
	row["backend_agent_id"] = fmt.Sprintf("%s:%s", RemoteBackendID, idSuffix)
}

// RowSpec is the identity and behavior of the remote agent being linked.
type RowSpec struct {
	Pubkey       string
	Name         string
	AgentCommand string
	AgentArgs    []string
	RespondTo    string
}

// clearedFields are set to nil unconditionally when present, because they
// describe local process history that a remote agent has none of.
var clearedFields = []string{
	"runtime_pid",
	"last_started_at",
	"last_stopped_at",
	"last_exit_code",
	"last_error",
	"last_error_code",
	"provider_binary_path",
	// backend_agent_id is deliberately NOT in this list — ApplyRemoteBackend
	// sets it, and it's what makes the status read "deployed".
}

// clearedIdentityFields are set to nil when present because the row is no
// longer linked to the throwaway probe's persona/team definition. Boot
// migration backfill_standalone_agents mints a fresh definition for it.
var clearedIdentityFields = []string{
	"persona_id", "slug", "team_id", "persona_team_dir",
	"persona_name_in_team", "source_team",
	"source_team_persona_slug", "persona_source_version",
}

// BuildRow clones the app's own row shape, then substitutes the remote
// identity. Copying a row the app just wrote guarantees the schema matches
// this exact build, instead of being reconstructed from source that may not
// correspond to the installed version.
func BuildRow(template Row, spec RowSpec) (Row, error) {
	raw, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var row Row
	if err := dec.Decode(&row); err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")

	row["pubkey"] = spec.Pubkey
	row["name"] = spec.Name
	if _, ok := row["display_name"]; ok {
		row["display_name"] = spec.Name
	}

	// No key: the card and the mention gate work without it, and its absence
	// is what prevents Desktop from ever starting a second copy of this agent.
	row["private_key_nsec"] = ""
	row["auth_tag"] = nil

	ApplyRemoteBackend(row)

	row["relay_url"] = "" // empty means "use the active workspace relay"

	// Belt and braces on top of the no-key refusal.
	row["start_on_app_launch"] = false
	row["auto_restart_on_config_change"] = false

	for _, field := range clearedIdentityFields {
		if _, ok := row[field]; ok {
			row[field] = nil
		}
	}

	// Descriptive only — Desktop never runs these for a remote agent.
	row["agent_command"] = spec.AgentCommand
	if _, ok := row["agent_args"]; ok {
		args := make([]any, len(spec.AgentArgs))
		for i, a := range spec.AgentArgs {
			args[i] = a
		}
		row["agent_args"] = args
	}
	if _, ok := row["agent_command_override"]; ok {
		row["agent_command_override"] = nil
	}

	row["respond_to"] = spec.RespondTo
	row["respond_to_allowlist"] = []any{}

	for _, field := range clearedFields {
		if _, ok := row[field]; ok {
			row[field] = nil
		}
	}

	if _, ok := row["env_vars"]; ok {
		row["env_vars"] = map[string]any{}
	}
	if _, ok := row["is_active"]; ok {
		row["is_active"] = true
	}
	if _, ok := row["is_builtin"]; ok {
		row["is_builtin"] = false
	}
	row["created_at"] = now
	row["updated_at"] = now

	return row, nil
}

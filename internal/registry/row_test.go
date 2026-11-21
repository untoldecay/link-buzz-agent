package registry

import "testing"

func TestBuildRowIsKeylessAndRemote(t *testing.T) {
	template := Row{
		"pubkey": "probe", "name": "ZZ-TEMPLATE-PROBE", "display_name": "ZZ-TEMPLATE-PROBE",
		"private_key_nsec": "nsec1secret", "auth_tag": "tag", "start_on_app_launch": true,
		"runtime_pid": 42, "persona_id": "p", "agent_args": []any{"x"}, "env_vars": map[string]any{"K": "V"},
	}
	spec := RowSpec{Pubkey: pkA, Name: "Research", AgentCommand: "hermes", AgentArgs: []string{"acp"}, RespondTo: "allowlist"}
	row, err := BuildRow(template, spec)
	if err != nil {
		t.Fatal(err)
	}
	if row["private_key_nsec"] != "" || row["auth_tag"] != nil {
		t.Errorf("key material kept: nsec=%v auth_tag=%v", row["private_key_nsec"], row["auth_tag"])
	}
	if row["start_on_app_launch"] != false || row["runtime_pid"] != nil || row["persona_id"] != nil {
		t.Errorf("local process state kept: %v", row)
	}
	if b, _ := row["backend"].(map[string]any); b["type"] != "provider" || row["backend_agent_id"] != "external:"+pkA[:16] {
		t.Errorf("not marked remote: backend=%v id=%v", row["backend"], row["backend_agent_id"])
	}
	if row["pubkey"] != pkA || row["name"] != "Research" || row["display_name"] != "Research" || row["respond_to"] != "allowlist" {
		t.Errorf("identity not substituted: %v", row)
	}
	if env, _ := row["env_vars"].(map[string]any); len(env) != 0 {
		t.Errorf("env vars kept: %v", env)
	}
	if template["private_key_nsec"] != "nsec1secret" {
		t.Error("BuildRow modified its template")
	}
}

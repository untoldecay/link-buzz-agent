package registry

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// ListEntry is one agent in an --all list file. Only Pubkey and Name are
// required; the other fields override the command-line defaults for that
// agent alone.
type ListEntry struct {
	Pubkey       string   `json:"pubkey"`
	Name         string   `json:"name"`
	RespondTo    string   `json:"respond_to,omitempty"`
	AgentCommand string   `json:"agent_command,omitempty"`
	AgentArgs    []string `json:"agent_args,omitempty"`
}

// ParseAgentList reads an agent list in either of two forms:
//
//   - a JSON array of ListEntry objects;
//   - plain text, one agent per line: "<64-hex pubkey> <display name>",
//     with blank lines and lines starting with # ignored.
//
// defaults supplies AgentCommand, AgentArgs and RespondTo for entries that
// don't set their own. Pubkeys are validated and lowercased; a pubkey listed
// twice is an error, because the second row would shadow the first.
func ParseAgentList(data []byte, defaults RowSpec) ([]RowSpec, error) {
	var entries []ListEntry
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return nil, fmt.Errorf("agent list is not valid JSON: %w", err)
		}
	} else {
		sc := bufio.NewScanner(bytes.NewReader(data))
		for n := 1; sc.Scan(); n++ {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			pubkey, name, _ := strings.Cut(line, " ")
			entries = append(entries, ListEntry{Pubkey: pubkey, Name: strings.TrimSpace(name)})
			if entries[len(entries)-1].Name == "" {
				return nil, fmt.Errorf("line %d: expected \"<pubkey> <name>\"", n)
			}
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("agent list is empty")
	}

	seen := make(map[string]bool)
	specs := make([]RowSpec, 0, len(entries))
	for i, e := range entries {
		spec := defaults
		spec.Pubkey = strings.ToLower(strings.TrimSpace(e.Pubkey))
		spec.Name = strings.TrimSpace(e.Name)
		if err := ValidatePubkey(spec.Pubkey); err != nil {
			return nil, fmt.Errorf("agent %d (%s): pubkey %w", i+1, spec.Name, err)
		}
		if spec.Name == "" {
			return nil, fmt.Errorf("agent %d (%s…): name is empty", i+1, spec.Pubkey[:16])
		}
		if seen[spec.Pubkey] {
			return nil, fmt.Errorf("agent %d (%s): pubkey listed twice", i+1, spec.Name)
		}
		seen[spec.Pubkey] = true
		if e.RespondTo != "" {
			spec.RespondTo = e.RespondTo
		}
		if err := ValidateRespondTo(spec.RespondTo); err != nil {
			return nil, fmt.Errorf("agent %d (%s): %w", i+1, spec.Name, err)
		}
		if e.AgentCommand != "" {
			spec.AgentCommand = e.AgentCommand
		}
		if e.AgentArgs != nil {
			spec.AgentArgs = e.AgentArgs
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

// ValidatePubkey accepts exactly 64 hex characters.
func ValidatePubkey(s string) error {
	if len(s) != 64 {
		return fmt.Errorf("must be exactly 64 hex characters, got %d", len(s))
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return fmt.Errorf("must be hex (0-9, a-f) — %q is not valid", string(c))
		}
	}
	return nil
}

// ValidateRespondTo accepts the three policies Desktop knows.
func ValidateRespondTo(s string) error {
	switch s {
	case "owner-only", "allowlist", "anyone":
		return nil
	}
	return fmt.Errorf("respond_to must be owner-only, allowlist or anyone, got %q", s)
}

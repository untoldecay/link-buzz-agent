package registry

import (
	"strings"
	"testing"
)

const (
	pkA = "1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f"
	pkB = "2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E"
)

var defaults = RowSpec{AgentCommand: "hermes", AgentArgs: []string{"acp"}, RespondTo: "owner-only"}

func TestParseAgentListJSON(t *testing.T) {
	data := `[{"pubkey":"` + pkA + `","name":"Research"},
	          {"pubkey":"` + pkB + `","name":" Code ","respond_to":"allowlist","agent_args":[]}]`
	specs, err := ParseAgentList([]byte(data), defaults)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 2 {
		t.Fatalf("got %d specs", len(specs))
	}
	if specs[0].RespondTo != "owner-only" || specs[0].AgentCommand != "hermes" || len(specs[0].AgentArgs) != 1 {
		t.Errorf("defaults not applied: %+v", specs[0])
	}
	if specs[1].Pubkey != strings.ToLower(pkB) || specs[1].Name != "Code" || specs[1].RespondTo != "allowlist" {
		t.Errorf("entry not normalized or override lost: %+v", specs[1])
	}
	if len(specs[1].AgentArgs) != 0 {
		t.Errorf("explicit empty agent_args should override the default: %+v", specs[1].AgentArgs)
	}
}

func TestParseAgentListText(t *testing.T) {
	data := "# agents\n\n" + pkA + " Research Agent\n" + pkB + "   Code Agent\n"
	specs, err := ParseAgentList([]byte(data), defaults)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 2 || specs[0].Name != "Research Agent" || specs[1].Name != "Code Agent" {
		t.Fatalf("got %+v", specs)
	}
}

func TestParseAgentListRejects(t *testing.T) {
	cases := map[string]string{
		"empty":       "# nothing\n",
		"no name":     pkA + "\n",
		"short key":   "abc Name\n",
		"not hex":     strings.Repeat("z", 64) + " Name\n",
		"duplicate":   pkA + " One\n" + strings.ToUpper(pkA) + " Two\n",
		"bad policy":  `[{"pubkey":"` + pkA + `","name":"X","respond_to":"everyone"}]`,
		"bad json":    `[{"pubkey":`,
		"json noname": `[{"pubkey":"` + pkA + `"}]`,
	}
	for name, data := range cases {
		if _, err := ParseAgentList([]byte(data), defaults); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

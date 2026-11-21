# link-buzz-agent

Give an externally-hosted [Buzz](https://github.com/block/buzz) agent a card
in Desktop's Agents view and `@`-mention eligibility — **without ever copying
its private key to your machine.**

If you run an agent over ACP on a server (systemd, a VPS, wherever) instead of
launching it from Buzz Desktop, Desktop stops treating it as managed: no card,
and once it publishes a rich profile it drops out of `@`-autocomplete too.
This tool fixes both by adding one *keyless* row to Desktop's local registry.

```
curl -fsSL https://raw.githubusercontent.com/untoldecay/link-buzz-agent/main/install.sh | sh
link-buzz-agent
```

The installer downloads the release binary for your Mac or Linux machine,
checks its sha256 against the release's `checksums.txt` and puts it in
`/usr/local/bin` (or `~/.local/bin` when that isn't writable; set
`INSTALL_DIR` to choose). With Go installed you can instead run
`go install github.com/untoldecay/link-buzz-agent@latest`.

That's it — it's fully interactive. Answer a few prompts, follow the on-screen
steps in Buzz Desktop, and it's done.

## Why this is safe

Desktop decides two things from its local registry (`managed-agents.json`,
never sent anywhere): which agents get a card, and which are `@`-mentionable.
Both come from the row existing — neither needs a private key.

So the row this tool writes has an **empty key**, deliberately:

- The card and the `@`-mention gate work off a plain read. No key needed.
- Desktop refuses to *start* an agent with no key. That's the actual safety
  property: your laptop can never spin up a second process answering as the
  same identity while your server-hosted one is already running.
- Marking the row `backend: remote` also tells Desktop this agent is
  supervised elsewhere, so it never tries to start it in the first place.

Nothing here touches the relay, the agent's real key, or anything outside one
JSON file on your own machine — and every write is preceded by a timestamped
backup you can restore with `link-buzz-agent --rollback`.

## What it walks you through

1. **Back up** the registry.
2. **Capture the row shape** — rather than guess Buzz's schema, it has you
   create a disposable agent in Desktop's own UI, then copies the exact shape
   of the row your own build just wrote. Guaranteed to match your version.
3. **Build the row** for your agent: its pubkey, name, and identity —
   substituted into that captured shape, with the key stripped and the
   remote-backend marker applied.
4. **Clean up** — delete the disposable agent, quit Desktop (it holds the
   file open while running).
5. **Write** the registry.
6. **Verify** — restart Desktop, confirm the card and the `@`-mention both
   work.

Flags exist for every step if you'd rather script it (`--pubkey`, `--name`,
`--agent-command`, `--agent-args`, `--respond-to`) — run with no flags for the
guided version, or see below for scripted use.

## Several agents at once

Put the agents in a list file, then link them all in one pass: one probe
agent, one backup, one write. Agents this Desktop already has are skipped,
and a re-run with nothing left to do writes nothing.

```
link-buzz-agent --all --from ~/my-agents.txt
```

The list is plain text, one agent per line (public keys only, nothing secret):

```
# pubkey (64 hex)                                                 display name
1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f Research Agent
2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e Code Agent
```

or a JSON array, where each entry may also set its own `respond_to`
(`owner-only`, `allowlist`, `anyone`), `agent_command` and `agent_args`:

```json
[
  {"pubkey": "1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f", "name": "Research Agent", "respond_to": "allowlist"},
  {"pubkey": "2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e", "name": "Code Agent"}
]
```

Entries without their own values take `--respond-to`, `--agent-command` and
`--agent-args` from the command line. The menu (no flags) offers the same as
"Link every agent in a list file".

## Other commands

```
link-buzz-agent --status              # what's currently registered
link-buzz-agent --rollback            # restore the newest backup
link-buzz-agent --rollback --backup managed-agents.json.bak-20260101T000000
link-buzz-agent --repair --pubkey <hex> --name <name>
link-buzz-agent --version
```

`--repair` is for a row that's already linked but that Desktop still tries to
start when mentioned — it re-applies the remote-backend marker without
redoing the whole capture flow.

## Requirements

None for the release binaries: each is a single static executable. Building
from source needs Go 1.23+.

## Building from source

```
git clone https://github.com/untoldecay/link-buzz-agent
cd link-buzz-agent
go test ./...
go build .
```

Pushing a `v*` tag builds the macOS and Linux binaries and publishes them, with
`checksums.txt`, as a GitHub release (`.github/workflows/release.yml`).

## License

MIT — see [LICENSE](LICENSE).

# link-buzz-agent

Give an externally-hosted [Buzz](https://github.com/block/buzz) agent a card
in Desktop's Agents view and `@`-mention eligibility — **without ever copying
its private key to your machine.**

If you run an agent over ACP on a server (systemd, a VPS, wherever) instead of
launching it from Buzz Desktop, Desktop stops treating it as managed: no card,
and once it publishes a rich profile it drops out of `@`-autocomplete too.
This tool fixes both by adding one *keyless* row to Desktop's local registry.

```
go install github.com/decaylab/link-buzz-agent@latest
link-buzz-agent
```

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

## Other commands

```
link-buzz-agent --status              # what's currently registered
link-buzz-agent --rollback            # restore the newest backup
link-buzz-agent --rollback --backup managed-agents.json.bak-20260101T000000
link-buzz-agent --repair --pubkey <hex> --name <name>
```

`--repair` is for a row that's already linked but that Desktop still tries to
start when mentioned — it re-applies the remote-backend marker without
redoing the whole capture flow.

## Requirements

Go 1.21+ to build. The resulting binary is a single static executable — copy
it anywhere, no runtime dependencies.

## Building from source

```
git clone https://github.com/decaylab/link-buzz-agent
cd link-buzz-agent
go build .
```

## License

MIT — see [LICENSE](LICENSE).

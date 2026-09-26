# Setting up an agent for this repo

Everything an agent needs is declared in [`toolchain.json`](toolchain.json); each tool's config file is an adapter of it. After setup, run:

```bash
python3 tools/harness/cli.py doctor
```

It checks the CLIs on your machine, that every required MCP server and skill pack is declared for each tool, and that each role has its skill and Claude adapter. It never reads secrets.

## 1. CLIs
Required: `git`, `gh` (`gh auth login`), `python3` 3.9+, `go`, `node` 20+, `npm`, `docker`. Optional: `railway` (`railway login`), `wrangler` (`wrangler login`), `infisical` (`infisical login`, then `infisical export --env prod --format dotenv > deploy/.env` inside the deploy worktree), `actionlint`.

## 2. Per tool

**Claude Code** — open the repo and trust it. `.claude/settings.json` registers the plugin marketplaces and enables `superpowers`, `frontend-design`, `code-review` and `figma` (accept the install prompt); `.mcp.json` declares `context7`, `playwright`, `railway` (approve them once). Role agents are in `.claude/agents/`, slash commands in `.claude/commands/`, skills through the `.claude/skills` symlink. The SessionStart hook prints the harness briefing.

**Codex** — trust the project so `.codex/config.toml` loads (MCP servers). Codex reads `AGENTS.md` and `.agents/skills` directly; install the superpowers pack per its README (https://github.com/obra/superpowers) for the process skills. Hook: `.codex/hooks.json`.

**Gemini CLI** — `.gemini/settings.json` loads `AGENTS.md` as context, runs the SessionStart hook and declares the MCP servers. Install the superpowers pack per its README.

**Any other agent** — read `AGENTS.md`, run `python3 tools/harness/cli.py context` before the first task, adopt a role from `.agents/roles/`, follow its skill in `.agents/skills/`, and add the MCP servers from `toolchain.json` in whatever format the tool uses.

## 3. Accounts the owner holds (never agents)
Supabase, Upstash, Railway, Cloudflare, Google Cloud OAuth, OpenRouter, Infisical. Values live in Infisical project `english-learning`; see `deploy/README.md`.

## Changing the toolchain
Edit `toolchain.json` first, then every adapter it lists, then run `doctor`. A new MCP server, skill pack or role that is missing from an adapter is a doctor failure.

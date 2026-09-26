# tools/harness/cli.py
"""Single entry point for every harness state mutation. Agents call this; they never hand-edit frontmatter."""
import argparse, datetime, json, os, pathlib, re, shutil, subprocess, sys, time, unicodedata

if __package__ in (None, ""):
    sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2]))
from tools.harness.frontmatter import split_document, parse, join_document
from tools.harness.schema import kind_of, validate, TRANSITIONS
from tools.harness.scan import scan, load
from tools.harness.state import render_state, render_context, plan_sort_key, PRIO_RANK

TEMPLATES = pathlib.Path(".agents/templates")
LOCK = pathlib.Path("harness/.lock")          # pre-2026-09-23 single lock, migrated on first use
LOCK_DIR = pathlib.Path("harness/.locks")
LOCK_STALE_SECONDS = 2 * 3600


def today():
    return datetime.date.today().isoformat()


def slugify(text):
    text = unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode()
    return re.sub(r"[^a-z0-9]+", "-", text.lower()).strip("-")[:60]


def read(path):
    fm, body = split_document(pathlib.Path(path).read_text())
    return parse(fm), body


def write(path, fm, body):
    kind = kind_of(path)
    errs = validate(kind, fm) if kind else []
    if errs:
        print("\n".join(f"error: {e}" for e in errs)); return 1
    pathlib.Path(path).write_text(join_document(fm, body))
    return 0


def template(name, **kw):
    return (TEMPLATES / f"{name}.md").read_text().format(**kw)


def refresh_state():
    text = render_state(scan("."))
    pathlib.Path("harness/STATE.md").write_text(text)
    return text


# ---- commands -------------------------------------------------------------

def cmd_validate(a):
    res = scan(".")
    for art in res.invalid:
        print(f"{art.rel}: " + "; ".join(art.errors))
    return 1 if res.invalid else 0


def cmd_state(a):
    print(refresh_state()); return 0


def _git(*args):
    try:
        return subprocess.run(["git", *args], capture_output=True, text=True, timeout=10).stdout.rstrip()
    except (OSError, subprocess.SubprocessError):
        return ""


def cmd_context(a):
    """Read-only session briefing. --hook wraps it in the SessionStart JSON that Claude Code, Gemini CLI and Codex all accept."""
    status = "\n".join(_git("status", "--short").splitlines()[:15])
    git = f"Branch: {_git('branch', '--show-current')}\n" + (f"{status}\n" if status else "") + \
          f"\nRecent commits:\n{_git('log', '--oneline', '-8')}"
    codemap = pathlib.Path("harness/CODEMAP.md")
    text = render_context(scan("."), codemap.read_text() if codemap.exists() else "", git)
    if a.hook:
        text = json.dumps({"hookSpecificOutput": {"hookEventName": "SessionStart", "additionalContext": text}})
    print(text); return 0


def cmd_slug(a):
    print(slugify(a.title)); return 0


def cmd_new_run(a):
    base = pathlib.Path("harness/ideas")
    prefix = f"{today()}-run-"
    n = 1 + max([int(p.name[len(prefix):]) for p in base.glob(prefix + "*") if p.name[len(prefix):].isdigit()] or [0])
    run = base / f"{prefix}{n:02d}"
    run.mkdir(parents=True)
    (run / "_run.md").write_text(template("run", run=run.name, mode="mvp" if a.mvp else "features"))
    print(run.as_posix()); return 0


def cmd_new_idea(a):
    run = pathlib.Path(a.run)
    if not run.is_dir():
        print(f"error: run dir not found: {run}"); return 1
    slug = slugify(a.title)
    path = run / f"{slug}.md"
    if path.exists():
        print(f"error: exists: {path}"); return 1
    fm = {"type": a.type, "status": "proposed", "source": a.source, "run": run.name}
    if a.order is not None:
        fm["order"] = a.order
    if a.priority:
        fm["priority"] = a.priority
    code = write(path, fm, template("idea", title=a.title))
    if code == 0:
        print(path.as_posix()); refresh_state()
    return code


def cmd_new_plan(a):
    ifm, ibody = read(a.idea)
    if ifm.get("status") != "selected":
        print(f"error: idea must be 'selected' (is '{ifm.get('status')}')"); return 1
    title = next((l[2:].strip() for l in ibody.splitlines() if l.startswith("# ")), "untitled")
    slug = pathlib.Path(a.idea).stem
    path = pathlib.Path("harness/plans") / f"{today()}-{slug}.md"
    if path.exists():
        print(f"error: exists: {path}"); return 1
    fm = {"idea": a.idea, "status": "draft", "priority": ifm.get("priority") or "medium", "merged": False}
    if isinstance(ifm.get("order"), int):
        fm["order"] = ifm["order"]
    code = write(path, fm, template("plan", title=title, idea=a.idea))
    if code:
        return code
    ifm["status"] = "planned"; ifm["plan"] = path.as_posix()
    write(a.idea, ifm, ibody)
    print(path.as_posix()); refresh_state(); return 0


def cmd_new_review(a):
    pfm, pbody = read(a.plan)
    title = next((l[2:].strip() for l in pbody.splitlines() if l.startswith("# ")), "untitled").replace(" — Plan", "")
    slug = re.sub(r"^\d{4}-\d{2}-\d{2}-", "", pathlib.Path(a.plan).stem)
    path = pathlib.Path("harness/reviews") / f"{today()}-{slug}.md"
    # A re-review of the same plan on the same day must not overwrite the first
    # one: the earlier review is the record of what was found and fixed.
    n = 2
    while path.exists():
        path = pathlib.Path("harness/reviews") / f"{today()}-{slug}-{n}.md"
        n += 1
    fm = {"plan": a.plan, "verdict": a.verdict, "bugs": a.bugs or []}
    body = template("review", title=title, plan=a.plan, branch=pfm.get("branch") or "-", worktree=pfm.get("worktree") or "-")
    code = write(path, fm, body)
    if code == 0:
        print(path.as_posix()); refresh_state()
    return code


def _coerce(v):
    if v in ("true", "false"): return v == "true"
    if re.fullmatch(r"-?\d+", v): return int(v)
    if v.startswith("[") and v.endswith("]"): return [x.strip() for x in v[1:-1].split(",") if x.strip()]
    return v


def cmd_blockers(a):
    res = scan(".")
    plans = [p for p in res.plans if not a.plan or p.rel == a.plan]
    found = False
    for plan in plans:
        for b in res.blockers_for(plan.rel):
            found = True
            print(f"{b.rel}\tblocks\t{plan.rel}")
    return 1 if found else 0


def cmd_set(a):
    kind = kind_of(a.file)
    if not kind:
        print("error: not a harness artifact"); return 1
    fm, body = read(a.file)
    if kind == "plan" and any(kv == "merged=true" for kv in a.pairs):
        blockers = scan(".").blockers_for(a.file)
        if blockers:
            print("error: cannot mark merged — unresolved blockers:")
            for b in blockers:
                print(f"  {b.rel}")
            return 1
    for kv in a.pairs:
        k, _, v = kv.partition("=")
        v = _coerce(v)
        if k == "status" and kind in TRANSITIONS:
            cur = fm.get("status")
            if v != cur and v not in TRANSITIONS[kind].get(cur, []):
                print(f"error: illegal transition {cur} -> {v} for {kind}"); return 1
        fm[k] = v
    code = write(a.file, fm, body)
    if code == 0:
        refresh_state()
    return code


def cmd_next(a):
    res = scan(".")
    if a.stage == "evaluate":
        # The evaluator is the single point that chooses between reviewer-filed
        # bugs (_inbox/) and ideator features (run folders): both queues, one list.
        # Blockers first, then by priority, then inbox before runs, then path.
        def key(i):
            return (0 if i.fm.get("blocks") else 1, PRIO_RANK.get(i.fm.get("priority"), 3), 0 if "/_inbox/" in i.rel else 1, i.rel)
        items = [i.rel for i in sorted(res.ideas, key=key) if i.fm["status"] == "proposed"]
    elif a.stage == "execute":
        items = [p.rel for p in sorted(res.plans, key=plan_sort_key) if p.fm["status"] == "approved"]
    elif a.stage == "review":
        items = [p.rel for p in sorted(res.plans, key=lambda p: p.rel) if p.fm["status"] == "done" and not res.reviews_for(p.rel)]
    else:
        items = []
    if a.all:
        print("\n".join(items))
    else:
        print(items[0] if items else "")
    return 0


def _lock_path(plan):
    """One lock per plan. Executors work in separate worktrees, so only same-plan runs collide."""
    return LOCK_DIR / (re.sub(r"[^A-Za-z0-9._-]", "_", plan) + ".lock")


def _migrate_legacy_lock():
    if not LOCK.exists():
        return
    plan, mtime = LOCK.read_text().strip(), LOCK.stat().st_mtime
    if plan:
        LOCK_DIR.mkdir(parents=True, exist_ok=True)
        dest = _lock_path(plan)
        if not dest.exists():
            dest.write_text(plan + "\n")
            os.utime(dest, (mtime, mtime))
    LOCK.unlink()


def _held_locks():
    if not LOCK_DIR.exists():
        return []
    return sorted(f for f in LOCK_DIR.iterdir() if f.suffix == ".lock")


def cmd_lock(a):
    _migrate_legacy_lock()
    path = _lock_path(a.plan)
    if path.exists() and time.time() - path.stat().st_mtime < LOCK_STALE_SECONDS:
        print(f"error: locked by {path.read_text().strip()}"); return 1
    LOCK_DIR.mkdir(parents=True, exist_ok=True)
    path.write_text(a.plan + "\n"); print("locked"); return 0


def cmd_unlock(a):
    _migrate_legacy_lock()
    if a.plan:
        path = _lock_path(a.plan)
        if not path.exists():
            print(f"error: not locked: {a.plan}"); return 1
        path.unlink(); print("unlocked"); return 0
    held = _held_locks()
    if len(held) > 1:
        print("error: several plans are locked — name the one to release:")
        for f in held:
            print(f"  {f.read_text().strip()}")
        return 1
    for f in held:
        f.unlink()
    print("unlocked"); return 0


def cmd_stale_worktrees(a):
    res = scan(".")
    merged = {pathlib.Path(p.fm.get("worktree", "")).name for p in res.plans if p.fm.get("merged")}
    for wt in sorted(pathlib.Path(".worktrees").glob("*")) if pathlib.Path(".worktrees").exists() else []:
        if wt.name in merged:
            print(wt.as_posix())
    return 0


# ---- doctor ----------------------------------------------------------------
# Checks this machine and every tool adapter against .agents/toolchain.json.
# Reads config files only; never reads secrets or runs the tools.

def _json(path):
    try:
        return json.loads(pathlib.Path(path).read_text())
    except (OSError, ValueError):
        return {}


def _codex_servers(path=".codex/config.toml"):
    try:
        text = pathlib.Path(path).read_text()
    except OSError:
        return set()
    return set(re.findall(r"^\[mcp_servers\.([A-Za-z0-9_-]+)\]", text, re.M))


def cmd_doctor(a):
    tc = _json(".agents/toolchain.json")
    if not tc:
        print("FAIL toolchain: .agents/toolchain.json missing or not JSON"); return 1
    fails = 0

    def report(level, what, msg):
        nonlocal fails
        fails += level == "FAIL"
        print(f"{level:<4} {what}{': ' + msg if msg else ''}")

    for need, group in (("required", "FAIL"), ("optional", "WARN")):
        for name, why in sorted(tc.get("clis", {}).get(need, {}).items()):
            report("ok" if shutil.which(name) else group, f"cli {name}", "" if shutil.which(name) else f"not on PATH ({why})")

    claude = _json(".claude/settings.json")
    enabled = {k for k, v in claude.get("enabledPlugins", {}).items() if v}
    declared = {
        "claude-code": set(_json(".mcp.json").get("mcpServers", {})),
        "codex": _codex_servers(),
        "gemini-cli": set(_json(".gemini/settings.json").get("mcpServers", {})),
    }
    plugin_mcp = {pk["name"] for pk in tc.get("skill_packs", []) if pk.get("claude_plugin") in enabled}
    for name, srv in sorted(tc.get("mcp_servers", {}).items()):
        missing = [t for t, have in declared.items() if name not in have and not (t == "claude-code" and name in plugin_mcp)]
        level = "ok" if not missing else ("FAIL" if srv.get("required") else "WARN")
        report(level, f"mcp {name}", f"not declared for {', '.join(missing)}" if missing else "")

    for pk in tc.get("skill_packs", []):
        ok = pk.get("claude_plugin") in enabled
        report("ok" if ok else ("FAIL" if pk.get("required") else "WARN"), f"skill-pack {pk['name']}",
               "" if ok else f"{pk.get('claude_plugin')} not enabled in .claude/settings.json")

    base = pathlib.Path(tc.get("project_skills", {}).get("path", ".agents/skills"))
    for name in tc.get("project_skills", {}).get("names", []):
        ok = (base / name / "SKILL.md").is_file()
        report("ok" if ok else "FAIL", f"skill {name}", "" if ok else f"{base / name / 'SKILL.md'} missing")

    for role, spec in sorted(tc.get("roles", {}).items()):
        gaps = [p for p in (spec["role"], str(base / spec["skill"] / "SKILL.md"), f".claude/agents/harness-{role}.md")
                if not pathlib.Path(p).is_file()]
        report("ok" if not gaps else "FAIL", f"role {role}", f"missing {', '.join(gaps)}" if gaps else "")

    hook = tc.get("hooks", {}).get("session_start", "")
    for tool, path in (("claude-code", ".claude/settings.json"), ("codex", ".codex/hooks.json"), ("gemini-cli", ".gemini/settings.json")):
        try:
            ok = bool(hook) and hook in pathlib.Path(path).read_text()
        except OSError:
            ok = False
        report("ok" if ok else "FAIL", f"hook {tool}", "" if ok else f"{path} does not run `{hook}`")

    print(f"\n{'FAIL' if fails else 'ok'}: {fails} problem(s)")
    return 1 if fails else 0


def main(argv=None):
    ap = argparse.ArgumentParser(prog="harness")
    sub = ap.add_subparsers(dest="cmd", required=True)
    sub.add_parser("validate").set_defaults(fn=cmd_validate)
    sub.add_parser("doctor").set_defaults(fn=cmd_doctor)
    sub.add_parser("state").set_defaults(fn=cmd_state)
    p = sub.add_parser("context"); p.add_argument("--hook", action="store_true"); p.set_defaults(fn=cmd_context)
    p = sub.add_parser("slug"); p.add_argument("title"); p.set_defaults(fn=cmd_slug)
    p = sub.add_parser("new-run"); p.add_argument("--mvp", action="store_true"); p.set_defaults(fn=cmd_new_run)
    p = sub.add_parser("new-idea")
    p.add_argument("--run", required=True); p.add_argument("--title", required=True)
    p.add_argument("--type", required=True, choices=["feature", "bug", "mvp-slice"])
    p.add_argument("--source", required=True, choices=["ideator", "human", "reviewer"])
    p.add_argument("--order", type=int); p.add_argument("--priority", choices=["high", "medium", "low"])
    p.set_defaults(fn=cmd_new_idea)
    p = sub.add_parser("new-plan"); p.add_argument("--idea", required=True); p.set_defaults(fn=cmd_new_plan)
    p = sub.add_parser("new-review"); p.add_argument("--plan", required=True)
    p.add_argument("--verdict", required=True, choices=["pass", "pass-with-bugs", "fail"]); p.add_argument("--bugs", nargs="*")
    p.set_defaults(fn=cmd_new_review)
    p = sub.add_parser("set"); p.add_argument("file"); p.add_argument("pairs", nargs="+"); p.set_defaults(fn=cmd_set)
    p = sub.add_parser("next"); p.add_argument("--stage", required=True, choices=["evaluate", "execute", "review"])
    p.add_argument("--all", action="store_true"); p.set_defaults(fn=cmd_next)
    p = sub.add_parser("blockers"); p.add_argument("--plan"); p.set_defaults(fn=cmd_blockers)
    p = sub.add_parser("lock"); p.add_argument("plan"); p.set_defaults(fn=cmd_lock)
    p = sub.add_parser("unlock"); p.add_argument("plan", nargs="?"); p.set_defaults(fn=cmd_unlock)
    sub.add_parser("stale-worktrees").set_defaults(fn=cmd_stale_worktrees)
    a = ap.parse_args(argv)
    return a.fn(a)


if __name__ == "__main__":
    sys.exit(main())

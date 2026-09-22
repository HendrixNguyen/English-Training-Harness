# tools/harness/state.py
"""Render harness/STATE.md from a ScanResult. Never hand-edited."""
import datetime

PRIO_RANK = {"high": 0, "medium": 1, "low": 2, None: 3}


def _line(a, extra=""):
    p = a.fm.get("priority")
    tag = f" [{p}]" if p else ""
    return f"- `{a.rel}` — {a.title}{tag}{extra}"


def plan_sort_key(a):
    return (PRIO_RANK.get(a.fm.get("priority"), 3), a.fm.get("order") or 999, a.rel)


def render_state(res):
    now = datetime.datetime.now().strftime("%Y-%m-%d %H:%M")
    out = [f"# Harness state\n", f"_Generated {now}. Do not edit — run `python3 tools/harness/cli.py state`._\n"]

    def section(title, items, empty="_none_"):
        out.append(f"\n## {title}\n")
        out.extend(items if items else [empty])

    section("Invalid", [f"- `{a.rel}` — " + "; ".join(a.errors) for a in res.invalid])
    blocked = []
    for plan in res.plans:
        for b in res.blockers_for(plan.rel):
            blocked.append(f"- `{b.rel}` — {b.title} — blocks `{plan.rel}`")
    section("Blockers (merge refused until fixed)", blocked)
    section("Inbox (reviewer bugs awaiting evaluation)", [_line(a) for a in res.ideas if "/_inbox/" in a.rel and a.fm["status"] == "proposed"])
    section("Proposed", [_line(a) for a in res.ideas if a.fm["status"] == "proposed" and "/_inbox/" not in a.rel])
    section("Selected", [_line(a) for a in res.ideas if a.fm["status"] == "selected"])
    section("Planned (awaiting approval)", [_line(a) for a in sorted(res.plans, key=plan_sort_key) if a.fm["status"] == "draft"])
    section("Approved", [_line(a) for a in sorted(res.plans, key=plan_sort_key) if a.fm["status"] == "approved"])
    section("Executing", [_line(a) for a in res.plans if a.fm["status"] == "executing"])
    done = [a for a in res.plans if a.fm["status"] == "done"]
    done_lines = []
    for a in sorted(done, key=lambda x: x.rel, reverse=True)[:10]:
        revs = res.reviews_for(a.rel)
        note = " (unreviewed)" if not revs else f" (review: {revs[-1].fm['verdict']})"
        note += " (merged)" if a.fm.get("merged") else ""
        done_lines.append(_line(a, note))
    section("Done (last 10)", done_lines)
    section("Failed", [_line(a) for a in res.plans if a.fm["status"] == "failed"])
    return "\n".join(out) + "\n"

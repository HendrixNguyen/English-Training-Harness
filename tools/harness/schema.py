# tools/harness/schema.py
"""Required keys, allowed values, and legal status transitions per artifact kind."""

ENUMS = {
    "idea": {
        "type": ["feature", "bug", "mvp-slice"],
        "status": ["proposed", "selected", "rejected", "planned"],
        "source": ["ideator", "human", "reviewer"],
        "priority": ["high", "medium", "low"],
    },
    "plan": {
        "status": ["draft", "approved", "executing", "done", "failed"],
        "priority": ["high", "medium", "low"],
    },
    "review": {
        "verdict": ["pass", "pass-with-bugs", "fail"],
    },
}

REQUIRED = {
    "idea": ["type", "status", "source", "run"],
    "plan": ["idea", "status", "priority", "merged"],
    "review": ["plan", "verdict", "bugs"],
}

TRANSITIONS = {
    "idea": {
        "proposed": ["selected", "rejected"],
        "selected": ["planned", "rejected"],
        "rejected": [],
        "planned": [],
    },
    "plan": {
        "draft": ["approved"],
        "approved": ["executing", "draft"],
        "executing": ["done", "failed"],
        "done": [],
        "failed": ["approved"],
    },
}


def kind_of(path):
    p = str(path).replace("\\", "/")
    name = p.rsplit("/", 1)[-1]
    if not name.endswith(".md") or name.startswith("_") or name.startswith("."):
        return None
    if "/harness/ideas/" in "/" + p or p.startswith("harness/ideas/"):
        return "idea"
    if "/harness/plans/" in "/" + p or p.startswith("harness/plans/"):
        return "plan"
    if "/harness/reviews/" in "/" + p or p.startswith("harness/reviews/"):
        return "review"
    return None


def validate(kind, fm):
    errs = []
    for k in REQUIRED[kind]:
        if k not in fm or fm[k] is None:
            errs.append(f"missing key: {k}")
    for k, allowed in ENUMS.get(kind, {}).items():
        if k in fm and fm[k] is not None and fm[k] not in allowed:
            errs.append(f"{k}: '{fm[k]}' not in {allowed}")
    if kind == "idea":
        if fm.get("blocks"):
            # A blocker is a review finding that must be fixed before its plan's
            # branch may be merged. It takes the fast path: straight to the
            # evaluator, never waiting for the next ideation run.
            if fm.get("type") != "bug":
                errs.append("blocks is only valid on a bug")
            if fm.get("priority") != "high":
                errs.append("a blocker must be priority high")
        if fm.get("status") == "rejected" and not fm.get("rejected_reason"):
            errs.append("rejected idea needs rejected_reason")
        if fm.get("type") == "mvp-slice" and not isinstance(fm.get("order"), int):
            errs.append("mvp-slice idea needs integer order")
        if fm.get("status") == "planned" and not fm.get("plan"):
            errs.append("planned idea needs plan path")
    if kind == "plan" and not isinstance(fm.get("merged"), bool):
        errs.append("merged must be true/false")
    if kind == "review" and not isinstance(fm.get("bugs"), list):
        errs.append("bugs must be a list")
    return errs

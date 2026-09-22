# tools/harness/scan.py
"""Walk harness/, parse every artifact, validate, return a ScanResult."""
import pathlib
from dataclasses import dataclass, field
from .frontmatter import split_document, parse
from .schema import kind_of, validate


@dataclass
class Artifact:
    rel: str
    kind: str
    fm: dict
    body: str
    title: str
    errors: list = field(default_factory=list)


@dataclass
class ScanResult:
    root: str
    ideas: list = field(default_factory=list)
    plans: list = field(default_factory=list)
    reviews: list = field(default_factory=list)
    invalid: list = field(default_factory=list)

    def reviews_for(self, plan_rel):
        return [r for r in self.reviews if r.fm.get("plan") == plan_rel]


def _title(body):
    for line in body.splitlines():
        if line.startswith("# "):
            return line[2:].strip()
    return "(untitled)"


def load(root, rel):
    text = pathlib.Path(root, rel).read_text()
    fm_text, body = split_document(text)
    fm = parse(fm_text)
    kind = kind_of(rel)
    errs = validate(kind, fm) if kind else ["unknown artifact kind"]
    if not fm_text:
        errs.insert(0, "no frontmatter")
    return Artifact(rel=rel, kind=kind or "?", fm=fm, body=body, title=_title(body), errors=errs)


def scan(root="."):
    res = ScanResult(root=root)
    base = pathlib.Path(root, "harness")
    for p in sorted(base.rglob("*.md")):
        rel = p.relative_to(root).as_posix()
        if kind_of(rel) is None:
            continue
        art = load(root, rel)
        if art.errors:
            res.invalid.append(art)
        else:
            getattr(res, art.kind + "s").append(art)
    return res

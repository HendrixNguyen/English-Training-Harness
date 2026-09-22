# tools/harness/frontmatter.py
"""Minimal YAML-subset frontmatter: scalars, quoted strings, inline lists, bools, ints, null."""
import re

_FENCE = "---"


def split_document(text):
    """Return (frontmatter_text, body). frontmatter_text is '' when absent."""
    if not text.startswith(_FENCE + "\n"):
        return "", text
    end = text.find("\n" + _FENCE + "\n", len(_FENCE))
    if end == -1:
        return "", text
    fm = text[len(_FENCE) + 1:end + 1]
    body = text[end + len(_FENCE) + 2:]
    return fm, body


def _scalar(raw):
    raw = raw.strip()
    if raw == "":
        return None
    if raw.startswith('"') and raw.endswith('"') and len(raw) >= 2:
        return raw[1:-1].replace('\\"', '"')
    if raw.startswith("[") and raw.endswith("]"):
        inner = raw[1:-1].strip()
        return [] if not inner else [_scalar(x) for x in inner.split(",")]
    if raw == "true":
        return True
    if raw == "false":
        return False
    if re.fullmatch(r"-?\d+", raw):
        return int(raw)
    return raw


def parse(fm_text):
    out = {}
    for line in fm_text.splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        key, sep, val = line.partition(":")
        if not sep:
            continue
        out[key.strip()] = _scalar(val)
    return out


def _emit(v):
    if v is None:
        return ""
    if isinstance(v, bool):
        return "true" if v else "false"
    if isinstance(v, int):
        return str(v)
    if isinstance(v, list):
        return "[" + ", ".join(_emit(x) for x in v) + "]"
    s = str(v)
    if s == "" or any(c in s for c in ':#[]",') or s in ("true", "false") or re.fullmatch(r"-?\d+", s):
        return '"' + s.replace('"', '\\"') + '"'
    return s


def serialize(d):
    return "".join(f"{k}: {_emit(v)}".rstrip() + "\n" for k, v in d.items())


def join_document(d, body):
    return f"{_FENCE}\n{serialize(d)}{_FENCE}\n{body}"

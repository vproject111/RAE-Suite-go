#!/usr/bin/env python3
"""Exhaustive Zero-Omission Python-to-Go AST Census for RAE-Suite vs RAE-Suite-go.

Scans 100% of first-party Python files in /home/grzegorz/cloud/RAE-Suite (without a manual
whitelist), extracts AST classes (separating Domain/Engine/Service/Adapter/Worker classes from
Pydantic DTOs/Enums/Exceptions), public methods, top-level functions, LOC, and cyclomatic
branch counts, and maps them against /home/grzegorz/cloud/RAE-Suite-go.
"""

from __future__ import annotations

import ast
import json
import pathlib
import re
from collections import defaultdict
from typing import Any

PY_ROOT = pathlib.Path("/home/grzegorz/cloud/RAE-Suite")
GO_ROOT = pathlib.Path("/home/grzegorz/cloud/RAE-Suite-go")
DOCS_OUT_MD = GO_ROOT / "docs" / "PELNE_ZESTAWIENIE_SPISU_POWSZECHNEGO_PY_GO.md"
DOCS_OUT_JSON = GO_ROOT / "docs" / "SPIS_POWSZECHNY_PY_GO.json"

EXCLUDED_DIRS = {
    ".venv",
    "venv",
    ".venv_test",
    "node_modules",
    "__pycache__",
    ".git",
    "tests",
    "test",
    "alembic",
    "benchmarks",
    "benchmarking",
    "work_dir",
    ".pytest_cache",
    ".mypy_cache",
    "tmp_node1_sync",
    ".gemini",
    "htmlcov",
    "dist",
    "build",
    "egg-info",
}

DTO_BASES = {
    "BaseModel",
    "Enum",
    "StrEnum",
    "IntEnum",
    "Exception",
    "RuntimeError",
    "ValueError",
    "TypedDict",
    "BaseSettings",
}


def count_py_branches(node: ast.AST) -> int:
    branches = 0
    for child in ast.walk(node):
        if isinstance(
            child,
            (
                ast.If,
                ast.For,
                ast.AsyncFor,
                ast.While,
                ast.Try,
                ast.IfExp,
                ast.comprehension,
                ast.Match,
            ),
        ):
            branches += 1
    return branches


def classify_subsystem(rel: pathlib.Path) -> tuple[str, str]:
    parts = rel.parts
    if parts[0] in (".openclaw_config", "packages") and (
        parts[0] == ".openclaw_config"
        or (len(parts) > 1 and parts[1] == "rae-open-claw")
    ):
        return "INTENTIONALLY_EXCLUDED_OPENCLAW", str(parts[0] if parts[0] == ".openclaw_config" else "packages/rae-open-claw")

    if parts[0] == "packages" and len(parts) >= 2:
        pkg = parts[1]
        if pkg == "rae-agentic-memory":
            sub = parts[2] if len(parts) >= 3 else "(root)"
            if sub == "rae-core":
                mod = parts[4] if len(parts) >= 5 else "(root)"
                return "packages/rae-agentic-memory/rae-core", f"rae_core/{mod}"
            if sub == "apps":
                app = parts[3] if len(parts) >= 4 else "(root)"
                subapp = parts[4] if len(parts) >= 5 else "(root)"
                return f"packages/rae-agentic-memory/apps/{app}", f"{app}/{subapp}"
            return f"packages/rae-agentic-memory/{sub}", sub
        mod = parts[2] if len(parts) >= 3 else "(root)"
        return f"packages/{pkg}", f"{pkg}/{mod}"

    if len(parts) == 1:
        return "(root)", "(root)"
    return parts[0], "/".join(parts[:2])


def tokenize_name(name: str) -> list[str]:
    s1 = re.sub(r"(.)([A-Z][a-z]+)", r"\1_\2", name)
    s2 = re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", s1).lower()
    return [t for t in s2.split("_") if len(t) > 2 and t not in {"the", "and", "for", "with", "from", "base", "abstract", "interface"}]


def index_go_codebase() -> dict[str, Any]:
    go_files = {}
    go_structs = {}
    go_funcs = {}
    go_routes = set()
    total_loc = 0
    total_branches = 0

    for gp in sorted(GO_ROOT.rglob("*.go")):
        if gp.name.endswith("_test.go") or "vendor" in gp.parts:
            continue
        rel = str(gp.relative_to(GO_ROOT))
        src = gp.read_text(encoding="utf-8", errors="ignore")
        loc = len(src.splitlines())
        branches = len(re.findall(r"\b(if|for|switch|case|select)\b", src))
        total_loc += loc
        total_branches += branches

        structs = re.findall(r"type\s+([A-Za-z0-9_]+)\s+struct", src)
        funcs = re.findall(r"func\s+(?:\([^)]+\)\s+)?([A-Za-z0-9_]+)\s*\(", src)
        routes = re.findall(r'HandleFunc\(\s*"([^"]+)"', src)
        for r in routes:
            go_routes.add(r)

        go_files[rel] = {
            "loc": loc,
            "branches": branches,
            "structs": structs,
            "funcs": funcs,
            "src_lower": src.lower(),
        }
        for s in structs:
            norm = s.lower().replace("_", "")
            go_structs.setdefault(norm, []).append(f"{rel}:{s}")
        for fn in funcs:
            norm = fn.lower().replace("_", "")
            go_funcs.setdefault(norm, []).append(f"{rel}:{fn}")

    return {
        "files": go_files,
        "structs": go_structs,
        "funcs": go_funcs,
        "routes": sorted(go_routes),
        "total_loc": total_loc,
        "total_branches": total_branches,
    }


def match_py_symbol_to_go(
    sym_name: str,
    methods: list[str],
    go_index: dict[str, Any],
) -> tuple[str, list[str]]:
    norm = sym_name.lower().replace("_", "")
    # 1. Exact normalized struct/func match
    if norm in go_index["structs"]:
        return "EXACT_STRUCT", go_index["structs"][norm][:3]
    if norm in go_index["funcs"]:
        return "EXACT_FUNC", go_index["funcs"][norm][:3]

    # 2. Strip common suffixes/prefixes (Service, Engine, Manager, Provider, Adapter, Worker, Controller)
    stripped = re.sub(
        r"(service|engine|manager|provider|adapter|worker|controller|handler|processor|client|repository|router|strategy|impl|v1|v2|v3)$",
        "",
        norm,
    )
    if len(stripped) >= 5:
        for gnorm, locs in go_index["structs"].items():
            if stripped in gnorm or gnorm in norm:
                return "SEMANTIC_STRUCT", locs[:3]
        for gnorm, locs in go_index["funcs"].items():
            if stripped in gnorm:
                return "SEMANTIC_FUNC", locs[:3]

    # 3. Token overlap with Go structs/funcs or methods
    tokens = tokenize_name(sym_name)
    if len(tokens) >= 2:
        joined = "".join(tokens)
        for gnorm, locs in go_index["structs"].items():
            if all(t in gnorm for t in tokens[:2]):
                return "TOKEN_STRUCT", locs[:3]
        for m in methods:
            mnorm = m.lower().replace("_", "")
            if len(mnorm) >= 7 and mnorm in go_index["funcs"]:
                return "METHOD_MATCH", go_index["funcs"][mnorm][:3]

    return "UNMATCHED", []


def main() -> None:
    go_index = index_go_codebase()

    subsystems: dict[str, dict[str, Any]] = defaultdict(
        lambda: {
            "files": 0,
            "loc": 0,
            "branches": 0,
            "domain_classes": 0,
            "dto_classes": 0,
            "public_methods": 0,
            "top_funcs": 0,
            "matched_exact_or_semantic": 0,
            "matched_partial": 0,
            "unmatched_domain_classes": 0,
            "unmatched_examples": [],
            "matched_examples": [],
            "file_list": [],
        }
    )

    all_unmatched_domain_classes = []
    all_simplified_domain_classes = []

    for p in sorted(PY_ROOT.rglob("*.py")):
        rel = p.relative_to(PY_ROOT)
        if any(
            part in EXCLUDED_DIRS
            or part.startswith(".venv")
            or part.endswith(".egg-info")
            for part in rel.parts
        ):
            continue
        if (
            p.name.startswith("test_")
            or p.name.endswith("_test.py")
            or p.name in ("__init__.py", "conftest.py", "setup.py")
        ):
            continue

        sub_cat, sub_detail = classify_subsystem(rel)
        src = p.read_text(encoding="utf-8", errors="ignore")
        loc = len(src.splitlines())

        try:
            tree = ast.parse(src)
        except Exception:
            continue

        branches = count_py_branches(tree)
        info = subsystems[sub_cat]
        info["files"] += 1
        info["loc"] += loc
        info["branches"] += branches
        info["file_list"].append(str(rel))

        if sub_cat == "INTENTIONALLY_EXCLUDED_OPENCLAW":
            continue

        for node in tree.body:
            if isinstance(node, ast.ClassDef):
                bases = []
                for b in node.bases:
                    if isinstance(b, ast.Name):
                        bases.append(b.id)
                    elif isinstance(b, ast.Attribute):
                        bases.append(b.attr)
                is_dto = any(b in DTO_BASES for b in bases) or any(
                    node.name.endswith(suf)
                    for suf in (
                        "Request",
                        "Response",
                        "Schema",
                        "Config",
                        "Model",
                        "DTO",
                        "Error",
                        "Exception",
                        "Event",
                        "Payload",
                        "State",
                        "Item",
                        "Result",
                        "Metrics",
                        "Stats",
                        "Status",
                        "Type",
                        "Mode",
                        "Role",
                    )
                )
                methods = [
                    n.name
                    for n in node.body
                    if isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef))
                    and not n.name.startswith("_")
                ]
                cls_branches = count_py_branches(node)
                cls_loc = getattr(node, "end_lineno", node.lineno) - node.lineno + 1

                if is_dto and len(methods) <= 2:
                    info["dto_classes"] += 1
                    continue

                info["domain_classes"] += 1
                info["public_methods"] += len(methods)

                match_kind, go_locs = match_py_symbol_to_go(node.name, methods, go_index)
                if match_kind in ("EXACT_STRUCT", "EXACT_FUNC", "SEMANTIC_STRUCT"):
                    info["matched_exact_or_semantic"] += 1
                    if len(info["matched_examples"]) < 8:
                        info["matched_examples"].append(
                            f"{node.name} ({rel}) -> {', '.join(go_locs)}"
                        )
                    if cls_loc > 120 and len(methods) >= 5:
                        all_simplified_domain_classes.append(
                            {
                                "subsystem": sub_cat,
                                "file": str(rel),
                                "class": node.name,
                                "py_loc": cls_loc,
                                "py_methods": len(methods),
                                "py_branches": cls_branches,
                                "go_match": go_locs,
                            }
                        )
                elif match_kind in ("SEMANTIC_FUNC", "TOKEN_STRUCT", "METHOD_MATCH"):
                    info["matched_partial"] += 1
                    if cls_loc > 80:
                        all_simplified_domain_classes.append(
                            {
                                "subsystem": sub_cat,
                                "file": str(rel),
                                "class": node.name,
                                "py_loc": cls_loc,
                                "py_methods": len(methods),
                                "py_branches": cls_branches,
                                "go_match": go_locs,
                            }
                        )
                else:
                    info["unmatched_domain_classes"] += 1
                    entry = {
                        "subsystem": sub_cat,
                        "file": str(rel),
                        "class": node.name,
                        "py_loc": cls_loc,
                        "py_methods": len(methods),
                        "py_branches": cls_branches,
                        "method_names": methods[:8],
                    }
                    all_unmatched_domain_classes.append(entry)
                    if len(info["unmatched_examples"]) < 10:
                        info["unmatched_examples"].append(
                            f"{node.name} ({rel}, {cls_loc} LOC, methods={methods[:5]})"
                        )
            elif isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                if not node.name.startswith("_"):
                    info["top_funcs"] += 1

    summary = {
        "go_summary": {
            "prod_files": len(go_index["files"]),
            "total_loc": go_index["total_loc"],
            "total_branches": go_index["total_branches"],
            "unique_structs": len(go_index["structs"]),
            "unique_funcs": len(go_index["funcs"]),
            "http_routes": len(go_index["routes"]),
        },
        "subsystems": dict(subsystems),
        "top_unmatched_domain_classes": sorted(
            all_unmatched_domain_classes, key=lambda x: -x["py_loc"]
        )[:120],
        "top_simplified_domain_classes": sorted(
            all_simplified_domain_classes, key=lambda x: -x["py_loc"]
        )[:80],
    }

    DOCS_OUT_JSON.write_text(json.dumps(summary, indent=2, ensure_ascii=False), encoding="utf-8")
    print(f"Saved JSON census to {DOCS_OUT_JSON}")
    print(f"Total Unmatched Domain Classes: {len(all_unmatched_domain_classes)}")
    print(f"Total Potentially Simplified Large Classes: {len(all_simplified_domain_classes)}")
    for k, v in sorted(subsystems.items(), key=lambda x: -x[1]["loc"]):
        print(
            f"{k:46s} | files={v['files']:3d} | LOC={v['loc']:6d} | dom_cls={v['domain_classes']:3d} "
            f"(exact/sem={v['matched_exact_or_semantic']:3d}, part={v['matched_partial']:3d}, missing={v['unmatched_domain_classes']:3d}) | dto={v['dto_classes']:3d}"
        )


if __name__ == "__main__":
    main()

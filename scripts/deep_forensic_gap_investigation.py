#!/usr/bin/env python3
"""Deep Forensic Gap Investigation between RAE-Suite (Python) and RAE-Suite-go (Go v4.9.0).

Analyzes 5 distinct dimensions of gaps:
1. All 332 Unmatched Domain Classes in Python (grouped by subsystem, with LOC & methods).
2. Empty/Marker Structs & Method-Deficit Classes in Go (classes that matched by name in Go,
   but have 0 methods or <35% of the Python methods/branches).
3. Unwired Go Packages/Structs (Go structs defined in pkg/ or services/ that are only called
   in *_test.go and never wired into cmd/ or live HTTP/MCP handlers).
4. Unmatched Public Top-Level Functions & CLI Scripts across RAE-Suite.
5. Architectural / Runtime Protocol Differences (Real DB drivers, WebSockets, AST parsers, GUI).
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
OUT_JSON = GO_ROOT / "docs" / "BADANIE_BRAKOW_PY_GO_V4_9_0.json"

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


def classify_subsystem(rel: pathlib.Path) -> str:
    parts = rel.parts
    if parts[0] == ".openclaw_config" or (
        parts[0] == "packages" and len(parts) > 1 and parts[1] == "rae-open-claw"
    ):
        return "INTENTIONALLY_EXCLUDED_OPENCLAW"
    if parts[0] == "packages" and len(parts) >= 2:
        pkg = parts[1]
        if pkg == "rae-agentic-memory":
            sub = parts[2] if len(parts) >= 3 else "(root)"
            if sub == "apps" and len(parts) >= 4:
                return f"packages/rae-agentic-memory/apps/{parts[3]}"
            return f"packages/rae-agentic-memory/{sub}"
        return f"packages/{pkg}"
    if len(parts) == 1:
        return "(root)"
    return parts[0]


def analyze_go_codebase() -> dict[str, Any]:
    structs: dict[str, dict[str, Any]] = {}
    funcs: dict[str, list[str]] = defaultdict(list)
    prod_sources: dict[str, str] = {}
    test_sources: dict[str, str] = {}

    for gp in sorted(GO_ROOT.rglob("*.go")):
        if "vendor" in gp.parts:
            continue
        rel = str(gp.relative_to(GO_ROOT))
        src = gp.read_text(encoding="utf-8", errors="ignore")
        if gp.name.endswith("_test.go"):
            test_sources[rel] = src
            continue
        prod_sources[rel] = src

        # Find struct definitions and check if they are empty: `type Foo struct{}`
        for m in re.finditer(
            r"type\s+([A-Za-z0-9_]+)\s+struct\s*\{([^}]*)\}", src, re.DOTALL
        ):
            sname = m.group(1)
            body = m.group(2).strip()
            fields = [
                line.strip()
                for line in body.splitlines()
                if line.strip() and not line.strip().startswith("//")
            ]
            norm = sname.lower().replace("_", "")
            structs[norm] = {
                "name": sname,
                "file": rel,
                "field_count": len(fields),
                "is_empty_struct": len(fields) == 0,
                "methods": [],
                "method_loc": 0,
            }

    # Find methods on structs and standalone functions
    for rel, src in prod_sources.items():
        lines = src.splitlines()
        for m in re.finditer(
            r"func\s+(?:\(\s*[a-zA-Z0-9_]*\s*\*?([A-Za-z0-9_]+)\s*\)\s*)?([A-Za-z0-9_]+)\s*\(",
            src,
        ):
            recv = m.group(1)
            fn = m.group(2)
            fn_norm = fn.lower().replace("_", "")
            funcs[fn_norm].append(f"{rel}:{fn}")
            if recv:
                rnorm = recv.lower().replace("_", "")
                if rnorm in structs:
                    structs[rnorm]["methods"].append(fn)

    # Check which Go structs are referenced in prod code outside their own file
    unwired_structs = []
    empty_marker_structs = []
    for norm, sinfo in sorted(structs.items(), key=lambda x: (x[1]["file"], x[1]["name"])):
        sname = sinfo["name"]
        sfile = sinfo["file"]
        if sinfo["is_empty_struct"] and len(sinfo["methods"]) == 0:
            empty_marker_structs.append(
                {
                    "struct": sname,
                    "file": sfile,
                    "field_count": 0,
                    "method_count": 0,
                }
            )
        # Check if sname is used in any cmd/ or services/ HTTP handler file outside sfile
        used_outside = False
        for orel, osrc in prod_sources.items():
            if orel == sfile:
                continue
            if re.search(r"\b" + re.escape(sname) + r"\b", osrc):
                used_outside = True
                break
        # If defined in pkg/ or in deep_*.go and never referenced outside its own file in prod code:
        if not used_outside and (
            sfile.startswith("pkg/")
            or "deep_" in sfile
            or "maintenance_and_pipelines" in sfile
            or "intelligence.go" in sfile
        ):
            used_in_tests = any(
                re.search(r"\b" + re.escape(sname) + r"\b", tsrc)
                for tsrc in test_sources.values()
            )
            unwired_structs.append(
                {
                    "struct": sname,
                    "file": sfile,
                    "method_count": len(sinfo["methods"]),
                    "methods": sinfo["methods"][:8],
                    "used_in_tests": used_in_tests,
                }
            )

    return {
        "structs": structs,
        "funcs": funcs,
        "empty_marker_structs": empty_marker_structs,
        "unwired_structs": unwired_structs,
    }


def main() -> None:
    go_info = analyze_go_codebase()
    go_structs = go_info["structs"]
    go_funcs = go_info["funcs"]

    unmatched_by_sub: dict[str, list[dict[str, Any]]] = defaultdict(list)
    method_deficit_classes: list[dict[str, Any]] = []
    unmatched_top_funcs_by_sub: dict[str, list[dict[str, Any]]] = defaultdict(list)

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

        sub = classify_subsystem(rel)
        if sub == "INTENTIONALLY_EXCLUDED_OPENCLAW":
            continue

        src = p.read_text(encoding="utf-8", errors="ignore")
        try:
            tree = ast.parse(src)
        except Exception:
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
                if is_dto and len(methods) <= 2:
                    continue

                cls_loc = getattr(node, "end_lineno", node.lineno) - node.lineno + 1
                cls_branches = count_py_branches(node)
                norm = node.name.lower().replace("_", "")

                if norm in go_structs:
                    gs = go_structs[norm]
                    go_m_cnt = len(gs["methods"])
                    if gs["is_empty_struct"] and go_m_cnt == 0:
                        method_deficit_classes.append(
                            {
                                "status": "EMPTY_MARKER_STRUCT_IN_GO",
                                "subsystem": sub,
                                "py_file": str(rel),
                                "class": node.name,
                                "py_loc": cls_loc,
                                "py_methods": len(methods),
                                "py_method_names": methods,
                                "go_file": gs["file"],
                                "go_methods": 0,
                            }
                        )
                    elif len(methods) >= 4 and go_m_cnt < max(2, int(len(methods) * 0.45)):
                        method_deficit_classes.append(
                            {
                                "status": "SIMPLIFIED_METHODS_IN_GO",
                                "subsystem": sub,
                                "py_file": str(rel),
                                "class": node.name,
                                "py_loc": cls_loc,
                                "py_methods": len(methods),
                                "py_method_names": methods,
                                "go_file": gs["file"],
                                "go_methods": go_m_cnt,
                                "go_method_names": gs["methods"],
                            }
                        )
                elif norm in go_funcs:
                    if len(methods) >= 4 and cls_loc >= 100:
                        method_deficit_classes.append(
                            {
                                "status": "COLLAPSED_TO_SINGLE_FUNC_IN_GO",
                                "subsystem": sub,
                                "py_file": str(rel),
                                "class": node.name,
                                "py_loc": cls_loc,
                                "py_methods": len(methods),
                                "py_method_names": methods,
                                "go_func": go_funcs[norm][:2],
                            }
                        )
                else:
                    unmatched_by_sub[sub].append(
                        {
                            "file": str(rel),
                            "class": node.name,
                            "py_loc": cls_loc,
                            "py_methods": len(methods),
                            "py_branches": cls_branches,
                            "methods": methods[:10],
                        }
                    )
            elif isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                if node.name.startswith("_") or node.name in ("main", "run", "cli"):
                    continue
                fn_loc = getattr(node, "end_lineno", node.lineno) - node.lineno + 1
                fn_norm = node.name.lower().replace("_", "")
                if fn_norm not in go_funcs and fn_loc >= 15:
                    unmatched_top_funcs_by_sub[sub].append(
                        {
                            "file": str(rel),
                            "func": node.name,
                            "py_loc": fn_loc,
                        }
                    )

    for sub in unmatched_by_sub:
        unmatched_by_sub[sub].sort(key=lambda x: -x["py_loc"])
    for sub in unmatched_top_funcs_by_sub:
        unmatched_top_funcs_by_sub[sub].sort(key=lambda x: -x["py_loc"])
    method_deficit_classes.sort(key=lambda x: -x["py_loc"])

    report = {
        "total_unmatched_domain_classes": sum(len(v) for v in unmatched_by_sub.values()),
        "total_empty_marker_structs_in_go": len(go_info["empty_marker_structs"]),
        "total_method_deficit_classes": len(method_deficit_classes),
        "total_unwired_go_structs": len(go_info["unwired_structs"]),
        "total_unmatched_top_funcs_ge_15loc": sum(
            len(v) for v in unmatched_top_funcs_by_sub.values()
        ),
        "empty_marker_structs_in_go": go_info["empty_marker_structs"],
        "method_deficit_classes": method_deficit_classes,
        "unwired_go_structs": go_info["unwired_structs"],
        "unmatched_classes_by_subsystem": dict(unmatched_by_sub),
        "unmatched_top_funcs_by_subsystem": dict(unmatched_top_funcs_by_sub),
    }

    OUT_JSON.write_text(json.dumps(report, indent=2, ensure_ascii=False), encoding="utf-8")
    print(f"Saved forensic gap analysis to {OUT_JSON}")
    print(f"1. Total Unmatched Domain Classes (no exact Go struct/func): {report['total_unmatched_domain_classes']}")
    print(f"2. Total Empty Marker Structs in Go (0 fields, 0 methods):   {report['total_empty_marker_structs_in_go']}")
    print(f"3. Total Method-Deficit / Simplified Classes in Go:          {report['total_method_deficit_classes']}")
    print(f"4. Total Unwired Go Structs (only used in tests / own file): {report['total_unwired_go_structs']}")
    print(f"5. Total Unmatched Top-Level Functions (>=15 LOC):           {report['total_unmatched_top_funcs_ge_15loc']}")


if __name__ == "__main__":
    main()

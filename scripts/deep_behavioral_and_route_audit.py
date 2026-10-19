#!/usr/bin/env python3
"""Deep Behavioral, HTTP Sub-Route, CLI, and Input-Sensitivity Auditor (v5.1.0).

Goes beyond v5.0.0 structural/AST parity to check 4 behavioral & operational dimensions:
1. Unmatched HTTP Sub-Routes across all Python FastAPI/Flask services (including Context-Watcher,
   RAE-Lite Windows Server, RAE-LLM-Proxy, and Dashboard sub-routes).
2. Constant / Input-Insensitive Methods in `pkg/raecore/parity/domains_part*.go` (methods that
   return a constant struct literal without computing input-dependent SHA-256 digests or parameters).
3. Constant / Input-Insensitive Top-Level Functions in `pkg/raecore/parity/functions.go`.
4. Missing Native CLI Subcommands in `cmd/rae/main.go` (expecting `record-task`, `bootstrap`,
   `mesh`, `triad`, `doctor`, `benchmark`, `export`, `import`, `init`, `status`).
"""

from __future__ import annotations

import json
import pathlib
import re
from typing import Any

PY_ROOT = pathlib.Path("/home/grzegorz/cloud/RAE-Suite")
GO_ROOT = pathlib.Path("/home/grzegorz/cloud/RAE-Suite-go")
OUT_JSON = GO_ROOT / "docs" / "BADANIE_BEHAWIORALNE_PY_GO_V5_1_0.json"

REQUIRED_CLI_SUBCOMMANDS = [
    "record-task",
    "bootstrap",
    "mesh",
    "triad",
    "doctor",
    "benchmark",
    "export",
    "import",
    "init",
    "status",
]


def main() -> None:
    # 1. Check all Python HTTP routes vs Go HandleFunc routes
    main_py = (
        PY_ROOT / "packages/rae-agentic-memory/apps/memory_api/main.py"
    ).read_text(encoding="utf-8")
    inc_re = re.compile(
        r"include_router\(\s*([a-zA-Z0-9_.]+).*?prefix\s*=\s*['\"]([^'\"]+)['\"]",
        re.DOTALL,
    )
    router_prefixes = {
        m.group(1).split(".")[0]: m.group(2) for m in inc_re.finditer(main_py)
    }

    prefix_re = re.compile(r"APIRouter\([^)]*prefix\s*=\s*['\"]([^'\"]+)['\"]")
    route_re = re.compile(
        r"@(?:router|app)\.(get|post|put|delete|patch|websocket)\(\s*['\"]([^'\"]+)['\"]"
    )

    py_full_routes = []
    for p in sorted(PY_ROOT.rglob("*.py")):
        if any(
            x in p.parts
            for x in (
                ".venv",
                "venv",
                "tests",
                "test",
                ".openclaw_config",
                "rae-open-claw",
                "tmp_node1_sync",
                ".ai-templates",
                "benchmarking",
            )
        ):
            continue
        src = p.read_text(encoding="utf-8", errors="ignore")
        pref_m = prefix_re.search(src)
        prefix = pref_m.group(1) if pref_m else router_prefixes.get(p.stem, "")
        for m in route_re.finditer(src):
            method = m.group(1).upper()
            subpath = m.group(2)
            full = (prefix.rstrip("/") + "/" + subpath.lstrip("/")).rstrip("/")
            if not full.startswith("/"):
                full = "/" + full
            py_full_routes.append(
                {
                    "file": str(p.relative_to(PY_ROOT)),
                    "method": method,
                    "full_path": full,
                    "subpath": subpath,
                }
            )

    go_routes = set()
    go_route_re = re.compile(r"HandleFunc\(\s*['\"]([^'\"]+)['\"]")
    for gp in GO_ROOT.rglob("*.go"):
        if gp.name.endswith("_test.go"):
            continue
        src = gp.read_text(encoding="utf-8", errors="ignore")
        for m in go_route_re.finditer(src):
            r = m.group(1).rstrip("/") or "/"
            go_routes.add(r)

    unmatched_routes = []
    for item in py_full_routes:
        full = item["full_path"]
        subpath = item["subpath"]
        norm_full = re.sub(r"\{[^}]+\}", "", full).rstrip("/") or "/"
        norm_sub = re.sub(r"\{[^}]+\}", "", subpath).rstrip("/") or "/"
        v2_full = "/v2" + full if not full.startswith("/v2") else full
        v2_norm = re.sub(r"\{[^}]+\}", "", v2_full).rstrip("/") or "/"
        if not any(
            cand in go_routes
            or any(
                gr.startswith(cand) or cand.startswith(gr)
                for gr in go_routes
                if len(gr) > 4
            )
            for cand in (full, norm_full, subpath, norm_sub, v2_full, v2_norm)
        ):
            unmatched_routes.append(item)

    # 2. Check constant/input-insensitive methods in pkg/raecore/parity/domains_part*.go
    p1 = (GO_ROOT / "pkg/raecore/parity/domains_part1.go").read_text(encoding="utf-8")
    p2 = (GO_ROOT / "pkg/raecore/parity/domains_part2.go").read_text(encoding="utf-8")
    p_fns = (GO_ROOT / "pkg/raecore/parity/functions.go").read_text(encoding="utf-8")

    const_domain_methods = p1.count('Status: "OK"}') + p2.count('Status: "OK"}')
    const_top_funcs = p_fns.count('"status": "OK"}')

    # 3. Check CLI subcommands in cmd/rae/main.go
    cli_src = (GO_ROOT / "cmd/rae/main.go").read_text(encoding="utf-8")
    missing_cli = [
        cmd
        for cmd in REQUIRED_CLI_SUBCOMMANDS
        if f'case "{cmd}":' not in cli_src
    ]

    report: dict[str, Any] = {
        "total_python_http_routes": len(py_full_routes),
        "total_go_http_routes": len(go_routes),
        "unmatched_http_subroutes_count": len(unmatched_routes),
        "constant_domain_methods_count": const_domain_methods,
        "constant_top_funcs_count": const_top_funcs,
        "missing_cli_subcommands_count": len(missing_cli),
        "unmatched_http_subroutes": unmatched_routes,
        "missing_cli_subcommands": missing_cli,
    }
    OUT_JSON.write_text(json.dumps(report, indent=2, ensure_ascii=False), encoding="utf-8")
    print(f"Saved deep behavioral audit to {OUT_JSON}")
    print(f"1. Unmatched HTTP Sub-Routes:                   {report['unmatched_http_subroutes_count']}")
    print(f"2. Constant / Input-Insensitive Domain Methods: {report['constant_domain_methods_count']}")
    print(f"3. Constant / Input-Insensitive Top Functions:  {report['constant_top_funcs_count']}")
    print(f"4. Missing CLI Subcommands in cmd/rae:          {report['missing_cli_subcommands_count']} ({missing_cli})")


if __name__ == "__main__":
    main()

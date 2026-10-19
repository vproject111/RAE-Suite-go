#!/usr/bin/env python3
"""
Reusable 5-Stage Automated Parity & Anti-Stub Auditor (Python <-> Go)
=====================================================================
Can be executed repeatedly via:
    python3 scripts/reusable_5stage_parity_auditor.py

Stages:
  1. AST Symbol & Endpoint Mapping Matrix (Python AST <-> Go AST)
  2. Static Anti-Stub & Dead-Logic Scanner on Go Handlers
  3. Differential Input-Sensitivity Probing: f(x1) != f(x2)
  4. Stateful End-to-End Lifecycle Invariant Chain (7 steps)
  5. Database Schema, SQL DDL & Environment Variable Contract Audit
"""

from __future__ import annotations

import ast
import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

GO_ROOT = Path(__file__).resolve().parent.parent
PY_ROOT = GO_ROOT.parent / "RAE-Suite"
CRL_ROOT = GO_ROOT.parent / "RAE-CRL"


def http_json(method: str, url: str, payload: dict[str, Any] | None = None) -> tuple[int, dict[str, Any]]:
    data = json.dumps(payload).encode("utf-8") if payload is not None else None
    headers = {"Content-Type": "application/json"} if payload is not None else {}
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=5.0) as resp:
            raw = resp.read().decode("utf-8", errors="replace")
            try:
                return resp.status, json.loads(raw)
            except Exception:
                return resp.status, {"_raw": raw}
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", errors="replace")
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {"_raw": raw}
    except Exception as e:
        return 0, {"_error": str(e)}


# =============================================================================
# STAGE 1: AST Symbol & Endpoint Mapping Matrix (Python AST <-> Go AST)
# =============================================================================
CANONICAL_DOMAIN_CAPABILITIES = [
    ("AuditAccounting5Pillar", ["audit_accounting.py"], ["pkg/raecore/accounting/ledger.go"], ["Ledger", "RecordInvocation"]),
    ("BatchCoalescingEngine", ["batch_engine.py"], ["pkg/raecore/economy/engine.go"], ["BatchOptimizationEngine", "BatchTask"]),
    ("CognitivePlannerToTMCTS", ["cognitive_planner.py"], ["pkg/raecore/cognition/planner.go"], ["CognitivePlanner", "CreatePlan", "SelectUCT"]),
    ("SwarmConsensusAndCouncil", ["swarm_consensus.py", "council_selector.py"], ["pkg/raecore/cognition/planner.go"], ["SwarmConsensusEngine", "DynamicCouncilSelector", "SelectCouncilQuorum"]),
    ("ContextBrokerAndTrust", ["context_broker.py", "context_trust.py"], ["pkg/raecore/economy/engine.go"], ["ContextBroker", "ContextTrustEvaluator", "EvaluateTrust"]),
    ("SagaStreamSpeculativeClaimReplay", ["saga_coordinator.py", "claim_check.py", "safe_replay.py"], ["pkg/raecore/workflows/engines.go"], ["SagaCoordinator", "ClaimCheckManager", "SafeReplayEngine"]),
    ("A2AProtocolAndPIIScrubber", ["a2a_protocol.py", "pii_scrubber.py"], ["pkg/raecore/workflows/engines.go"], ["A2AProtocolAdapter", "IngestionPIIScrubber", "ScrubText"]),
    ("ShadowCuriosityToolForgeThinRuntime", ["shadow_evaluator.py", "curiosity_engine.py", "tool_forge.py"], ["pkg/raecore/workflows/engines.go"], ["ShadowEvaluator", "CuriosityEngine", "ToolForge", "ThinDeviceRuntime"]),
    ("ImprovementPlaneAndOps", ["improvement_plane.py", "cluster_manager.py", "circuit_breaker.py"], ["pkg/raecore/ops/engines.go"], ["ImprovementPlane", "ClusterManager", "InfraReconciler"]),
    ("HermesOpenCodeBridge", ["dual_escalation.py"], ["pkg/raecore/ops/hermes_opencode.go"], ["HermesOpenCodeBridge", "Dispatch"]),
    ("GitOpsPhasesGlobalSearch", ["gitops_daemon.py", "project_phase_analyzer.py", "portal_global_search.py"], ["pkg/raecore/ops/gitops_phases_search.go"], ["GitOpsDaemon", "AnalyzeProjectPhases", "ExecuteLiveGlobalSearch"]),
    ("InformationBottleneckSelector", ["ib_selector.py"], ["pkg/raecore/math/ib_selector.go"], ["InformationBottleneckSelector", "SelectContext"]),
    ("TemporalGraphAndSnapshots", ["temporal_graph.py"], ["pkg/raecore/graph/temporal.go"], ["TemporalGraphService", "ReconstructGraphAtTime", "CompareGraphs"]),
    ("EntityResolutionGraph", ["entity_resolution.py"], ["pkg/raecore/graph/entity_resolution.go"], ["RunClusteringAndMerging", "NormalizeEntityKey"]),
    ("GraphTopologyAlgorithms", ["graph_algorithms.py", "graph_topology_engine.py"], ["pkg/raecore/graph/algorithms.go"], ["ComputePageRank", "ComputeBetweennessCentrality", "ComputeClusteringCoefficients", "ComputeModularityQ"]),
    ("MarkovDecisionProcessMDP", ["state.py", "actions.py", "reward.py", "action_executor.py"], ["pkg/raecore/mdp/mdp.go"], ["RAEState", "Engine", "SelectOptimalAction", "Step"]),
    ("ConsolidationFSMDreamingCommunities", ["consolidation_fsm.py", "community_summarizer.py", "memory_maintenance.py"], ["pkg/raecore/reflection/lifecycle_dreaming.go"], ["ConsolidationFSM", "RunDreamingCycle", "SummarizeCommunities", "SummarizeSession"]),
    ("GovernanceRulesPolicyRetentionBudget", ["rules_engine.py", "policy_versioning_service.py", "retention_service.py", "budget_service.py"], ["pkg/raecore/security/governance_engines.go"], ["RulesEngine", "PolicyVersioningService", "RetentionService", "BudgetService"]),
    ("ContextProvenanceISO42001", ["context_provenance.py"], ["pkg/raecore/security/provenance.go"], ["ContextProvenanceService", "CreateDecisionContext", "GetDecisionLineage"]),
    ("SixFactorImportanceAndCostController", ["importance_scoring.py", "cost_controller.py"], ["pkg/raecore/math/importance_cost.go"], ["ComputeSixFactorImportance", "CalculateModelCost"]),
    ("QueryClassifierStrategyRouterFusion", ["classifier.py", "router.py", "fusion.py", "optimizer.py"], ["pkg/raecore/search/classifier_router_fusion.go"], ["ClassifyQuery", "BuildRoutingPlan", "ConfidenceWeightedFuse"]),
    ("IREvaluatorDriftAndABTesting", ["evaluation_service.py", "drift_detector.py"], ["pkg/raecore/search/evaluator_drift.go"], ["EvaluateSearchIR", "ComputeKLAndJSDDrift", "EvaluateABTestSignificance"]),
    ("EvidenceConflictAndTwoPassRewriter", ["conflict_detector.py", "query_rewriter.py"], ["pkg/raecore/search/conflict_and_rewriter.go"], ["DetectEvidenceConflicts", "CreateRewritePlan"]),
    ("AgnosticDatabaseBroker5Backends", ["broker.py", "postgres.py", "qdrant.py", "redis.py"], ["pkg/drivers/agnostic/broker.go"], ["AgnosticStorageBroker", "SwitchBackend", "Migrate", "ExportSQL"]),
    ("RAEMesh16BucketMerkleAndE2ECrypto", ["sync_protocol.py", "encryption.py"], ["pkg/raemesh/mesh.go", "pkg/raemesh/encryption.go"], ["ComputeMerkleTree", "DiffMerkleBuckets", "E2EEncryption"]),
    ("PhoenixASTRecipesBehaviorPostMortemMeta", ["behavior.py", "post_mortem.py", "system_model.py", "longitudinal.py"], ["services/phoenix/service.go", "services/phoenix/behavior_and_postmortem.go", "services/phoenix/meta_reflection.go", "services/phoenix/recipes_exec.go"], ["CompareBehavior6D", "AnalyzePostMortem", "ApplyRecipe"]),
    ("Quality210CatalogBinaryPDFMultiAST", ["policy_catalog.py", "dtp_validator.py", "calculator_verifier.py"], ["services/quality/deep_engines.go", "services/quality/binary_pdf_and_ast.go"], ["ValidatePDFDocument", "VerifyCalculatorTiers", "ParseBinaryPDFAndValidate", "AnalyzeMultiLangAST"]),
    ("HiveAtomicSlicerV76AndSupervisorV2", ["atomic_slicer.py", "supervisor.py"], ["services/hive/slicer_and_supervisor.go"], ["SliceCodeAtomically", "CheckAndHealSupervisor"]),
    ("LabFailureMiningAndSafeRollout", ["failure_mining_engine.py", "safe_rollout_manager.py"], ["services/lab/mining_rollout.go"], ["MineFailurePatterns", "EvaluateSafeRollout"]),
    ("ConsistencyServiceReconciliation", ["consistency_service.py"], ["services/memory/v47_deep_routes.go"], ["ConsistencyReport", "VerifyAndRepairConsistency"]),
]


def run_stage1_ast_inventory() -> dict[str, Any]:
    py_classes = 0
    py_functions = 0
    if PY_ROOT.exists():
        for p in PY_ROOT.rglob("*.py"):
            s = str(p)
            if any(x in s for x in (".venv", "venv", "site-packages", "__pycache__", "node_modules", "alembic", "tests", "work_dir")):
                continue
            try:
                tree = ast.parse(p.read_text(encoding="utf-8", errors="ignore"))
                for node in ast.walk(tree):
                    if isinstance(node, ast.ClassDef):
                        py_classes += 1
                    elif isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                        py_functions += 1
            except Exception:
                pass

    go_structs = 0
    go_funcs = 0
    go_routes: set[str] = set()
    struct_re = re.compile(r"^type\s+([A-Za-z0-9_]+)\s+struct\b", re.MULTILINE)
    func_re = re.compile(r"^func\s+(?:\([^)]+\)\s+)?([A-Za-z0-9_]+)\s*\(", re.MULTILINE)
    route_re = re.compile(r'HandleFunc\(\s*"([^"]+)"')

    for gp in GO_ROOT.rglob("*.go"):
        txt = gp.read_text(encoding="utf-8", errors="ignore")
        go_structs += len(struct_re.findall(txt))
        go_funcs += len(func_re.findall(txt))
        for m in route_re.findall(txt):
            go_routes.add(m)

    missing_caps = []
    mapped_caps = 0
    for cap_name, py_files, go_files, req_symbols in CANONICAL_DOMAIN_CAPABILITIES:
        missing_files = [gf for gf in go_files if not (GO_ROOT / gf).exists()]
        combined_src = ""
        for gf in go_files:
            p = GO_ROOT / gf
            if p.exists():
                combined_src += "\n" + p.read_text(encoding="utf-8", errors="ignore")
        missing_syms = [sym for sym in req_symbols if sym not in combined_src]
        if not missing_files and not missing_syms:
            mapped_caps += 1
        else:
            missing_caps.append({
                "capability": cap_name,
                "missing_go_files": missing_files,
                "missing_go_symbols": missing_syms,
            })

    return {
        "stage": "1. AST Symbol & Endpoint Mapping Matrix",
        "passed": len(missing_caps) == 0 and len(go_routes) >= 175,
        "py_first_party_classes": py_classes,
        "py_first_party_functions": py_functions,
        "go_structs": go_structs,
        "go_functions": go_funcs,
        "go_registered_http_routes": len(go_routes),
        "domain_capabilities_mapped": f"{mapped_caps}/{len(CANONICAL_DOMAIN_CAPABILITIES)}",
        "missing_capabilities": missing_caps,
    }


# =============================================================================
# STAGE 2: Static Anti-Stub & Dead-Logic Scanner on Go Handlers
# =============================================================================
ALLOWED_STATIC_INFO_PATHS = {
    "/health",
    "/ready",
    "/live",
    "/docs",
    "/openapi.json",
    "/mcp/sse",
    "/manifest.json",
    "/sw.js",
    "/logout",
    "/v2/graph/health",
    "/v2/automation/events/types",
    "/v2/automation/templates",
    "/v2/automation/health",
    "/v2/automation/info",
    "/v2/evaluation/metrics/supported",
    "/v2/evaluation/drift/severity-levels",
    "/v2/evaluation/quality/thresholds",
    "/v2/evaluation/benchmark/suites",
    "/v2/evaluation/health",
    "/v2/evaluation/info",
    "/v2/search/weights/profiles",
    "/v2/search/health",
    "/v2/search/info",
    "/v2/dashboard/visualizations",
    "/v2/dashboard/health",
    "/v2/dashboard/health/simple",
    "/v2/dashboard/info",
    "/v2/dashboard/compliance/rls-status",
    "/v2/tuning/theories",
    "/system/tables",
    "/v2/portal/views",
    "/v2/quality/manifest",
    "/v2/phoenix/recipes",
    "/metrics/prometheus",
}


def run_stage2_static_antistub_scanner() -> dict[str, Any]:
    findings: list[dict[str, Any]] = []
    handler_re = re.compile(
        r'mux\.HandleFunc\(\s*"([^"]+)"\s*,\s*func\(\s*\w+\s+http\.ResponseWriter\s*,\s*(\w+)\s*\*http\.Request\s*\)\s*\{',
        re.MULTILINE,
    )

    total_handlers = 0
    for gp in (GO_ROOT / "services").rglob("*.go"):
        if gp.name.endswith("_test.go"):
            continue
        txt = gp.read_text(encoding="utf-8", errors="ignore")
        rel_path = str(gp.relative_to(GO_ROOT))

        # Check for forbidden OpenClawEscalation view in portal
        if "OpenClawEscalation" in txt:
            findings.append({
                "file": rel_path,
                "route": "PortalViews",
                "issue": "FORBIDDEN_OPENCLAW_VIEW_NAME: 'OpenClawEscalation' present instead of 'HermesOpenCodeEscalation'",
            })

        for match in handler_re.finditer(txt):
            total_handlers += 1
            route = match.group(1)
            req_var = match.group(2)
            start_idx = match.end()
            # extract balanced braces of handler body
            depth = 1
            i = start_idx
            while i < len(txt) and depth > 0:
                if txt[i] == "{":
                    depth += 1
                elif txt[i] == "}":
                    depth -= 1
                i += 1
            body = txt[start_idx : i - 1]

            if route in ALLOWED_STATIC_INFO_PATHS:
                continue

            # Rule 1: handler ignores *http.Request (`_ *http.Request`) AND doesn't call any service/store/engine method
            has_service_call = any(
                tok in body
                for tok in (
                    "s.",
                    "ext.",
                    "search.",
                    "graph.",
                    "raemath.",
                    "mdpEngine.",
                    "fsm.",
                    "rulesEng.",
                    "policyVer.",
                    "retentionSvc.",
                    "budgetSvc.",
                    "ops.",
                    "Evaluate",
                    "Build",
                    "Analyze",
                    "Detect",
                    "Verify",
                    "Run",
                    "Compute",
                )
            )
            if req_var == "_" and not has_service_call:
                findings.append({
                    "file": rel_path,
                    "route": route,
                    "issue": "STATIC_LITERAL_STUB: ignores *http.Request and makes no service/store/engine call",
                })

            # Rule 2: decodes req from r.Body, but never references req fields or passes req to a function
            if ".Decode(&req)" in body:
                after_decode = body.split(".Decode(&req)", 1)[1]
                uses_req = ("req." in after_decode) or (", req" in after_decode) or ("(req)" in after_decode) or ("req[" in after_decode)
                if not uses_req:
                    findings.append({
                        "file": rel_path,
                        "route": route,
                        "issue": "UNUSED_DECODED_REQUEST: decodes &req from JSON body but never uses req in logic",
                    })

    return {
        "stage": "2. Static Anti-Stub & Dead-Logic Scanner",
        "passed": len(findings) == 0,
        "total_handlers_scanned": total_handlers,
        "stubs_or_dead_logic_found": len(findings),
        "findings": findings,
    }


# =============================================================================
# STAGE 3: Differential Input-Sensitivity Probing f(x1) != f(x2)
# =============================================================================
SENSITIVITY_PROBES = [
    (
        "Memory `/v2/search/analyze` (QueryClassifier sensitivity)",
        "POST",
        18000,
        "/v2/search/analyze",
        {"query": "when did the deployment happen yesterday?"},
        {"query": "how is node A connected to node B in the graph?"},
    ),
    (
        "Memory `/v2/search/weights/calculate` (Dynamic weight routing)",
        "POST",
        18000,
        "/v2/search/weights/calculate",
        {"query": "step 1 how to configure postgres"},
        {"query": "when was the last commit timestamp"},
    ),
    (
        "Memory `/v2/evaluation/drift/detect` (KL/JSD drift sensitivity)",
        "POST",
        18000,
        "/v2/evaluation/drift/detect",
        {"baseline": [0.5, 0.5], "current": [0.5, 0.5]},
        {"baseline": [0.9, 0.1], "current": [0.1, 0.9]},
    ),
    (
        "Memory `/v2/evaluation/ab-test/compare` (Welch t-test sensitivity)",
        "POST",
        18000,
        "/v2/evaluation/ab-test/compare",
        {"arm_a": "classic_ib", "rewards_a": [0.5, 0.5, 0.5], "arm_b": "fluid_100", "rewards_b": [0.5, 0.5, 0.5]},
        {"arm_a": "classic_ib", "rewards_a": [0.2, 0.21, 0.19], "arm_b": "fluid_100", "rewards_b": [0.95, 0.96, 0.94]},
    ),
    (
        "Memory `/v2/mdp/policy/select` (MDP optimal action under budget exhaustion vs healthy)",
        "POST",
        18000,
        "/v2/mdp/policy/select",
        {"max_context_tokens": 8192, "remaining_tokens": 100000, "remaining_usd": 10.0},
        {"max_context_tokens": 8192, "remaining_tokens": 0, "remaining_usd": 0.0},
    ),
    (
        "Memory `/v2/graph/algorithms/pagerank` (PageRank topology sensitivity)",
        "POST",
        18000,
        "/v2/graph/algorithms/pagerank",
        {"adjacency": {"A": ["B"], "B": ["A"]}},
        {"adjacency": {"A": ["C"], "B": ["C"], "D": ["C"], "C": ["A"]}},
    ),
    (
        "Memory `/v2/governance/budget/check` (Budget 80% WARN vs 100% HARD_BLOCK)",
        "POST",
        18000,
        "/v2/governance/budget/check",
        {"tenant_id": "probe-t1", "daily_limit_usd": 10.0, "daily_token_limit": 100000, "cost_usd": 1.0, "tokens": 1000},
        {"tenant_id": "probe-t2", "daily_limit_usd": 10.0, "daily_token_limit": 100000, "cost_usd": 12.0, "tokens": 1000},
    ),
    (
        "Memory `/v2/math/importance/six-factor` (6-factor importance trivial vs critical)",
        "POST",
        18000,
        "/v2/math/importance/six-factor",
        {"item": {"id": "p1", "content": "ping ok", "layer": "episodic"}, "graph_degree": 0},
        {"item": {"id": "p2", "content": "Critical security CVE architecture invariant restricted leak", "layer": "reflective", "info_class": "restricted", "tags": ["security", "cve", "iso"]}, "graph_degree": 8},
    ),
    (
        "Memory `/v2/math/cost/calculate` (CostController tier0 vs tier3)",
        "POST",
        18000,
        "/v2/math/cost/calculate",
        {"model_id": "deepseek-r1:8b", "prompt_tokens": 100000, "completion_tokens": 50000},
        {"model_id": "anthropic/claude-sonnet-5", "prompt_tokens": 100000, "completion_tokens": 50000},
    ),
    (
        "Orchestrator `/v2/fabric/route` (Workload routing sensitivity)",
        "POST",
        18009,
        "/v2/fabric/route",
        {"workload": "heavy_ml_gpu"},
        {"workload": "stress_db"},
    ),
    (
        "Orchestrator `/v2/knowledge/resolve` (Knowledge sufficiency sensitivity)",
        "POST",
        18009,
        "/v2/knowledge/resolve",
        {"query": "ISO 42001", "project": "RAE-Suite-go", "min_confidence": 0.5},
        {"query": "nonexistent_quantum_flux_xyz_987654", "project": "empty-proj", "min_confidence": 0.99},
    ),
    (
        "Quality `/v2/quality/risk-coverage` (Risk coverage sensitivity to executed_tests)",
        "POST",
        18010,
        "/v2/quality/risk-coverage",
        {"executed_tests": ["test_calc_monotonicity", "test_OrderCalc_golden_set", "test_auth_jwt_expiry", "test_keycloak_rbac", "test_stripe_webhook_signature", "test_idempotency_key", "test_dtp_bleed_3mm", "test_cmyk_tac_limit"]},
        {"executed_tests": ["test_calc_monotonicity"]},
    ),
    (
        "Quality `/v2/quality/defect-metrics` (Defect density sensitivity to code/findings)",
        "POST",
        18010,
        "/v2/quality/defect-metrics",
        {"code": "package main\nfunc Clean() int { return 1 }\n"},
        {"code": "package main\nconst p = \"/home/grzegorz/secret\"\n", "escaped_defects": 2},
    ),
    (
        "Quality `/v2/quality/predictive-plan` (Hotspot prediction sensitivity to files)",
        "POST",
        18010,
        "/v2/quality/predictive-plan",
        {"files": ["pkg/raecore/math/controller.go"]},
        {"files": ["services/memory/extended_routes.go", "services/orchestrator/service.go"]},
    ),
    (
        "Phoenix `/v2/phoenix/create` (Module synthesis sensitivity to spec)",
        "POST",
        18012,
        "/v2/phoenix/create",
        {"module_name": " billing_calc ", "spec": "Calculate VAT and net total"},
        {"module_name": " auth_guard ", "spec": "Validate HMAC JWT token signature"},
    ),
    (
        "Phoenix `/sessions/analyze` (Session anomaly sensitivity to error traces)",
        "POST",
        18012,
        "/sessions/analyze",
        {"session_id": "s-clean", "error_traces": []},
        {"session_id": "s-fail", "error_traces": ["panic: nil pointer dereference", "timeout exceeded"]},
    ),
    (
        "Portal `/v2/portal/global-search` (Portal global search sensitivity to query)",
        "POST",
        18082,
        "/v2/portal/global-search",
        {"query": "ISO", "project": "RAE-Suite-go"},
        {"query": "nonexistent_term_zzz_99999", "project": "RAE-Suite-go"},
    ),
]


def run_stage3_sensitivity_probing() -> dict[str, Any]:
    insensitive: list[dict[str, Any]] = []
    passed_count = 0

    for label, method, port, path, x1, x2 in SENSITIVITY_PROBES:
        s1, out1 = http_json(method, f"http://127.0.0.1:{port}{path}", x1)
        s2, out2 = http_json(method, f"http://127.0.0.1:{port}{path}", x2)
        if s1 != 200 or s2 != 200:
            insensitive.append({
                "probe": label,
                "path": path,
                "reason": f"HTTP status error: s1={s1}, s2={s2}",
            })
            continue
        # Remove volatile timestamps/UUIDs before comparing
        for d in (out1, out2):
            for k in ("timestamp", "generated_at", "created_at", "executed_at", "id", "request_id"):
                d.pop(k, None)
        if json.dumps(out1, sort_keys=True) == json.dumps(out2, sort_keys=True):
            insensitive.append({
                "probe": label,
                "path": path,
                "reason": "INPUT_INSENSITIVE_STUB: f(x1) == f(x2) for distinct inputs",
                "output": out1,
            })
        else:
            passed_count += 1

    return {
        "stage": "3. Differential Input-Sensitivity Probing f(x1) != f(x2)",
        "passed": len(insensitive) == 0,
        "probes_total": len(SENSITIVITY_PROBES),
        "probes_passed": passed_count,
        "insensitive_endpoints": insensitive,
    }


# =============================================================================
# STAGE 4: Stateful End-to-End Workflow Invariant Chain
# =============================================================================
def run_stage4_stateful_invariants() -> dict[str, Any]:
    steps: list[dict[str, Any]] = []

    # Step 1: Merkle root before ingestion
    _, m_before = http_json("GET", "http://127.0.0.1:18000/v2/mesh/merkle-tree?project=stateful-audit")
    root_before = m_before.get("root_hash", "")

    # Step 2: Semantic document ingestion -> creates chunks + graph triples
    s_ing, ing_out = http_json(
        "POST",
        "http://127.0.0.1:18000/v2/memories/ingest-document",
        {
            "title": "Stateful E2E Architecture Spec",
            "text": "# Core Engine\nRAEMemory connects to QdrantVectorStore.\n## Security\nHardFramesGuard enforces ISO42001.",
            "project": "stateful-audit",
            "layer": "semantic",
        },
    )
    step2_ok = s_ing == 200 and ing_out.get("chunks_stored", 0) >= 2
    steps.append({"step": "1. Semantic Document Ingestion creates chunks & graph triples", "passed": step2_ok, "detail": ing_out})

    # Step 3: Store memory with Importance=0 -> verify 6-factor auto-scoring sets Importance > 0.30
    s_mem, mem_out = http_json(
        "POST",
        "http://127.0.0.1:18000/v2/memories/",
        {
            "id": "gdpr-target-mem-01",
            "tenant_id": "tenant-e2e",
            "project": "stateful-audit",
            "layer": "episodic",
            "content": "Critical security incident for user anna.nowak@example.com in production",
            "tags": ["security", "incident"],
            "importance": 0.0,
        },
    )
    auto_imp = float(mem_out.get("importance", 0.0))
    step3_ok = s_mem == 200 and auto_imp >= 0.40
    steps.append({"step": "2. StoreMemory with Importance=0 auto-computes 6-Factor Importance", "passed": step3_ok, "importance": auto_imp})

    # Step 4: Verify Merkle tree root hash mutated after storing memories
    _, m_after = http_json("GET", "http://127.0.0.1:18000/v2/mesh/merkle-tree?project=stateful-audit")
    root_after = m_after.get("root_hash", "")
    step4_ok = bool(root_after) and root_after != root_before
    steps.append({"step": "3. 16-Bucket Merkle Tree root_hash mutates on memory write", "passed": step4_ok, "before": root_before[:16], "after": root_after[:16]})

    # Step 5: ConsolidationFSM + Dreaming cycle generates reflective insights
    s_drm, drm_out = http_json("POST", "http://127.0.0.1:18000/v2/dreaming/run", {"min_importance": 0.3})
    step5_ok = s_drm == 200 and drm_out.get("insights_generated", 0) >= 1
    steps.append({"step": "4. DreamingWorker synthesizes cross-episode insights into Reflective layer", "passed": step5_ok, "insights": drm_out.get("insights_generated", 0)})

    # Step 6: GDPR Right-to-be-Forgotten erases anna.nowak@example.com and emits SHA-256 proof
    s_gdpr, gdpr_out = http_json(
        "POST",
        "http://127.0.0.1:18000/v2/governance/retention/gdpr-erase",
        {"tenant_id": "tenant-e2e", "subject_identifier": "anna.nowak@example.com"},
    )
    step6_ok = (
        s_gdpr == 200
        and gdpr_out.get("deleted_memory_count", 0) >= 1
        and len(gdpr_out.get("proof_hash_sha256", "")) == 64
    )
    steps.append({"step": "5. GDPR Art. 17 Right-to-be-Forgotten erases PII memory & signs SHA-256 proof", "passed": step6_ok, "audit": gdpr_out})

    # Step 7: Verify erased memory is no longer returned by search and results is [] (not null)
    _, q_after = http_json("GET", "http://127.0.0.1:18000/v2/memories/?query=anna.nowak@example.com&project=stateful-audit")
    raw_results = q_after.get("results")
    results_list = raw_results if isinstance(raw_results, list) else []
    rem_hits = [r for r in results_list if "anna.nowak@example.com" in json.dumps(r)]
    step7_ok = isinstance(raw_results, list) and len(rem_hits) == 0
    steps.append({"step": "6. Post-erasure query confirms zero residual PII records & non-null JSON array", "passed": step7_ok, "residual_hits": len(rem_hits), "results_is_list": isinstance(raw_results, list)})

    all_ok = all(s["passed"] for s in steps)
    return {
        "stage": "4. Stateful End-to-End Workflow Invariant Chain",
        "passed": all_ok,
        "steps_passed": f"{sum(1 for s in steps if s['passed'])}/{len(steps)}",
        "steps": steps,
    }


# =============================================================================
# STAGE 5: Database Schema, SQL DDL & Environment Variable Contract Audit
# =============================================================================
REQUIRED_ENV_VARS = [
    "RAE_STORAGE_BACKEND",
    "RAE_DB_MODE",
    "POSTGRES_URL",
    "QDRANT_URL",
    "REDIS_ADDR",
    "PHOENIX_MAX_ATTEMPTS",
    "OLLAMA_API_URL",
    "OPENROUTER_API_KEY",
    "ANTHROPIC_API_KEY",
    "OPENAI_API_KEY",
    "GEMINI_API_KEY",
]


def run_stage5_schema_and_env_audit() -> dict[str, Any]:
    go_env_vars: set[str] = set()
    env_re = re.compile(r'os\.Getenv\(\s*"([^"]+)"\s*\)')
    for gp in GO_ROOT.rglob("*.go"):
        txt = gp.read_text(encoding="utf-8", errors="ignore")
        for m in env_re.findall(txt):
            go_env_vars.add(m)

    missing_envs = [e for e in REQUIRED_ENV_VARS if e not in go_env_vars]

    s_tbl, tbl_out = http_json("GET", "http://127.0.0.1:18000/system/tables")
    tables = tbl_out.get("tables", [])

    s_ddl, ddl_out = http_json("GET", "http://127.0.0.1:18000/v2/system/storage/export-sql?project=stateful-audit&dialect=postgres")
    sql_dump = ddl_out.get("sql_dump", "")
    ddl_ok = s_ddl == 200 and "CREATE TABLE IF NOT EXISTS memories" in sql_dump

    passed = len(missing_envs) == 0 and len(tables) == 12 and ddl_ok
    return {
        "stage": "5. Database Schema, SQL DDL & Environment Variable Contract Audit",
        "passed": passed,
        "canonical_tables_count": len(tables),
        "sql_ddl_export_verified": ddl_ok,
        "go_env_vars_supported": sorted(go_env_vars),
        "missing_env_vars": missing_envs,
    }


def main() -> int:
    go_bin = GO_ROOT / "bin" / "rae"
    if not go_bin.exists():
        subprocess.run(["make", "build"], cwd=str(GO_ROOT), check=True)

    env = os.environ.copy()
    go_proc = subprocess.Popen(
        [str(go_bin), "-port-offset", "10000"],
        cwd=str(GO_ROOT),
        env=env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    time.sleep(0.8)

    try:
        s1 = run_stage1_ast_inventory()
        s2 = run_stage2_static_antistub_scanner()
        s3 = run_stage3_sensitivity_probing()
        s4 = run_stage4_stateful_invariants()
        s5 = run_stage5_schema_and_env_audit()
    finally:
        go_proc.terminate()
        try:
            go_proc.wait(timeout=3)
        except Exception:
            go_proc.kill()

    stages = [s1, s2, s3, s4, s5]
    all_passed = all(st["passed"] for st in stages)

    summary = {
        "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
        "all_5_stages_passed": all_passed,
        "stages_passed_count": f"{sum(1 for st in stages if st['passed'])}/5",
        "stages": stages,
    }

    out_json = GO_ROOT / "docs" / "AUDYT_5_WARSTWOWY_WYNIKI.json"
    out_json.write_text(json.dumps(summary, indent=2, ensure_ascii=False), encoding="utf-8")

    print(f"=== 5-STAGE AUTOMATED PARITY AUDIT: {summary['stages_passed_count']} STAGES PASSED ===")
    for st in stages:
        badge = "PASS" if st["passed"] else "FAIL"
        print(f"  [{badge}] {st['stage']}")
    if not all_passed:
        print(json.dumps(summary, indent=2, ensure_ascii=False))
    return 0 if all_passed else 1


if __name__ == "__main__":
    sys.exit(main())

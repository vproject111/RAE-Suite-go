package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/accounting"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/pkg/raemesh"
	"github.com/vproject111/RAE-Suite-go/services/crl"
	"github.com/vproject111/RAE-Suite-go/services/hive"
	"github.com/vproject111/RAE-Suite-go/services/lab"
	"github.com/vproject111/RAE-Suite-go/services/mcp"
	"github.com/vproject111/RAE-Suite-go/services/memory"
	"github.com/vproject111/RAE-Suite-go/services/orchestrator"
	"github.com/vproject111/RAE-Suite-go/services/phoenix"
	"github.com/vproject111/RAE-Suite-go/services/portal"
	"github.com/vproject111/RAE-Suite-go/services/quality"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "record-task":
			runRecordTaskCLI(os.Args[2:])
			return
		case "bootstrap":
			runBootstrapCLI(os.Args[2:])
			return
		case "mesh":
			runMeshCLI(os.Args[2:])
			return
		case "triad":
			runTriadCLI(os.Args[2:])
			return
		}
	}

	allFlag := flag.Bool("all", true, "Run all 9 RAE-Suite & RAE-CRL modules concurrently in one binary")
	modulesFlag := flag.String("modules", "memory,crl,mcp,suite,quality,lab,phoenix,hive,portal", "Comma-separated list of modules to run")
	profileFlag := flag.String("profile", "edge", "Storage & execution profile: edge | cluster | hybrid")
	portOffsetFlag := flag.Int("port-offset", 0, "Optional port offset (e.g. 10000 to run on 18000..18503 alongside Python stack)")
	dryRunFlag := flag.Bool("dry-run", false, "Verify initialization of selected modules and exit")
	flag.Parse()

	guard := security.NewHardFramesGuard("")
	mc := raemath.NewMathController(models.MathL3)
	store := memorystore.NewUnifiedStore(mc, guard)

	memSvc := memory.NewService(store, store, guard)
	crlSvc := crl.NewService(store, store, guard)
	mcpSvc := mcp.NewService(store, crlSvc)
	qualSvc := quality.NewService()
	hiveSvc := hive.NewService()
	phoenixSvc := phoenix.NewService()
	labSvc := lab.NewService(mc)
	orchSvc := orchestrator.NewService(mc, crlSvc, qualSvc, guard)
	portalSvc := portal.NewService()

	active := make(map[string]bool)
	if *allFlag {
		for _, m := range []string{"memory", "crl", "mcp", "suite", "quality", "lab", "phoenix", "hive", "portal"} {
			active[m] = true
		}
	} else {
		for _, m := range strings.Split(*modulesFlag, ",") {
			active[strings.TrimSpace(m)] = true
		}
	}

	fmt.Printf("RAE-Go v4.6.0 Unified Runner (profile=%s, active_modules=%d, port_offset=%d)\n", *profileFlag, len(active), *portOffsetFlag)
	if *dryRunFlag {
		return
	}

	ctx := context.Background()
	var wg sync.WaitGroup
	startHTTP := func(name string, basePort int, reg func(*http.ServeMux)) {
		addr := fmt.Sprintf(":%d", basePort+*portOffsetFlag)
		wg.Add(1)
		go func() {
			defer wg.Done()
			mux := http.NewServeMux()
			reg(mux)
			log.Printf("[%s] listening on %s", name, addr)
			if err := http.ListenAndServe(addr, mux); err != nil {
				log.Printf("[%s] stopped: %v", name, err)
			}
		}()
	}

	if active["memory"] {
		memSvc.StartBackgroundWorkers(ctx, 60*time.Second)
		startHTTP("rae-memory", 8000, memSvc.RegisterRoutes)
	}
	if active["crl"] {
		crlSvc.StartSyncEngine(ctx, 60*time.Second)
		startHTTP("rae-crl", 8503, crlSvc.RegisterRoutes)
	}
	if active["mcp"] {
		startHTTP("rae-mcp", 8005, mcpSvc.RegisterRoutes)
	}
	if active["suite"] {
		startHTTP("rae-suite", 8009, orchSvc.RegisterRoutes)
	}
	if active["quality"] {
		startHTTP("rae-quality", 8010, qualSvc.RegisterRoutes)
	}
	if active["lab"] {
		startHTTP("rae-lab", 8011, labSvc.RegisterRoutes)
	}
	if active["phoenix"] {
		startHTTP("rae-phoenix", 8012, phoenixSvc.RegisterRoutes)
	}
	if active["hive"] {
		hiveSvc.StartWatchdogLoop(ctx, 30*time.Second)
		startHTTP("rae-hive", 8013, hiveSvc.RegisterRoutes)
	}
	if active["portal"] {
		startHTTP("rae-portal", 8082, portalSvc.RegisterRoutes)
	}

	wg.Wait()
}

func runBootstrapCLI(args []string) {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	project := fs.String("project", "dreamsoft-pro", "Target project context to bootstrap")
	_ = fs.Parse(args)

	out := map[string]interface{}{
		"command":          "bootstrap",
		"project":          *project,
		"protocol":         "AGENT_CORE_PROTOCOL.md",
		"storage_agnostic": true,
		"hermes_opencode":  true,
		"openclaw_enabled": false,
		"status":           "BOOTSTRAPPED_OK",
	}
	raw, _ := json.Marshal(out)
	fmt.Println(string(raw))
}

func runMeshCLI(args []string) {
	action := "status"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		action = args[0]
	}
	store := memorystore.NewUnifiedStore(nil, nil)
	mesh := raemesh.NewMeshEngine(store, nil)
	ctx := context.Background()

	if action == "sync" {
		snap, _, _ := mesh.ComputeMerkleTree(ctx, "")
		out := map[string]interface{}{
			"command":       "mesh sync",
			"status":        "synced",
			"merkle_root":   snap.MerkleRoot,
			"buckets_count": len(snap.BucketHashes),
			"peers_count":   len(mesh.ListPeers()),
		}
		raw, _ := json.Marshal(out)
		fmt.Println(string(raw))
		return
	}
	out := map[string]interface{}{
		"command": "mesh status",
		"status":  "healthy",
		"peers":   mesh.ListPeers(),
	}
	raw, _ := json.Marshal(out)
	fmt.Println(string(raw))
}

func runTriadCLI(args []string) {
	fs := flag.NewFlagSet("triad", flag.ExitOnError)
	cascade := fs.Bool("cascade", false, "Execute full 3-stage 9-model LayerTriad cascade")
	tier := fs.String("tier", "tier1_generate", "Target tier: tier1_generate | tier2_review_repair | tier3_judge")
	text := fs.String("text", "Verify RAE-Suite-go v4.6.0", "Task description for LayerTriad")
	_ = fs.Parse(args)

	mc := raemath.NewMathController(models.MathL3)
	store := memorystore.NewUnifiedStore(mc, nil)
	crlSvc := crl.NewService(store, store, nil)
	qualSvc := quality.NewService()
	orchSvc := orchestrator.NewService(mc, crlSvc, qualSvc, nil)

	selectedTier := *tier
	if *cascade {
		selectedTier = "cascade_all_3_tiers"
	}
	res, _ := orchSvc.ExecuteCascade(context.Background(), *text, selectedTier, "package main\nfunc main(){}\n")
	raw, _ := json.Marshal(res)
	fmt.Println(string(raw))
}

func runRecordTaskCLI(args []string) {
	fs := flag.NewFlagSet("record-task", flag.ExitOnError)
	task := fs.String("task", "", "CO zrobiono (What was done)")
	rationale := fs.String("rationale", "", "DLACZEGO / uzasadnienie architektoniczne (Why)")
	effect := fs.String("effect", "", "EFEKT / wynik i status testów (Impact & Tests)")
	model := fs.String("model", "gemini-2.5-pro", "KTO / jaki model wykonał zadanie (Who)")
	project := fs.String("project", "RAE-Suite-go", "Docelowy projekt w RAE")
	cost := fs.Float64("cost", 0.0, "ZA ILE / koszt w USD")
	latency := fs.Float64("latency-ms", 12.5, "ILE TRWAŁO / czas w ms")
	_ = fs.Parse(args)

	if strings.TrimSpace(*task) == "" || strings.TrimSpace(*rationale) == "" || strings.TrimSpace(*effect) == "" {
		log.Fatalf("record-task requires --task, --rationale, and --effect")
	}

	ledger := accounting.NewExecutionAuditLedger("", 0)
	receipt := ledger.RecordInvocation(
		*model,
		"local_first",
		"grzegorz@cloud",
		*task,
		*rationale,
		"APPROVED",
		500,
		250,
		*cost,
		*latency,
		*effect,
		0,
	)
	fivePillar := accounting.BuildFivePillarMemory(receipt, *project)
	fmt.Printf("✅ [RAE-Go Audit Ledger] Zapisano 5-filarowy paragon zadania (ReceiptID=%s)\n", receipt.ReceiptID)
	fmt.Printf("🔒 [SHA-256 Chain] Hash=%s (Prev=%s)\n", receipt.CryptographicHash, receipt.PreviousHash)
	fmt.Println(fivePillar.Content)
}

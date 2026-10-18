package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
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

	fmt.Printf("RAE-Go v4.1 Unified Runner (profile=%s, active_modules=%d, port_offset=%d)\n", *profileFlag, len(active), *portOffsetFlag)
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

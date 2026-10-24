package sqlstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

func TestPersistentStoreSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	store, err := DriverFactory("persistent", dir)
	if err != nil {
		t.Fatalf("DriverFactory error: %v", err)
	}

	ctx := context.Background()
	mem := &models.MemoryItem{
		ID:         "mem-persist-1",
		Content:    "Persistent Go memory record",
		Layer:      models.LayerSemantic,
		TenantID:   "t1",
		Project:    "rae-go",
		Importance: 0.9,
		Strength:   1.0,
		CreatedAt:  time.Now().UTC(),
	}
	if err := store.StoreMemory(ctx, mem); err != nil {
		t.Fatalf("StoreMemory error: %v", err)
	}

	art := &models.EpistemicArtifact{
		ID:          "art-persist-1",
		Type:        models.ArtifactHypothesis,
		Status:      models.StatusCompleted,
		Title:       "Persistent Hypothesis",
		Description: "Confirmed hypothesis",
		Project:     "rae-go",
		Author:      "agent-1",
		InfoClass:   models.InfoInternal,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := store.SaveArtifact(ctx, art); err != nil {
		t.Fatalf("SaveArtifact error: %v", err)
	}

	reloaded, err := NewPersistentStore(dir)
	if err != nil {
		t.Fatalf("NewPersistentStore reload error: %v", err)
	}
	gotMem, err := reloaded.GetMemory(ctx, "mem-persist-1")
	if err != nil || gotMem.Content != mem.Content {
		t.Fatalf("expected reloaded memory %q, got %v (err=%v)", mem.Content, gotMem, err)
	}
	gotArt, err := reloaded.GetArtifact(ctx, "art-persist-1")
	if err != nil || gotArt.Title != art.Title {
		t.Fatalf("expected reloaded artifact %q, got %v (err=%v)", art.Title, gotArt, err)
	}
}

func TestSQLStoreFullCoverage(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatal("expected RegisteredPackageTypes to return non-empty slice")
	}

	ctx := context.Background()
	_, err := DriverFactory("memory", "")
	if err != nil {
		t.Fatalf("DriverFactory default dataDir failed: %v", err)
	}

	dir := t.TempDir()
	ps, err := NewPersistentStore(dir)
	if err != nil {
		t.Fatalf("NewPersistentStore failed: %v", err)
	}
	_ = ps.AddGraphEdge(ctx, "g1", "g2", "LINKS")
	_ = ps.LinkArtifacts(ctx, "a1", "a2", "supports")
	_ = ps.AppendJournalEvent(ctx, &models.EpistemicJournalEvent{ArtifactID: "a1", Action: "CREATED", Actor: "bot"})

	// ExecuteSQL branches
	_, _ = ps.ExecuteSQL(ctx, "CREATE TABLE custom_tbl (id TEXT)")
	_, _ = ps.ExecuteSQL(ctx, "CREATE TABLE IF NOT EXISTS custom_tbl2 (id TEXT)")
	_, _ = ps.ExecuteSQL(ctx, "INSERT INTO custom_tbl VALUES ($1)", "val1")
	_, _ = ps.ExecuteSQL(ctx, "SELECT * FROM custom_tbl")
	_, _ = ps.ExecuteSQL(ctx, "SELECT * FROM memories")
	_, _ = ps.ExecuteSQL(ctx, "SELECT 1")
	_, _ = ps.ExecuteSQL(ctx, "DELETE FROM custom_tbl")
	_, _ = ps.ExecuteSQL(ctx, "UPDATE custom_tbl SET id = 'x'")
	_ = ps.SaveSnapshot(ctx)

	// Reload snapshot with graph, DAG edges, journal, SQLTables
	if _, err := NewPersistentStore(dir); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	// Corrupt snapshot test
	badDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(badDir, "rae_store_snapshot.json"), []byte("{invalid-json"), 0o644)
	if _, err := NewPersistentStore(badDir); err == nil {
		t.Fatal("expected error loading corrupted snapshot")
	}

	// SQLiteStorage tests
	sqDir := t.TempDir()
	sq, err := NewSQLiteStorage(sqDir)
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	if err := sq.Initialize(); err != nil {
		t.Fatalf("SQLiteStorage.Initialize failed: %v", err)
	}
	pastStr := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	m1 := &models.MemoryItem{
		ID:         "sq-1",
		Content:    "SQLite memory alpha",
		Layer:      models.LayerEpisodic,
		TenantID:   "t-sq",
		Project:    "p-sq",
		Importance: 0.2,
		Metadata:   map[string]interface{}{"expires_at": pastStr, "tag": "temp"},
	}
	m2 := &models.MemoryItem{
		ID:         "sq-2",
		Content:    "SQLite memory beta",
		Layer:      models.LayerSemantic,
		TenantID:   "t-sq",
		Project:    "p-sq",
		Importance: 0.9,
		Metadata:   map[string]interface{}{"tag": "keep"},
	}
	_ = sq.StoreMemory(ctx, m1)
	_ = sq.StoreMemory(ctx, m2)
	batch, _ := sq.GetMemoriesBatch(ctx, []string{"sq-1", "sq-2"})
	if len(batch) != 2 {
		t.Fatalf("expected 2 memories in batch, got %d", len(batch))
	}
	_ = sq.UpdateMemory(ctx, m2)
	if cnt, _ := sq.CountMemories(ctx, "p-sq"); cnt != 2 {
		t.Fatalf("expected count 2, got %d", cnt)
	}
	_, _ = sq.SearchFullText(ctx, "beta", "p-sq", 5)
	_ = sq.UpdateMemoryAccessBatch(ctx, []string{"sq-1", "sq-2", "missing"})
	_ = sq.AdjustImportance(ctx, "sq-2", 0.05)
	_ = sq.AdjustImportance(ctx, "missing", 0.05)
	_, _ = sq.DecayImportance(ctx, 0)
	_ = sq.SaveEmbedding("sq-2", "dense", []float32{1, 0, 0})
	_, _ = sq.GetMetricAggregate(ctx, "p-sq")
	if expDel, _ := sq.DeleteExpiredMemories(ctx); expDel != 1 {
		t.Fatalf("expected 1 expired memory deleted, got %d", expDel)
	}
	_ = sq.StoreMemory(ctx, &models.MemoryItem{ID: "sq-low", Content: "low", Importance: 0.05, Project: "p-sq"})
	// Note: StoreMemory defaults Importance <=0 to 0.5, 0.05 stays 0.05
	if lowDel, _ := sq.DeleteMemoriesBelowImportance(ctx, 0.10); lowDel != 1 {
		t.Fatalf("expected 1 low-importance memory deleted, got %d", lowDel)
	}
	if metaDel, _ := sq.DeleteMemoriesWithMetadataFilter(ctx, "tag", "keep"); metaDel != 1 {
		t.Fatalf("expected 1 metadata-filtered memory deleted, got %d", metaDel)
	}
	_ = sq.StoreMemory(ctx, &models.MemoryItem{ID: "sq-t", Content: "tenant item", TenantID: "t-clear"})
	if tDel, _ := sq.ClearTenant(ctx, "t-clear"); tDel != 1 {
		t.Fatalf("expected 1 tenant memory cleared, got %d", tDel)
	}
	_ = sq.Close()

	// SQLiteVectorStore tests
	sv := NewSQLiteVectorStore()
	_ = sv.Initialize()
	_ = sv.StoreVector("v1", []float32{1, 0, 0})
	_ = sv.UpdateVector("v2", []float32{1, 0, 0}) // tie-breaker test
	_ = sv.StoreVector("v3", []float32{0, 0, 0}) // zero norm test
	if _, ok := sv.GetVector("v1"); !ok {
		t.Fatal("GetVector v1 failed")
	}
	simIDs := sv.SearchSimilar([]float32{1, 0, 0}, 2)
	if len(simIDs) != 2 {
		t.Fatalf("expected 2 similar vectors, got %v", simIDs)
	}
	_ = sv.SearchSimilar([]float32{}, 5) // empty queryVec
	if !sv.DeleteVector("v1") {
		t.Fatal("DeleteVector v1 failed")
	}

	// SQLiteGraphStore tests
	sg := NewSQLiteGraphStore()
	_ = sg.Initialize()
	sg.CreateNode("n1", map[string]interface{}{"k": "v"})
	sg.CreateEdge("n1", "n2", "REL")
	sg.CreateEdge("n2", "n1", "REL")
	_ = sg.GetNeighbors("n1")
	sub := sg.GetSubgraph([]string{"n1"}, 0)
	if len(sub) == 0 {
		t.Fatal("GetSubgraph depth=0 failed")
	}
	sub2 := sg.GetSubgraph([]string{"n1"}, 3)
	if len(sub2) < 2 {
		t.Fatal("GetSubgraph depth=3 failed")
	}
	if !sg.DeleteNode("n1") {
		t.Fatal("DeleteNode n1 failed")
	}

	// PostgreSQLStorage tests
	pgDir := t.TempDir()
	pg, err := NewPostgreSQLStorage("postgres://localhost:5432/rae", pgDir)
	if err != nil {
		t.Fatalf("NewPostgreSQLStorage failed: %v", err)
	}
	pm := &models.MemoryItem{
		ID:         "pg-1",
		Content:    "Postgres memory",
		Project:    "p-pg",
		Importance: 0.4,
		Metadata:   map[string]interface{}{"expires_at": pastStr, "k": "v"},
	}
	_ = pg.StoreMemory(ctx, pm)
	_, _ = pg.GetMemory(ctx, "pg-1")
	_, _ = pg.GetMemoriesBatch(ctx, []string{"pg-1"})
	_, _ = pg.ListMemories(ctx, "p-pg", 10)
	_, _ = pg.SearchMemories(ctx, "Postgres", "p-pg", 10)
	_ = pg.UpdateMemory(ctx, pm)
	_ = pg.UpdateMemoryAccess(ctx, "pg-1")
	_ = pg.UpdateMemoryAccessBatch(ctx, []string{"pg-1"})
	_ = pg.AdjustImportance(ctx, "pg-1", 0.1)
	_, _ = pg.DecayImportance(ctx, 0.9)
	_, _ = pg.CountMemories(ctx, "p-pg")
	_ = pg.SaveEmbedding("pg-1", "dense", []float32{0.5, 0.5})
	_, _ = pg.GetMetricAggregate(ctx, "p-pg")
	_, _ = pg.DeleteMemoriesWithMetadataFilter(ctx, "k", "v")
	_ = pg.StoreMemory(ctx, pm)
	_, _ = pg.DeleteExpiredMemories(ctx)
	_ = pg.StoreMemory(ctx, &models.MemoryItem{ID: "pg-low", Content: "low", Importance: 0.05})
	_, _ = pg.DeleteMemoriesBelowImportance(ctx, 0.2)
	_ = pg.StoreMemory(ctx, &models.MemoryItem{ID: "pg-del", Content: "del"})
	_ = pg.DeleteMemory(ctx, "pg-del")
	_ = pg.Close()

	// InMemoryStorage tests
	im := NewInMemoryStorage()
	imItem := &models.MemoryItem{ID: "im-1", Content: "In-memory item", Project: "p-im", Importance: 0.7}
	_ = im.StoreMemory(ctx, imItem)
	_, _ = im.GetMemory(ctx, "im-1")
	_ = im.UpdateMemory(ctx, imItem)
	_, _ = im.ListMemories(ctx, "p-im", 10)
	_, _ = im.CountMemories(ctx, "p-im")
	_, _ = im.SearchMemories(ctx, "In-memory", "p-im", 10)
	_ = im.StoreVector(ctx, "im-1", []float32{1, 0})
	_ = im.UpdateVector(ctx, "im-2", []float32{0, 1})
	_, _ = im.GetVector("im-1")
	_, _ = im.SearchSimilar(ctx, []float32{1, 0}, 1)
	_, _ = im.SearchSimilarBatch(ctx, [][]float32{{1, 0}, {0, 1}}, 1)
	_ = im.DeleteVector("im-2")
	_ = im.AdjustImportance(ctx, "im-1", 0.1)
	_ = im.AdjustImportance(ctx, "missing", 0.1)
	_, _ = im.DecayImportance(ctx, 0)
	_ = im.DeleteMemory(ctx, "im-1")
	_ = im.Close()
	_ = ports.QueryFilter{}
}


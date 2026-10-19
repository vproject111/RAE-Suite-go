package graph

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// ChangeType enumerates temporal knowledge graph mutations.
type ChangeType string

const (
	ChangeNodeAdded      ChangeType = "NODE_ADDED"
	ChangeNodeRemoved    ChangeType = "NODE_REMOVED"
	ChangeEdgeAdded      ChangeType = "EDGE_ADDED"
	ChangeEdgeModified   ChangeType = "EDGE_MODIFIED"
	ChangeEdgeRemoved    ChangeType = "EDGE_REMOVED"
	ChangeWeightDecayed  ChangeType = "WEIGHT_DECAYED"
)

// GraphChange represents an immutable event in the temporal knowledge graph log.
type GraphChange struct {
	ID         string     `json:"id"`
	Project    string     `json:"project"`
	ChangeType ChangeType `json:"change_type"`
	SourceNode string     `json:"source_node"`
	TargetNode string     `json:"target_node,omitempty"`
	Relation   string     `json:"relation,omitempty"`
	Weight     float64    `json:"weight"`
	Timestamp  time.Time  `json:"timestamp"`
}

// GraphSnapshot captures the complete state of the knowledge graph at a point in time.
type GraphSnapshot struct {
	SnapshotID string              `json:"snapshot_id"`
	Project    string              `json:"project"`
	Label      string              `json:"label"`
	CreatedAt  time.Time           `json:"created_at"`
	Nodes      []string            `json:"nodes"`
	Adjacency  map[string][]string `json:"adjacency"`
	EdgeCount  int                 `json:"edge_count"`
	Density    float64             `json:"density"`
}

// TemporalComparison summarizes differences between two graph states at times t1 and t2.
type TemporalComparison struct {
	T1           time.Time `json:"t1"`
	T2           time.Time `json:"t2"`
	AddedNodes   []string  `json:"added_nodes"`
	RemovedNodes []string  `json:"removed_nodes"`
	AddedEdges   int       `json:"added_edges"`
	DensityT1    float64   `json:"density_t1"`
	DensityT2    float64   `json:"density_t2"`
	DensityDelta float64   `json:"density_delta"`
}

// EmergingPattern identifies rapidly growing concepts/clusters in the temporal graph.
type EmergingPattern struct {
	Node         string  `json:"node"`
	EdgeGrowth   int     `json:"edge_growth"`
	GrowthVelocity float64 `json:"growth_velocity"`
}

// TemporalGraphService implements event-sourced temporal graph reconstruction, snapshots, diffing, and emerging pattern detection.
type TemporalGraphService struct {
	mu        sync.RWMutex
	changes   []GraphChange
	snapshots map[string]GraphSnapshot
}

// NewTemporalGraphService creates a new TemporalGraphService.
func NewTemporalGraphService() *TemporalGraphService {
	return &TemporalGraphService{
		snapshots: make(map[string]GraphSnapshot),
	}
}

// RecordChange appends a GraphChange event to the temporal log.
func (s *TemporalGraphService) RecordChange(ch GraphChange) GraphChange {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch.ID == "" {
		ch.ID = fmt.Sprintf("gch-%d", len(s.changes)+1)
	}
	if ch.Timestamp.IsZero() {
		ch.Timestamp = time.Now().UTC()
	}
	if ch.Weight <= 0 {
		ch.Weight = 1.0
	}
	s.changes = append(s.changes, ch)
	return ch
}

// CalculateDensity computes directed graph density |E| / (|V| * (|V| - 1)).
func CalculateDensity(nodeCount, edgeCount int) float64 {
	if nodeCount <= 1 {
		return 0.0
	}
	maxEdges := float64(nodeCount * (nodeCount - 1))
	return math.Round((float64(edgeCount)/maxEdges)*10000) / 10000
}

// ReconstructGraphAtTime replays all GraphChange events up to targetTime (or all if targetTime.IsZero()).
func (s *TemporalGraphService) ReconstructGraphAtTime(project string, targetTime time.Time) GraphSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodeSet := make(map[string]bool)
	adjSet := make(map[string]map[string]bool)

	for _, ch := range s.changes {
		if project != "" && ch.Project != "" && ch.Project != project {
			continue
		}
		if !targetTime.IsZero() && ch.Timestamp.After(targetTime) {
			continue
		}
		switch ch.ChangeType {
		case ChangeNodeAdded:
			nodeSet[ch.SourceNode] = true
		case ChangeNodeRemoved:
			delete(nodeSet, ch.SourceNode)
			delete(adjSet, ch.SourceNode)
			for u := range adjSet {
				delete(adjSet[u], ch.SourceNode)
			}
		case ChangeEdgeAdded, ChangeEdgeModified:
			nodeSet[ch.SourceNode] = true
			if ch.TargetNode != "" {
				nodeSet[ch.TargetNode] = true
				if adjSet[ch.SourceNode] == nil {
					adjSet[ch.SourceNode] = make(map[string]bool)
				}
				adjSet[ch.SourceNode][ch.TargetNode] = true
			}
		case ChangeEdgeRemoved:
			if adjSet[ch.SourceNode] != nil {
				delete(adjSet[ch.SourceNode], ch.TargetNode)
			}
		}
	}

	nodes := make([]string, 0, len(nodeSet))
	for n := range nodeSet {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)

	adj := make(map[string][]string, len(adjSet))
	edgeCount := 0
	for u, targets := range adjSet {
		list := make([]string, 0, len(targets))
		for v := range targets {
			list = append(list, v)
			edgeCount++
		}
		sort.Strings(list)
		if len(list) > 0 {
			adj[u] = list
		}
	}

	ts := targetTime
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return GraphSnapshot{
		SnapshotID: fmt.Sprintf("snap-%d", ts.UnixNano()),
		Project:    project,
		Label:      "reconstructed",
		CreatedAt:  ts,
		Nodes:      nodes,
		Adjacency:  adj,
		EdgeCount:  edgeCount,
		Density:    CalculateDensity(len(nodes), edgeCount),
	}
}

// CreateSnapshot saves a named point-in-time snapshot of the current graph.
func (s *TemporalGraphService) CreateSnapshot(project, label string) GraphSnapshot {
	snap := s.ReconstructGraphAtTime(project, time.Now().UTC())
	s.mu.Lock()
	defer s.mu.Unlock()
	snap.SnapshotID = fmt.Sprintf("snap-%d", len(s.snapshots)+1)
	if label != "" {
		snap.Label = label
	}
	s.snapshots[snap.SnapshotID] = snap
	return snap
}

// RestoreSnapshot restores the temporal graph state to a previously saved snapshot.
func (s *TemporalGraphService) RestoreSnapshot(snapshotID string) (GraphSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, ok := s.snapshots[snapshotID]
	if !ok {
		return GraphSnapshot{}, false
	}
	// Reset changes for this project to match the snapshot
	var filtered []GraphChange
	for _, ch := range s.changes {
		if ch.Project != snap.Project {
			filtered = append(filtered, ch)
		}
	}
	now := time.Now().UTC()
	for _, n := range snap.Nodes {
		filtered = append(filtered, GraphChange{
			ID:         fmt.Sprintf("gch-rest-%s", n),
			Project:    snap.Project,
			ChangeType: ChangeNodeAdded,
			SourceNode: n,
			Timestamp:  now,
		})
	}
	for u, targets := range snap.Adjacency {
		for _, v := range targets {
			filtered = append(filtered, GraphChange{
				ID:         fmt.Sprintf("gch-rest-%s-%s", u, v),
				Project:    snap.Project,
				ChangeType: ChangeEdgeAdded,
				SourceNode: u,
				TargetNode: v,
				Relation:   "RESTORED",
				Weight:     1.0,
				Timestamp:  now,
			})
		}
	}
	s.changes = filtered
	return snap, true
}

// CompareGraphs computes structural diffs between timestamps t1 and t2.
func (s *TemporalGraphService) CompareGraphs(project string, t1, t2 time.Time) TemporalComparison {
	g1 := s.ReconstructGraphAtTime(project, t1)
	g2 := s.ReconstructGraphAtTime(project, t2)

	set1 := make(map[string]bool, len(g1.Nodes))
	for _, n := range g1.Nodes {
		set1[n] = true
	}
	set2 := make(map[string]bool, len(g2.Nodes))
	for _, n := range g2.Nodes {
		set2[n] = true
	}

	var added, removed []string
	for _, n := range g2.Nodes {
		if !set1[n] {
			added = append(added, n)
		}
	}
	for _, n := range g1.Nodes {
		if !set2[n] {
			removed = append(removed, n)
		}
	}

	return TemporalComparison{
		T1:           t1,
		T2:           t2,
		AddedNodes:   added,
		RemovedNodes: removed,
		AddedEdges:   g2.EdgeCount - g1.EdgeCount,
		DensityT1:    g1.Density,
		DensityT2:    g2.Density,
		DensityDelta: math.Round((g2.Density-g1.Density)*10000) / 10000,
	}
}

// FindEmergingPatterns identifies nodes with highest edge growth velocity in the temporal log.
func (s *TemporalGraphService) FindEmergingPatterns(project string) []EmergingPattern {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[string]int)
	for _, ch := range s.changes {
		if project != "" && ch.Project != "" && ch.Project != project {
			continue
		}
		if ch.ChangeType == ChangeEdgeAdded || ch.ChangeType == ChangeNodeAdded {
			counts[ch.SourceNode]++
			if ch.TargetNode != "" {
				counts[ch.TargetNode]++
			}
		}
	}

	patterns := make([]EmergingPattern, 0, len(counts))
	for node, cnt := range counts {
		patterns = append(patterns, EmergingPattern{
			Node:           node,
			EdgeGrowth:     cnt,
			GrowthVelocity: math.Round(float64(cnt)*0.5*100) / 100,
		})
	}
	sort.Slice(patterns, func(i, j int) bool {
		if patterns[i].EdgeGrowth == patterns[j].EdgeGrowth {
			return patterns[i].Node < patterns[j].Node
		}
		return patterns[i].EdgeGrowth > patterns[j].EdgeGrowth
	})
	return patterns
}

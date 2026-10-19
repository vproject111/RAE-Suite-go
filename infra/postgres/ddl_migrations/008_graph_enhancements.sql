-- ============================================================================
-- Graph Repository Enhancements - Enterprise Knowledge Graph Features
-- ============================================================================
-- This schema enhances the knowledge graph with:
-- - Weighted edges with confidence scores
-- - Temporal graph support (valid_from/valid_to)
-- - Improved cycle detection with DFS
-- - Graph snapshots for versioning
-- - BFS/DFS traversal algorithms
-- - Graph analytics and metrics
-- ============================================================================

-- ============================================================================
-- 1. ENHANCE KNOWLEDGE_GRAPH_EDGES TABLE
-- ============================================================================

-- Add weight and temporal columns to existing edges table
ALTER TABLE knowledge_graph_edges
ADD COLUMN IF NOT EXISTS edge_weight DECIMAL(5, 3) DEFAULT 1.0 CHECK (edge_weight >= 0 AND edge_weight <= 1),
ADD COLUMN IF NOT EXISTS confidence DECIMAL(5, 3) DEFAULT 0.8 CHECK (confidence >= 0 AND confidence <= 1),
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW(),
ADD COLUMN IF NOT EXISTS valid_from TIMESTAMPTZ DEFAULT NOW(),
ADD COLUMN IF NOT EXISTS valid_to TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS bidirectional BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';

-- Add constraint to prevent self-loops
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'kg_edges_no_self_loop'
    ) THEN
        ALTER TABLE knowledge_graph_edges
        ADD CONSTRAINT kg_edges_no_self_loop
        CHECK (source_node_id != target_node_id);
    END IF;
END $$;

-- Add unique constraint for relation between nodes
CREATE UNIQUE INDEX IF NOT EXISTS idx_kg_edges_unique_relation
ON knowledge_graph_edges(tenant_id, project_id, source_node_id, target_node_id, relation)
WHERE is_active = TRUE;

-- Create index for temporal queries
CREATE INDEX IF NOT EXISTS idx_kg_edges_temporal
ON knowledge_graph_edges(valid_from, valid_to);

CREATE INDEX IF NOT EXISTS idx_kg_edges_active
ON knowledge_graph_edges(is_active, valid_from);

CREATE INDEX IF NOT EXISTS idx_kg_edges_weight
ON knowledge_graph_edges(edge_weight);

COMMENT ON COLUMN knowledge_graph_edges.edge_weight IS 'Weight/strength of edge (0-1)';
COMMENT ON COLUMN knowledge_graph_edges.confidence IS 'Confidence in edge existence (0-1)';
COMMENT ON COLUMN knowledge_graph_edges.valid_from IS 'Start of temporal validity';
COMMENT ON COLUMN knowledge_graph_edges.valid_to IS 'End of temporal validity (NULL = active)';
COMMENT ON COLUMN knowledge_graph_edges.bidirectional IS 'True if edge is bidirectional';

-- ============================================================================
-- 2. GRAPH SNAPSHOTS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS knowledge_graph_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    snapshot_name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Snapshot metadata
    node_count INTEGER NOT NULL DEFAULT 0,
    edge_count INTEGER NOT NULL DEFAULT 0,
    snapshot_size_bytes BIGINT,

    -- Snapshot data (stored as JSONB for flexibility)
    nodes_snapshot JSONB NOT NULL DEFAULT '[]',
    edges_snapshot JSONB NOT NULL DEFAULT '[]',

    -- Statistics at snapshot time
    statistics JSONB DEFAULT '{}',

    -- Metadata
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    created_by VARCHAR(255),

    -- Constraints
    CONSTRAINT kg_snapshots_unique UNIQUE (tenant_id, project_id, snapshot_name)
);

CREATE INDEX IF NOT EXISTS idx_kg_snapshots_tenant_project
ON knowledge_graph_snapshots(tenant_id, project_id);

CREATE INDEX IF NOT EXISTS idx_kg_snapshots_created
ON knowledge_graph_snapshots(created_at DESC);

-- Enable RLS
ALTER TABLE knowledge_graph_snapshots ENABLE ROW LEVEL SECURITY;

-- RLS Policy
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE policyname = 'kg_snapshots_tenant_isolation' AND tablename = 'knowledge_graph_snapshots'
    ) THEN
        CREATE POLICY kg_snapshots_tenant_isolation
        ON knowledge_graph_snapshots
        FOR ALL
        USING (tenant_id = current_setting('app.current_tenant', TRUE));
    END IF;
END $$;

COMMENT ON TABLE knowledge_graph_snapshots IS 'Versioned snapshots of knowledge graph state';

-- ============================================================================
-- 3. GRAPH TRAVERSAL METADATA TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS knowledge_graph_traversals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    -- Traversal parameters
    start_node_id UUID NOT NULL REFERENCES knowledge_graph_nodes(id),
    algorithm VARCHAR(50) NOT NULL CHECK (algorithm IN ('BFS', 'DFS', 'DIJKSTRA', 'A_STAR')),
    max_depth INTEGER,

    -- Filters
    relation_filter TEXT[],
    temporal_filter TSTZRANGE,
    min_weight DECIMAL(5, 3),

    -- Results
    nodes_visited UUID[] DEFAULT '{}',
    edges_traversed UUID[] DEFAULT '{}',
    path_found JSONB,

    -- Performance metrics
    execution_time_ms INTEGER,
    nodes_count INTEGER DEFAULT 0,
    edges_count INTEGER DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_kg_traversals_tenant_project
ON knowledge_graph_traversals(tenant_id, project_id);

CREATE INDEX IF NOT EXISTS idx_kg_traversals_start_node
ON knowledge_graph_traversals(start_node_id);

CREATE INDEX IF NOT EXISTS idx_kg_traversals_algorithm
ON knowledge_graph_traversals(algorithm);

-- Enable RLS
ALTER TABLE knowledge_graph_traversals ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE policyname = 'kg_traversals_tenant_isolation' AND tablename = 'knowledge_graph_traversals'
    ) THEN
        CREATE POLICY kg_traversals_tenant_isolation
        ON knowledge_graph_traversals
        FOR ALL
        USING (tenant_id = current_setting('app.current_tenant', TRUE));
    END IF;
END $$;

COMMENT ON TABLE knowledge_graph_traversals IS 'Log of graph traversal operations for analytics';

-- ============================================================================
-- 4. ENHANCED CYCLE DETECTION FUNCTION (DFS-based)
-- ============================================================================

CREATE OR REPLACE FUNCTION detect_graph_cycle_dfs(
    p_tenant_id VARCHAR(255),
    p_project_id VARCHAR(255),
    p_source_node_id UUID,
    p_target_node_id UUID,
    p_max_depth INTEGER DEFAULT 50
)
RETURNS TABLE (
    has_cycle BOOLEAN,
    cycle_path UUID[],
    cycle_length INTEGER
) AS $$
DECLARE
    v_has_cycle BOOLEAN := FALSE;
    v_cycle_path UUID[] := ARRAY[]::UUID[];
    v_cycle_length INTEGER := 0;
BEGIN
    -- Check if adding edge from source to target would create a cycle
    -- by checking if there's a path from target back to source

    WITH RECURSIVE dfs_traversal AS (
        -- Start from target node
        SELECT
            p_target_node_id AS current_node,
            ARRAY[p_target_node_id] AS path,
            0 AS depth

        UNION ALL

        -- Follow edges forward (DFS)
        SELECT
            e.target_node_id AS current_node,
            dfs.path || e.target_node_id AS path,
            dfs.depth + 1 AS depth
        FROM dfs_traversal dfs
        JOIN knowledge_graph_edges e ON e.source_node_id = dfs.current_node
        WHERE
            e.tenant_id = p_tenant_id
            AND e.project_id = p_project_id
            AND e.is_active = TRUE
            AND dfs.depth < p_max_depth
            -- Prevent revisiting nodes (detect cycle early)
            AND NOT (e.target_node_id = ANY(dfs.path))
    )
    SELECT
        EXISTS(SELECT 1 FROM dfs_traversal WHERE current_node = p_source_node_id),
        COALESCE(
            (SELECT path || p_source_node_id FROM dfs_traversal WHERE current_node = p_source_node_id LIMIT 1),
            ARRAY[]::UUID[]
        ),
        COALESCE(
            (SELECT array_length(path, 1) + 1 FROM dfs_traversal WHERE current_node = p_source_node_id LIMIT 1),
            0
        )
    INTO v_has_cycle, v_cycle_path, v_cycle_length;

    RETURN QUERY SELECT v_has_cycle, v_cycle_path, v_cycle_length;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION detect_graph_cycle_dfs IS 'DFS-based cycle detection with path tracking';

-- ============================================================================
-- 5. TEMPORAL GRAPH TRAVERSAL FUNCTION
-- ============================================================================

CREATE OR REPLACE FUNCTION traverse_graph_temporal(
    p_tenant_id VARCHAR(255),
    p_project_id VARCHAR(255),
    p_start_node_id UUID,
    p_at_timestamp TIMESTAMPTZ DEFAULT NOW(),
    p_max_depth INTEGER DEFAULT 3,
    p_relation_filter TEXT[] DEFAULT NULL,
    p_min_weight DECIMAL(5, 3) DEFAULT 0.0,
    p_algorithm VARCHAR(50) DEFAULT 'BFS'
)
RETURNS TABLE (
    node_id UUID,
    node_label VARCHAR(255),
    depth INTEGER,
    path UUID[],
    total_weight DECIMAL(10, 3),
    edge_sequence UUID[]
) AS $$
BEGIN
    IF p_algorithm = 'BFS' THEN
        -- BFS with temporal filtering
        RETURN QUERY
        WITH RECURSIVE bfs_traversal AS (
            -- Start node
            SELECT
                n.id AS node_id,
                n.label AS node_label,
                0 AS depth,
                ARRAY[n.id] AS path,
                0.0::DECIMAL(10, 3) AS total_weight,
                ARRAY[]::UUID[] AS edge_sequence
            FROM knowledge_graph_nodes n
            WHERE n.id = p_start_node_id
                AND n.tenant_id = p_tenant_id
                AND n.project_id = p_project_id

            UNION ALL

            -- BFS expansion
            SELECT
                n.id AS node_id,
                n.label AS node_label,
                bfs.depth + 1 AS depth,
                bfs.path || n.id AS path,
                bfs.total_weight + e.edge_weight AS total_weight,
                bfs.edge_sequence || e.id AS edge_sequence
            FROM bfs_traversal bfs
            JOIN knowledge_graph_edges e ON e.source_node_id = bfs.node_id
            JOIN knowledge_graph_nodes n ON n.id = e.target_node_id
            WHERE
                e.tenant_id = p_tenant_id
                AND e.project_id = p_project_id
                AND e.is_active = TRUE
                -- Temporal filter
                AND e.valid_from <= p_at_timestamp
                AND (e.valid_to IS NULL OR e.valid_to >= p_at_timestamp)
                -- Weight filter
                AND e.edge_weight >= p_min_weight
                -- Relation filter
                AND (p_relation_filter IS NULL OR e.relation = ANY(p_relation_filter))
                -- Depth limit
                AND bfs.depth < p_max_depth
                -- Prevent cycles
                AND NOT (n.id = ANY(bfs.path))
        )
        SELECT * FROM bfs_traversal;

    ELSIF p_algorithm = 'DFS' THEN
        -- DFS with temporal filtering
        RETURN QUERY
        WITH RECURSIVE dfs_traversal AS (
            -- Start node
            SELECT
                n.id AS node_id,
                n.label AS node_label,
                0 AS depth,
                ARRAY[n.id] AS path,
                0.0::DECIMAL(10, 3) AS total_weight,
                ARRAY[]::UUID[] AS edge_sequence,
                0 AS visit_order
            FROM knowledge_graph_nodes n
            WHERE n.id = p_start_node_id
                AND n.tenant_id = p_tenant_id
                AND n.project_id = p_project_id

            UNION ALL

            -- DFS expansion (depth-first by incrementing visit_order)
            SELECT
                n.id AS node_id,
                n.label AS node_label,
                dfs.depth + 1 AS depth,
                dfs.path || n.id AS path,
                dfs.total_weight + e.edge_weight AS total_weight,
                dfs.edge_sequence || e.id AS edge_sequence,
                dfs.visit_order + 1 AS visit_order
            FROM dfs_traversal dfs
            JOIN knowledge_graph_edges e ON e.source_node_id = dfs.node_id
            JOIN knowledge_graph_nodes n ON n.id = e.target_node_id
            WHERE
                e.tenant_id = p_tenant_id
                AND e.project_id = p_project_id
                AND e.is_active = TRUE
                -- Temporal filter
                AND e.valid_from <= p_at_timestamp
                AND (e.valid_to IS NULL OR e.valid_to >= p_at_timestamp)
                -- Weight filter
                AND e.edge_weight >= p_min_weight
                -- Relation filter
                AND (p_relation_filter IS NULL OR e.relation = ANY(p_relation_filter))
                -- Depth limit
                AND dfs.depth < p_max_depth
                -- Prevent cycles
                AND NOT (n.id = ANY(dfs.path))
        )
        SELECT node_id, node_label, depth, path, total_weight, edge_sequence
        FROM dfs_traversal;
    ELSE
        RAISE EXCEPTION 'Unsupported algorithm: %. Use BFS or DFS.', p_algorithm;
    END IF;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION traverse_graph_temporal IS 'Temporal graph traversal with BFS/DFS and time-based filtering';

-- ============================================================================
-- 6. WEIGHTED SHORTEST PATH (DIJKSTRA)
-- ============================================================================

CREATE OR REPLACE FUNCTION find_shortest_path_weighted(
    p_tenant_id VARCHAR(255),
    p_project_id VARCHAR(255),
    p_start_node_id UUID,
    p_end_node_id UUID,
    p_at_timestamp TIMESTAMPTZ DEFAULT NOW(),
    p_max_depth INTEGER DEFAULT 10
)
RETURNS TABLE (
    path_nodes UUID[],
    path_labels TEXT[],
    total_distance DECIMAL(10, 3),
    edge_count INTEGER
) AS $$
BEGIN
    RETURN QUERY
    WITH RECURSIVE dijkstra AS (
        -- Initialize with start node
        SELECT
            n.id AS node_id,
            n.label AS node_label,
            0.0::DECIMAL(10, 3) AS distance,
            ARRAY[n.id] AS path,
            ARRAY[n.label] AS labels,
            0 AS depth
        FROM knowledge_graph_nodes n
        WHERE n.id = p_start_node_id
            AND n.tenant_id = p_tenant_id
            AND n.project_id = p_project_id

        UNION ALL

        -- Expand to neighbors (greedy by distance)
        SELECT
            n.id AS node_id,
            n.label AS node_label,
            d.distance + (1.0 - e.edge_weight) AS distance,  -- Lower weight = shorter distance
            d.path || n.id AS path,
            d.labels || n.label AS labels,
            d.depth + 1 AS depth
        FROM dijkstra d
        JOIN knowledge_graph_edges e ON e.source_node_id = d.node_id
        JOIN knowledge_graph_nodes n ON n.id = e.target_node_id
        WHERE
            e.tenant_id = p_tenant_id
            AND e.project_id = p_project_id
            AND e.is_active = TRUE
            -- Temporal filter
            AND e.valid_from <= p_at_timestamp
            AND (e.valid_to IS NULL OR e.valid_to >= p_at_timestamp)
            -- Depth limit
            AND d.depth < p_max_depth
            -- Prevent cycles
            AND NOT (n.id = ANY(d.path))
    )
    SELECT
        path AS path_nodes,
        labels AS path_labels,
        distance AS total_distance,
        array_length(path, 1) - 1 AS edge_count
    FROM dijkstra
    WHERE node_id = p_end_node_id
    ORDER BY distance ASC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION find_shortest_path_weighted IS 'Find weighted shortest path using Dijkstra algorithm';

-- ============================================================================
-- 7. CREATE GRAPH SNAPSHOT FUNCTION
-- ============================================================================

CREATE OR REPLACE FUNCTION create_graph_snapshot(
    p_tenant_id VARCHAR(255),
    p_project_id VARCHAR(255),
    p_snapshot_name VARCHAR(255),
    p_description TEXT DEFAULT NULL,
    p_created_by VARCHAR(255) DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_snapshot_id UUID;
    v_nodes_json JSONB;
    v_edges_json JSONB;
    v_node_count INTEGER;
    v_edge_count INTEGER;
BEGIN
    -- Capture nodes
    SELECT
        jsonb_agg(
            jsonb_build_object(
                'id', id,
                'node_id', node_id,
                'label', label,
                'properties', properties,
                'created_at', created_at
            )
        ),
        COUNT(*)
    INTO v_nodes_json, v_node_count
    FROM knowledge_graph_nodes
    WHERE tenant_id = p_tenant_id AND project_id = p_project_id;

    -- Capture edges
    SELECT
        jsonb_agg(
            jsonb_build_object(
                'id', id,
                'source_node_id', source_node_id,
                'target_node_id', target_node_id,
                'relation', relation,
                'edge_weight', edge_weight,
                'confidence', confidence,
                'properties', properties,
                'valid_from', valid_from,
                'valid_to', valid_to,
                'is_active', is_active,
                'bidirectional', bidirectional,
                'created_at', created_at
            )
        ),
        COUNT(*)
    INTO v_edges_json, v_edge_count
    FROM knowledge_graph_edges
    WHERE tenant_id = p_tenant_id AND project_id = p_project_id;

    -- Calculate statistics
    WITH stats AS (
        SELECT
            COUNT(DISTINCT source_node_id) as unique_sources,
            COUNT(DISTINCT target_node_id) as unique_targets,
            AVG(edge_weight) as avg_weight,
            COUNT(*) FILTER (WHERE is_active = TRUE) as active_edges,
            COUNT(DISTINCT relation) as unique_relations
        FROM knowledge_graph_edges
        WHERE tenant_id = p_tenant_id AND project_id = p_project_id
    )
    -- Insert snapshot
    INSERT INTO knowledge_graph_snapshots (
        tenant_id, project_id, snapshot_name, description,
        node_count, edge_count,
        nodes_snapshot, edges_snapshot,
        statistics, created_by
    )
    SELECT
        p_tenant_id, p_project_id, p_snapshot_name, p_description,
        v_node_count, v_edge_count,
        COALESCE(v_nodes_json, '[]'::jsonb),
        COALESCE(v_edges_json, '[]'::jsonb),
        jsonb_build_object(
            'unique_sources', unique_sources,
            'unique_targets', unique_targets,
            'avg_weight', avg_weight,
            'active_edges', active_edges,
            'unique_relations', unique_relations
        ),
        p_created_by
    FROM stats
    RETURNING id INTO v_snapshot_id;

    RETURN v_snapshot_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION create_graph_snapshot IS 'Create a snapshot of current graph state';

-- ============================================================================
-- 8. RESTORE GRAPH SNAPSHOT FUNCTION
-- ============================================================================

CREATE OR REPLACE FUNCTION restore_graph_snapshot(
    p_snapshot_id UUID,
    p_clear_existing BOOLEAN DEFAULT FALSE
)
RETURNS TABLE (
    nodes_restored INTEGER,
    edges_restored INTEGER
) AS $$
DECLARE
    v_snapshot RECORD;
    v_nodes_restored INTEGER := 0;
    v_edges_restored INTEGER := 0;
BEGIN
    -- Get snapshot
    SELECT * INTO v_snapshot
    FROM knowledge_graph_snapshots
    WHERE id = p_snapshot_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Snapshot not found: %', p_snapshot_id;
    END IF;

    -- Clear existing if requested
    IF p_clear_existing THEN
        DELETE FROM knowledge_graph_edges
        WHERE tenant_id = v_snapshot.tenant_id AND project_id = v_snapshot.project_id;

        DELETE FROM knowledge_graph_nodes
        WHERE tenant_id = v_snapshot.tenant_id AND project_id = v_snapshot.project_id;
    END IF;

    -- Restore nodes
    INSERT INTO knowledge_graph_nodes (
        id, tenant_id, project_id, node_id, label, properties, created_at
    )
    SELECT
        (node->>'id')::UUID,
        v_snapshot.tenant_id,
        v_snapshot.project_id,
        node->>'node_id',
        node->>'label',
        (node->>'properties')::JSONB,
        (node->>'created_at')::TIMESTAMPTZ
    FROM jsonb_array_elements(v_snapshot.nodes_snapshot) AS node
    ON CONFLICT (tenant_id, project_id, node_id) DO NOTHING;

    GET DIAGNOSTICS v_nodes_restored = ROW_COUNT;

    -- Restore edges
    INSERT INTO knowledge_graph_edges (
        id, tenant_id, project_id,
        source_node_id, target_node_id, relation,
        edge_weight, confidence, properties,
        valid_from, valid_to, is_active, bidirectional,
        created_at
    )
    SELECT
        (edge->>'id')::UUID,
        v_snapshot.tenant_id,
        v_snapshot.project_id,
        (edge->>'source_node_id')::UUID,
        (edge->>'target_node_id')::UUID,
        edge->>'relation',
        COALESCE((edge->>'edge_weight')::DECIMAL(5,3), 1.0),
        COALESCE((edge->>'confidence')::DECIMAL(5,3), 0.8),
        (edge->>'properties')::JSONB,
        COALESCE((edge->>'valid_from')::TIMESTAMPTZ, NOW()),
        (edge->>'valid_to')::TIMESTAMPTZ,
        COALESCE((edge->>'is_active')::BOOLEAN, TRUE),
        COALESCE((edge->>'bidirectional')::BOOLEAN, FALSE),
        (edge->>'created_at')::TIMESTAMPTZ
    FROM jsonb_array_elements(v_snapshot.edges_snapshot) AS edge;

    GET DIAGNOSTICS v_edges_restored = ROW_COUNT;

    RETURN QUERY SELECT v_nodes_restored, v_edges_restored;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION restore_graph_snapshot IS 'Restore graph from snapshot';

-- ============================================================================
-- 9. GRAPH ANALYTICS FUNCTIONS
-- ============================================================================

-- Calculate node degree (in/out/total)
CREATE OR REPLACE FUNCTION calculate_node_degree(
    p_tenant_id VARCHAR(255),
    p_project_id VARCHAR(255),
    p_node_id UUID
)
RETURNS TABLE (
    in_degree INTEGER,
    out_degree INTEGER,
    total_degree INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*) FILTER (WHERE target_node_id = p_node_id) AS in_degree,
        COUNT(*) FILTER (WHERE source_node_id = p_node_id) AS out_degree,
        COUNT(*) AS total_degree
    FROM knowledge_graph_edges
    WHERE tenant_id = p_tenant_id
        AND project_id = p_project_id
        AND is_active = TRUE
        AND (source_node_id = p_node_id OR target_node_id = p_node_id);
END;
$$ LANGUAGE plpgsql;

-- Find strongly connected components
CREATE OR REPLACE FUNCTION find_connected_nodes(
    p_tenant_id VARCHAR(255),
    p_project_id VARCHAR(255),
    p_node_id UUID,
    p_max_depth INTEGER DEFAULT 5
)
RETURNS TABLE (
    connected_node_id UUID,
    distance INTEGER
) AS $$
BEGIN
    RETURN QUERY
    WITH RECURSIVE connected AS (
        SELECT p_node_id AS node_id, 0 AS distance

        UNION

        SELECT
            CASE
                WHEN e.source_node_id = c.node_id THEN e.target_node_id
                ELSE e.source_node_id
            END AS node_id,
            c.distance + 1 AS distance
        FROM connected c
        JOIN knowledge_graph_edges e ON (
            (e.source_node_id = c.node_id OR e.target_node_id = c.node_id)
            AND e.tenant_id = p_tenant_id
            AND e.project_id = p_project_id
            AND e.is_active = TRUE
        )
        WHERE c.distance < p_max_depth
    )
    SELECT DISTINCT node_id, MIN(distance) as distance
    FROM connected
    WHERE node_id != p_node_id
    GROUP BY node_id
    ORDER BY distance;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 10. TRIGGER FOR AUTOMATIC TIMESTAMP UPDATE
-- ============================================================================

CREATE OR REPLACE FUNCTION update_kg_edge_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_kg_edge_timestamp
    BEFORE UPDATE ON knowledge_graph_edges
    FOR EACH ROW
    EXECUTE FUNCTION update_kg_edge_timestamp();

-- ============================================================================
-- 11. VIEWS FOR ANALYTICS
-- ============================================================================

-- Graph statistics view
CREATE OR REPLACE VIEW knowledge_graph_statistics AS
SELECT
    n.tenant_id,
    n.project_id,
    COUNT(DISTINCT n.id) as total_nodes,
    COUNT(DISTINCT e.id) as total_edges,
    COUNT(DISTINCT e.id) FILTER (WHERE e.is_active = TRUE) as active_edges,
    COUNT(DISTINCT e.relation) as unique_relations,
    AVG(e.edge_weight) FILTER (WHERE e.is_active = TRUE) as avg_edge_weight,
    AVG(e.confidence) FILTER (WHERE e.is_active = TRUE) as avg_confidence,
    COUNT(DISTINCT e.id) FILTER (WHERE e.bidirectional = TRUE) as bidirectional_edges,
    MAX(n.created_at) as latest_node_created,
    MAX(e.created_at) as latest_edge_created
FROM knowledge_graph_nodes n
LEFT JOIN knowledge_graph_edges e ON (
    e.tenant_id = n.tenant_id AND e.project_id = n.project_id
)
GROUP BY n.tenant_id, n.project_id;

COMMENT ON VIEW knowledge_graph_statistics IS 'Aggregated statistics for knowledge graphs';

-- Top nodes by connectivity
CREATE OR REPLACE VIEW top_connected_nodes AS
SELECT
    n.id,
    n.tenant_id,
    n.project_id,
    n.node_id,
    n.label,
    COUNT(DISTINCT e_out.id) as out_degree,
    COUNT(DISTINCT e_in.id) as in_degree,
    COUNT(DISTINCT e_out.id) + COUNT(DISTINCT e_in.id) as total_degree
FROM knowledge_graph_nodes n
LEFT JOIN knowledge_graph_edges e_out ON e_out.source_node_id = n.id AND e_out.is_active = TRUE
LEFT JOIN knowledge_graph_edges e_in ON e_in.target_node_id = n.id AND e_in.is_active = TRUE
GROUP BY n.id, n.tenant_id, n.project_id, n.node_id, n.label
HAVING COUNT(DISTINCT e_out.id) + COUNT(DISTINCT e_in.id) > 0
ORDER BY total_degree DESC
LIMIT 100;

COMMENT ON VIEW top_connected_nodes IS 'Top 100 most connected nodes by degree';

-- ============================================================================
-- END OF GRAPH ENHANCEMENTS SCHEMA
-- ============================================================================

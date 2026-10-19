-- ============================================================================
-- Reflection System Schema for RAE - Enterprise Implementation
-- ============================================================================
-- This schema implements a complete hierarchical reflection system with:
-- - Hierarchical reflections (insight → meta-insight)
-- - Reflection scoring and prioritization
-- - Reflection graphs with typed relationships
-- - Clustering and caching support
-- - Full telemetry and audit trail
-- ============================================================================

-- ============================================================================
-- 1. REFLECTION_TYPES ENUM
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE reflection_type AS ENUM (
        'insight',       -- Basic insight from memories
        'analysis',      -- Analytical reflection
        'pattern',       -- Pattern recognition
        'meta',          -- Meta-insight (reflection on reflections)
        'synthesis'      -- Synthesized knowledge
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON TYPE reflection_type IS 'Types of reflections in the hierarchical system';

-- ============================================================================
-- 2. REFLECTIONS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS reflections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    -- Content
    content TEXT NOT NULL,
    summary VARCHAR(500),

    -- Classification
    type reflection_type NOT NULL DEFAULT 'insight',
    priority INTEGER NOT NULL DEFAULT 3 CHECK (priority BETWEEN 1 AND 5),

    -- Scoring
    score DECIMAL(5, 3) NOT NULL DEFAULT 0.5 CHECK (score BETWEEN 0 AND 1),
    novelty_score DECIMAL(5, 3) CHECK (novelty_score BETWEEN 0 AND 1),
    importance_score DECIMAL(5, 3) CHECK (importance_score BETWEEN 0 AND 1),
    utility_score DECIMAL(5, 3) CHECK (utility_score BETWEEN 0 AND 1),
    confidence_score DECIMAL(5, 3) CHECK (confidence_score BETWEEN 0 AND 1),

    -- Hierarchical structure
    parent_reflection_id UUID REFERENCES reflections(id) ON DELETE CASCADE,
    depth_level INTEGER NOT NULL DEFAULT 0 CHECK (depth_level >= 0),

    -- Source tracking
    source_memory_ids UUID[] NOT NULL DEFAULT '{}',
    source_reflection_ids UUID[] DEFAULT '{}',

    -- Embeddings for similarity search
    embedding vector(1536),  -- OpenAI ada-002 dimensions

    -- Clustering metadata
    cluster_id VARCHAR(100),
    cluster_centroid vector(1536),

    -- Caching
    cache_key VARCHAR(255),
    reuse_count INTEGER DEFAULT 0,

    -- Telemetry
    generation_model VARCHAR(100),
    generation_duration_ms INTEGER,
    generation_tokens_used INTEGER,
    generation_cost_usd DECIMAL(10, 6),

    -- Metadata
    tags TEXT[],
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    accessed_count INTEGER DEFAULT 0,

    -- Constraints
    CONSTRAINT reflections_tenant_project_idx UNIQUE (tenant_id, project_id, id)
);

-- Indexes for fast queries
CREATE INDEX IF NOT EXISTS idx_reflections_tenant_project ON reflections(tenant_id, project_id);
CREATE INDEX IF NOT EXISTS idx_reflections_type ON reflections(type);
CREATE INDEX IF NOT EXISTS idx_reflections_priority ON reflections(priority);
CREATE INDEX IF NOT EXISTS idx_reflections_score ON reflections(score DESC);
CREATE INDEX IF NOT EXISTS idx_reflections_parent ON reflections(parent_reflection_id);
CREATE INDEX IF NOT EXISTS idx_reflections_cluster ON reflections(cluster_id);
CREATE INDEX IF NOT EXISTS idx_reflections_cache_key ON reflections(cache_key);
CREATE INDEX IF NOT EXISTS idx_reflections_created_at ON reflections(created_at DESC);

-- Vector similarity search index
CREATE INDEX IF NOT EXISTS idx_reflections_embedding ON reflections USING ivfflat (embedding vector_cosine_ops);

-- Enable RLS
ALTER TABLE reflections ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Tenants can only see their own reflections
CREATE POLICY reflections_tenant_isolation ON reflections
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE));

COMMENT ON TABLE reflections IS 'Hierarchical reflection system with scoring, clustering, and caching';
COMMENT ON COLUMN reflections.type IS 'Type of reflection: insight, analysis, pattern, meta, synthesis';
COMMENT ON COLUMN reflections.priority IS 'Priority level 1-5 (5 = highest)';
COMMENT ON COLUMN reflections.score IS 'Overall reflection quality score (0-1)';
COMMENT ON COLUMN reflections.novelty_score IS 'How novel/unique is this reflection (0-1)';
COMMENT ON COLUMN reflections.importance_score IS 'How important/significant (0-1)';
COMMENT ON COLUMN reflections.utility_score IS 'How useful/actionable (0-1)';
COMMENT ON COLUMN reflections.confidence_score IS 'Model confidence in reflection (0-1)';
COMMENT ON COLUMN reflections.parent_reflection_id IS 'Parent reflection for hierarchical structure';
COMMENT ON COLUMN reflections.depth_level IS 'Depth in hierarchy: 0=base, 1=meta, 2=meta-meta, etc.';
COMMENT ON COLUMN reflections.cluster_id IS 'Clustering group identifier for related reflections';

-- ============================================================================
-- 3. REFLECTION_RELATIONSHIPS TABLE (ReflectionGraph)
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE reflection_relation_type AS ENUM (
        'supports',      -- Reflection B supports/reinforces reflection A
        'contradicts',   -- Reflection B contradicts reflection A
        'refines',       -- Reflection B refines/elaborates on reflection A
        'generalizes',   -- Reflection B generalizes reflection A
        'exemplifies',   -- Reflection B is an example of reflection A
        'derives_from',  -- Reflection B is derived from reflection A
        'relates_to'     -- Generic relationship
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON TYPE reflection_relation_type IS 'Types of relationships between reflections';

CREATE TABLE IF NOT EXISTS reflection_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    -- Relationship
    source_reflection_id UUID NOT NULL REFERENCES reflections(id) ON DELETE CASCADE,
    target_reflection_id UUID NOT NULL REFERENCES reflections(id) ON DELETE CASCADE,
    relation_type reflection_relation_type NOT NULL,

    -- Strength and confidence
    strength DECIMAL(5, 3) NOT NULL DEFAULT 0.5 CHECK (strength BETWEEN 0 AND 1),
    confidence DECIMAL(5, 3) NOT NULL DEFAULT 0.5 CHECK (confidence BETWEEN 0 AND 1),

    -- Evidence
    supporting_evidence TEXT[],

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    -- Constraints
    CONSTRAINT reflection_relationships_no_self_loop CHECK (source_reflection_id != target_reflection_id),
    CONSTRAINT reflection_relationships_unique_edge UNIQUE (source_reflection_id, target_reflection_id, relation_type)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_reflection_relationships_source ON reflection_relationships(source_reflection_id);
CREATE INDEX IF NOT EXISTS idx_reflection_relationships_target ON reflection_relationships(target_reflection_id);
CREATE INDEX IF NOT EXISTS idx_reflection_relationships_type ON reflection_relationships(relation_type);
CREATE INDEX IF NOT EXISTS idx_reflection_relationships_strength ON reflection_relationships(strength DESC);

-- Enable RLS
ALTER TABLE reflection_relationships ENABLE ROW LEVEL SECURITY;

-- RLS Policy
CREATE POLICY reflection_relationships_tenant_isolation ON reflection_relationships
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE));

COMMENT ON TABLE reflection_relationships IS 'Graph of relationships between reflections';
COMMENT ON COLUMN reflection_relationships.relation_type IS 'Type of relationship: supports, contradicts, refines, etc.';
COMMENT ON COLUMN reflection_relationships.strength IS 'Strength of the relationship (0-1)';
COMMENT ON COLUMN reflection_relationships.confidence IS 'Confidence in this relationship (0-1)';

-- ============================================================================
-- 4. REFLECTION_CLUSTERS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS reflection_clusters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    -- Cluster identification
    cluster_id VARCHAR(100) NOT NULL,

    -- Cluster properties
    centroid vector(1536),
    size INTEGER NOT NULL DEFAULT 0,
    cohesion_score DECIMAL(5, 3) CHECK (cohesion_score BETWEEN 0 AND 1),

    -- Summary
    cluster_summary TEXT,
    dominant_themes TEXT[],

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    -- Constraints
    CONSTRAINT reflection_clusters_unique UNIQUE (tenant_id, project_id, cluster_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_reflection_clusters_tenant_project ON reflection_clusters(tenant_id, project_id);
CREATE INDEX IF NOT EXISTS idx_reflection_clusters_cluster_id ON reflection_clusters(cluster_id);

-- Enable RLS
ALTER TABLE reflection_clusters ENABLE ROW LEVEL SECURITY;

-- RLS Policy
CREATE POLICY reflection_clusters_tenant_isolation ON reflection_clusters
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE));

COMMENT ON TABLE reflection_clusters IS 'Clustering metadata for grouped reflections';
COMMENT ON COLUMN reflection_clusters.cohesion_score IS 'How cohesive/tight is this cluster (0-1)';

-- ============================================================================
-- 5. REFLECTION_USAGE_LOG TABLE (Telemetry)
-- ============================================================================

CREATE TABLE IF NOT EXISTS reflection_usage_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    -- Reflection reference
    reflection_id UUID NOT NULL REFERENCES reflections(id) ON DELETE CASCADE,

    -- Usage context
    usage_type VARCHAR(50) NOT NULL, -- 'query', 'api_call', 'agent_execution', etc.
    query_text TEXT,

    -- Results
    relevance_score DECIMAL(5, 3) CHECK (relevance_score BETWEEN 0 AND 1),
    rank_position INTEGER,

    -- User context
    user_id VARCHAR(255),
    session_id VARCHAR(255),

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamp
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_reflection_usage_log_reflection ON reflection_usage_log(reflection_id);
CREATE INDEX IF NOT EXISTS idx_reflection_usage_log_tenant_project ON reflection_usage_log(tenant_id, project_id);
CREATE INDEX IF NOT EXISTS idx_reflection_usage_log_timestamp ON reflection_usage_log(timestamp DESC);

-- Enable RLS
ALTER TABLE reflection_usage_log ENABLE ROW LEVEL SECURITY;

-- RLS Policy
CREATE POLICY reflection_usage_log_tenant_isolation ON reflection_usage_log
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE));

COMMENT ON TABLE reflection_usage_log IS 'Audit log of reflection usage for analytics and telemetry';

-- ============================================================================
-- 6. HELPER FUNCTIONS
-- ============================================================================

-- Function to calculate composite reflection score
CREATE OR REPLACE FUNCTION calculate_reflection_score(
    p_novelty DECIMAL,
    p_importance DECIMAL,
    p_utility DECIMAL,
    p_confidence DECIMAL
) RETURNS DECIMAL AS $$
BEGIN
    -- Weighted average: importance (0.4), utility (0.3), novelty (0.2), confidence (0.1)
    RETURN (
        COALESCE(p_importance, 0.5) * 0.4 +
        COALESCE(p_utility, 0.5) * 0.3 +
        COALESCE(p_novelty, 0.5) * 0.2 +
        COALESCE(p_confidence, 0.5) * 0.1
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION calculate_reflection_score IS 'Calculate composite reflection score from component scores';

-- Function to update reflection timestamps
CREATE OR REPLACE FUNCTION update_reflection_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for automatic timestamp updates
CREATE TRIGGER trigger_update_reflection_timestamp
    BEFORE UPDATE ON reflections
    FOR EACH ROW
    EXECUTE FUNCTION update_reflection_timestamp();

-- Function to increment reflection access count
CREATE OR REPLACE FUNCTION increment_reflection_access(p_reflection_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE reflections
    SET
        accessed_count = accessed_count + 1,
        last_accessed_at = NOW()
    WHERE id = p_reflection_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION increment_reflection_access IS 'Increment access count and update last accessed timestamp';

-- Function to detect cycles in reflection relationships
CREATE OR REPLACE FUNCTION detect_reflection_cycle(
    p_source_id UUID,
    p_target_id UUID
) RETURNS BOOLEAN AS $$
DECLARE
    has_cycle BOOLEAN := FALSE;
BEGIN
    -- Use recursive CTE to check if adding this edge would create a cycle
    WITH RECURSIVE path AS (
        SELECT
            target_reflection_id AS node_id,
            1 AS depth,
            ARRAY[p_target_id] AS visited
        FROM reflection_relationships
        WHERE source_reflection_id = p_target_id

        UNION ALL

        SELECT
            rr.target_reflection_id,
            p.depth + 1,
            p.visited || rr.target_reflection_id
        FROM reflection_relationships rr
        JOIN path p ON p.node_id = rr.source_reflection_id
        WHERE
            p.depth < 50  -- Prevent infinite recursion
            AND NOT (rr.target_reflection_id = ANY(p.visited))  -- Detect cycle
    )
    SELECT EXISTS (
        SELECT 1 FROM path WHERE node_id = p_source_id
    ) INTO has_cycle;

    RETURN has_cycle;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION detect_reflection_cycle IS 'Detect if adding a relationship would create a cycle';

-- ============================================================================
-- 7. VIEWS FOR ANALYTICS
-- ============================================================================

-- View for reflection statistics per project
CREATE OR REPLACE VIEW reflection_statistics AS
SELECT
    tenant_id,
    project_id,
    COUNT(*) as total_reflections,
    COUNT(*) FILTER (WHERE type = 'insight') as insights,
    COUNT(*) FILTER (WHERE type = 'analysis') as analyses,
    COUNT(*) FILTER (WHERE type = 'pattern') as patterns,
    COUNT(*) FILTER (WHERE type = 'meta') as meta_insights,
    COUNT(*) FILTER (WHERE type = 'synthesis') as syntheses,
    AVG(score) as avg_score,
    AVG(priority) as avg_priority,
    COUNT(*) FILTER (WHERE parent_reflection_id IS NOT NULL) as hierarchical_count,
    MAX(depth_level) as max_depth,
    SUM(accessed_count) as total_accesses,
    SUM(generation_cost_usd) as total_generation_cost_usd
FROM reflections
GROUP BY tenant_id, project_id;

COMMENT ON VIEW reflection_statistics IS 'Aggregated statistics for reflections per tenant/project';

-- View for top reflections
CREATE OR REPLACE VIEW top_reflections AS
SELECT
    id,
    tenant_id,
    project_id,
    type,
    summary,
    score,
    priority,
    accessed_count,
    created_at
FROM reflections
ORDER BY score DESC, accessed_count DESC, priority DESC
LIMIT 100;

COMMENT ON VIEW top_reflections IS 'Top 100 reflections by score, access count, and priority';

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================

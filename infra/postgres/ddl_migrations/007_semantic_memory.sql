-- ============================================================================
-- Semantic Memory Schema for RAE - Enterprise Knowledge Nodes
-- ============================================================================
-- This schema implements a semantic memory layer with:
-- - Semantic nodes (concepts, topics, entities)
-- - Canonical forms and definitions
-- - Semantic relationships
-- - Priority decay (TTL/LTM model)
-- - Ontological classification
-- ============================================================================

-- ============================================================================
-- 1. SEMANTIC_NODE_TYPES ENUM
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE semantic_node_type AS ENUM (
        'concept',       -- Abstract concept
        'topic',         -- Domain topic
        'entity',        -- Named entity
        'term',          -- Technical term
        'category',      -- Category/classification
        'relation'       -- Relationship type
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON TYPE semantic_node_type IS 'Types of semantic nodes in the knowledge graph';

-- ============================================================================
-- 2. SEMANTIC_NODES TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS semantic_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    -- Core identification
    node_id VARCHAR(500) NOT NULL,  -- Canonical identifier (e.g., "authentication_pattern")
    label VARCHAR(500) NOT NULL,    -- Human-readable label
    node_type semantic_node_type NOT NULL DEFAULT 'concept',

    -- Canonical form
    canonical_form TEXT NOT NULL,   -- Standardized representation
    aliases TEXT[] DEFAULT '{}',    -- Alternative names/synonyms

    -- Definitions and context
    definition TEXT,                -- Primary definition
    definitions JSONB DEFAULT '[]', -- Multiple definitions with sources
    context TEXT,                   -- Contextual information
    examples TEXT[],                -- Usage examples

    -- Ontological classification
    categories TEXT[] DEFAULT '{}', -- Categories this node belongs to
    domain VARCHAR(255),            -- Domain (e.g., "security", "architecture")

    -- Relations (stored as JSONB for flexibility)
    relations JSONB DEFAULT '{}',   -- Related nodes: {"related_to": ["node1", "node2"], ...}

    -- Embeddings for semantic search
    embedding vector(1536),         -- Vector embedding

    -- Priority and decay (TTL/LTM model)
    priority INTEGER NOT NULL DEFAULT 3 CHECK (priority BETWEEN 1 AND 5),
    importance_score DECIMAL(5, 3) NOT NULL DEFAULT 0.5 CHECK (importance_score BETWEEN 0 AND 1),

    -- Decay tracking
    last_reinforced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    reinforcement_count INTEGER DEFAULT 0,
    decay_rate DECIMAL(5, 3) DEFAULT 0.01 CHECK (decay_rate >= 0 AND decay_rate <= 1),

    -- Decay status
    is_degraded BOOLEAN DEFAULT FALSE,
    degradation_timestamp TIMESTAMP WITH TIME ZONE,

    -- Source tracking
    source_memory_ids UUID[] DEFAULT '{}',
    extraction_model VARCHAR(100),
    extraction_confidence DECIMAL(5, 3),

    -- Metadata
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    accessed_count INTEGER DEFAULT 0,

    -- Constraints
    CONSTRAINT semantic_nodes_unique UNIQUE (tenant_id, project_id, node_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_semantic_nodes_tenant_project ON semantic_nodes(tenant_id, project_id);
CREATE INDEX IF NOT EXISTS idx_semantic_nodes_node_id ON semantic_nodes(node_id);
CREATE INDEX IF NOT EXISTS idx_semantic_nodes_type ON semantic_nodes(node_type);
CREATE INDEX IF NOT EXISTS idx_semantic_nodes_domain ON semantic_nodes(domain);
CREATE INDEX IF NOT EXISTS idx_semantic_nodes_priority ON semantic_nodes(priority);
CREATE INDEX IF NOT EXISTS idx_semantic_nodes_degraded ON semantic_nodes(is_degraded);
CREATE INDEX IF NOT EXISTS idx_semantic_nodes_last_reinforced ON semantic_nodes(last_reinforced_at);

-- Vector similarity search
CREATE INDEX IF NOT EXISTS idx_semantic_nodes_embedding ON semantic_nodes USING ivfflat (embedding vector_cosine_ops);

-- Full-text search on canonical_form and definition
CREATE INDEX IF NOT EXISTS idx_semantic_nodes_fulltext ON semantic_nodes
    USING gin(to_tsvector('english', canonical_form || ' ' || COALESCE(definition, '')));

-- Enable RLS
ALTER TABLE semantic_nodes ENABLE ROW LEVEL SECURITY;

-- RLS Policy
CREATE POLICY semantic_nodes_tenant_isolation ON semantic_nodes
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE));

COMMENT ON TABLE semantic_nodes IS 'Semantic knowledge nodes with canonical forms and decay modeling';
COMMENT ON COLUMN semantic_nodes.canonical_form IS 'Standardized/normalized representation';
COMMENT ON COLUMN semantic_nodes.priority IS 'Priority level 1-5, affects decay rate';
COMMENT ON COLUMN semantic_nodes.importance_score IS 'Importance score 0-1';
COMMENT ON COLUMN semantic_nodes.decay_rate IS 'Rate of priority decay per day (0-1)';
COMMENT ON COLUMN semantic_nodes.is_degraded IS 'True if node has decayed below threshold';

-- ============================================================================
-- 3. SEMANTIC_RELATIONSHIPS TABLE
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE semantic_relation_type AS ENUM (
        'is_a',          -- Hyponymy (X is a type of Y)
        'part_of',       -- Meronymy (X is part of Y)
        'related_to',    -- Generic relation
        'synonym_of',    -- Synonymy
        'antonym_of',    -- Antonymy
        'causes',        -- Causation
        'requires',      -- Dependency
        'similar_to',    -- Similarity
        'derives_from',  -- Etymology/derivation
        'implements',    -- Implementation
        'uses'           -- Usage relation
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS semantic_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    -- Relationship
    source_node_id UUID NOT NULL REFERENCES semantic_nodes(id) ON DELETE CASCADE,
    target_node_id UUID NOT NULL REFERENCES semantic_nodes(id) ON DELETE CASCADE,
    relation_type semantic_relation_type NOT NULL,

    -- Strength and confidence
    strength DECIMAL(5, 3) NOT NULL DEFAULT 0.5 CHECK (strength BETWEEN 0 AND 1),
    confidence DECIMAL(5, 3) NOT NULL DEFAULT 0.5 CHECK (confidence BETWEEN 0 AND 1),

    -- Evidence
    evidence_text TEXT,
    source_memory_ids UUID[] DEFAULT '{}',

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    -- Constraints
    CONSTRAINT semantic_relationships_no_self_loop CHECK (source_node_id != target_node_id),
    CONSTRAINT semantic_relationships_unique UNIQUE (source_node_id, target_node_id, relation_type)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_semantic_relationships_source ON semantic_relationships(source_node_id);
CREATE INDEX IF NOT EXISTS idx_semantic_relationships_target ON semantic_relationships(target_node_id);
CREATE INDEX IF NOT EXISTS idx_semantic_relationships_type ON semantic_relationships(relation_type);

-- Enable RLS
ALTER TABLE semantic_relationships ENABLE ROW LEVEL SECURITY;

-- RLS Policy
CREATE POLICY semantic_relationships_tenant_isolation ON semantic_relationships
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE));

COMMENT ON TABLE semantic_relationships IS 'Typed relationships between semantic nodes';

-- ============================================================================
-- 4. SEMANTIC_INDEX TABLE (for fast topic → vector lookups)
-- ============================================================================

CREATE TABLE IF NOT EXISTS semantic_index (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    -- Topic/term
    topic VARCHAR(500) NOT NULL,
    normalized_topic VARCHAR(500) NOT NULL,  -- Lowercased, stemmed

    -- References
    semantic_node_id UUID REFERENCES semantic_nodes(id) ON DELETE CASCADE,

    -- Embedding for quick lookup
    topic_embedding vector(1536),

    -- Statistics
    occurrence_count INTEGER DEFAULT 1,
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    -- Constraints
    CONSTRAINT semantic_index_unique UNIQUE (tenant_id, project_id, normalized_topic)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_semantic_index_tenant_project ON semantic_index(tenant_id, project_id);
CREATE INDEX IF NOT EXISTS idx_semantic_index_normalized_topic ON semantic_index(normalized_topic);
CREATE INDEX IF NOT EXISTS idx_semantic_index_node ON semantic_index(semantic_node_id);
CREATE INDEX IF NOT EXISTS idx_semantic_index_embedding ON semantic_index
    USING ivfflat (topic_embedding vector_cosine_ops);

-- Enable RLS
ALTER TABLE semantic_index ENABLE ROW LEVEL SECURITY;

-- RLS Policy
CREATE POLICY semantic_index_tenant_isolation ON semantic_index
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE));

COMMENT ON TABLE semantic_index IS 'Fast lookup index for topic → semantic node mapping';

-- ============================================================================
-- 5. HELPER FUNCTIONS
-- ============================================================================

-- Function to update semantic node timestamp
CREATE OR REPLACE FUNCTION update_semantic_node_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for automatic timestamp updates
CREATE TRIGGER trigger_update_semantic_node_timestamp
    BEFORE UPDATE ON semantic_nodes
    FOR EACH ROW
    EXECUTE FUNCTION update_semantic_node_timestamp();

-- Function to reinforce semantic node (reset decay)
CREATE OR REPLACE FUNCTION reinforce_semantic_node(p_node_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE semantic_nodes
    SET
        last_reinforced_at = NOW(),
        reinforcement_count = reinforcement_count + 1,
        accessed_count = accessed_count + 1,
        last_accessed_at = NOW(),
        is_degraded = FALSE,
        degradation_timestamp = NULL,
        -- Increase priority on reinforcement (max 5)
        priority = LEAST(5, priority + 1)
    WHERE id = p_node_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION reinforce_semantic_node IS 'Reinforce a semantic node, resetting its decay timer';

-- Function to apply decay to semantic nodes
CREATE OR REPLACE FUNCTION apply_semantic_decay(
    p_tenant_id VARCHAR(255),
    p_project_id VARCHAR(255),
    p_decay_threshold_days INTEGER DEFAULT 60
)
RETURNS TABLE (
    degraded_count INTEGER,
    total_decayed INTEGER
) AS $$
DECLARE
    v_degraded_count INTEGER := 0;
    v_total_decayed INTEGER := 0;
BEGIN
    -- Calculate days since last reinforcement
    WITH decayed_nodes AS (
        UPDATE semantic_nodes
        SET
            is_degraded = TRUE,
            degradation_timestamp = CASE
                WHEN is_degraded = FALSE THEN NOW()
                ELSE degradation_timestamp
            END,
            -- Decrease priority on decay (min 1)
            priority = GREATEST(1, priority - 1)
        WHERE
            tenant_id = p_tenant_id
            AND project_id = p_project_id
            AND EXTRACT(EPOCH FROM (NOW() - last_reinforced_at)) / 86400 > p_decay_threshold_days
            AND is_degraded = FALSE
        RETURNING id
    )
    SELECT COUNT(*) INTO v_degraded_count FROM decayed_nodes;

    -- Count total degraded nodes
    SELECT COUNT(*) INTO v_total_decayed
    FROM semantic_nodes
    WHERE tenant_id = p_tenant_id
      AND project_id = p_project_id
      AND is_degraded = TRUE;

    RETURN QUERY SELECT v_degraded_count, v_total_decayed;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION apply_semantic_decay IS 'Apply decay to semantic nodes not accessed in threshold days';

-- Function to normalize topic for indexing
CREATE OR REPLACE FUNCTION normalize_topic(p_topic TEXT)
RETURNS TEXT AS $$
BEGIN
    -- Lowercase, trim, replace multiple spaces with single space
    RETURN lower(trim(regexp_replace(p_topic, '\s+', ' ', 'g')));
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION normalize_topic IS 'Normalize topic string for consistent indexing';

-- ============================================================================
-- 6. VIEWS FOR ANALYTICS
-- ============================================================================

-- View for semantic node statistics
CREATE OR REPLACE VIEW semantic_node_statistics AS
SELECT
    tenant_id,
    project_id,
    COUNT(*) as total_nodes,
    COUNT(*) FILTER (WHERE node_type = 'concept') as concepts,
    COUNT(*) FILTER (WHERE node_type = 'topic') as topics,
    COUNT(*) FILTER (WHERE node_type = 'entity') as entities,
    COUNT(*) FILTER (WHERE node_type = 'term') as terms,
    COUNT(*) FILTER (WHERE node_type = 'category') as categories,
    COUNT(*) FILTER (WHERE is_degraded = TRUE) as degraded_nodes,
    AVG(priority) as avg_priority,
    AVG(importance_score) as avg_importance,
    SUM(accessed_count) as total_accesses,
    SUM(reinforcement_count) as total_reinforcements
FROM semantic_nodes
GROUP BY tenant_id, project_id;

COMMENT ON VIEW semantic_node_statistics IS 'Aggregated statistics for semantic nodes';

-- View for most active semantic nodes
CREATE OR REPLACE VIEW top_semantic_nodes AS
SELECT
    id,
    tenant_id,
    project_id,
    node_id,
    label,
    node_type,
    priority,
    importance_score,
    accessed_count,
    reinforcement_count,
    last_accessed_at
FROM semantic_nodes
WHERE is_degraded = FALSE
ORDER BY accessed_count DESC, reinforcement_count DESC, priority DESC
LIMIT 100;

COMMENT ON VIEW top_semantic_nodes IS 'Top 100 most active semantic nodes';

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================

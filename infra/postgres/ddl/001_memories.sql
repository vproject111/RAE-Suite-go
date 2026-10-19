CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS vector;

-- Drop all tables to ensure clean slate if script runs multiple times
DROP TABLE IF EXISTS memory_embeddings CASCADE;
DROP TABLE IF EXISTS knowledge_graph_edges CASCADE;
DROP TABLE IF EXISTS knowledge_graph_nodes CASCADE;
DROP TABLE IF EXISTS user_tenant_roles CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
DROP TABLE IF EXISTS budgets CASCADE;
DROP TABLE IF EXISTS access_logs CASCADE;
DROP TABLE IF EXISTS token_savings_log CASCADE;
DROP TABLE IF EXISTS memories CASCADE;

-- 1. Memories
CREATE TABLE memories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  content TEXT NOT NULL,
  source VARCHAR(255),
  importance REAL DEFAULT 0.5,
  layer VARCHAR(50),
  tags TEXT[],
  timestamp TIMESTAMPTZ DEFAULT NOW(),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  last_accessed_at TIMESTAMPTZ DEFAULT NOW(),
  usage_count INTEGER DEFAULT 0,
  project VARCHAR(255),
  metadata JSONB DEFAULT '{}',
  agent_id VARCHAR(255) NOT NULL DEFAULT 'default',
  expires_at TIMESTAMPTZ,
  memory_type VARCHAR(50),
  strength REAL DEFAULT 1.0,
  session_id VARCHAR(255),
  embedding vector
);
CREATE INDEX idx_memories_tenant_id ON memories (tenant_id);
CREATE INDEX idx_memories_project ON memories (project);
CREATE INDEX idx_memories_agent_id ON memories (agent_id);

-- 2. Embeddings
CREATE TABLE memory_embeddings (
    memory_id UUID REFERENCES memories(id) ON DELETE CASCADE,
    embedding vector NOT NULL,
    model_name VARCHAR(255) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (memory_id, model_name)
);

-- 3. Tenants
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    tier VARCHAR(50) DEFAULT 'free' NOT NULL,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    status VARCHAR(50) DEFAULT 'active',
    contact_email VARCHAR(255),
    company_name VARCHAR(255),
    current_memory_count INTEGER DEFAULT 0,
    current_project_count INTEGER DEFAULT 0,
    api_calls_today INTEGER DEFAULT 0,
    subscription_start TIMESTAMPTZ,
    subscription_end TIMESTAMPTZ
);

-- 4. Roles
CREATE TABLE user_tenant_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    project_ids TEXT[] DEFAULT '{}',
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    assigned_by VARCHAR(255),
    expires_at TIMESTAMPTZ,
    CONSTRAINT uq_user_tenant_role UNIQUE (user_id, tenant_id)
);
CREATE INDEX idx_utr_user_id ON user_tenant_roles (user_id);
CREATE INDEX idx_utr_tenant_id ON user_tenant_roles (tenant_id);

-- 5. Budgets
CREATE TABLE budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    project_id VARCHAR NOT NULL,
    monthly_limit FLOAT,
    monthly_usage FLOAT DEFAULT 0.0 NOT NULL,
    daily_limit FLOAT,
    daily_usage FLOAT DEFAULT 0.0 NOT NULL,
    last_usage_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT uq_tenant_project_budget UNIQUE (tenant_id, project_id)
);

-- 6. Access Logs
CREATE TABLE access_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    allowed BOOLEAN NOT NULL,
    denial_reason TEXT,
    ip_address VARCHAR(50),
    user_agent TEXT,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);
CREATE INDEX ix_access_logs_tenant_id ON access_logs (tenant_id);
CREATE INDEX ix_access_logs_timestamp ON access_logs (timestamp);
CREATE INDEX ix_access_logs_user_id ON access_logs (user_id);

-- 7. Token Savings
CREATE TABLE token_savings_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    project_id VARCHAR NOT NULL,
    request_id VARCHAR,
    predicted_tokens INTEGER DEFAULT 0,
    real_tokens INTEGER DEFAULT 0,
    saved_tokens INTEGER DEFAULT 0,
    estimated_cost_saved_usd FLOAT DEFAULT 0.0,
    savings_type VARCHAR,
    model VARCHAR,
    timestamp TIMESTAMPTZ DEFAULT NOW()
);

-- 8. Knowledge Graph
CREATE TABLE knowledge_graph_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    project_id VARCHAR NOT NULL,
    node_id VARCHAR NOT NULL,
    label VARCHAR NOT NULL,
    properties JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_project_node UNIQUE (tenant_id, project_id, node_id)
);
CREATE INDEX idx_kg_nodes_tp ON knowledge_graph_nodes (tenant_id, project_id);

CREATE TABLE knowledge_graph_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    project_id VARCHAR NOT NULL,
    source_node_id UUID NOT NULL REFERENCES knowledge_graph_nodes(id) ON DELETE CASCADE,
    target_node_id UUID NOT NULL REFERENCES knowledge_graph_nodes(id) ON DELETE CASCADE,
    relation VARCHAR NOT NULL,
    properties JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_kg_edges_relation ON knowledge_graph_edges (relation);

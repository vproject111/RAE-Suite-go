CREATE TABLE knowledge_graph_nodes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id VARCHAR(255) NOT NULL,
  project_id VARCHAR(255) NOT NULL,
  node_id VARCHAR(255) NOT NULL,
  label VARCHAR(255) NOT NULL,
  properties JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(tenant_id, project_id, node_id)
);

CREATE TABLE knowledge_graph_edges (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id VARCHAR(255) NOT NULL,
  project_id VARCHAR(255) NOT NULL,
  source_node_id UUID NOT NULL REFERENCES knowledge_graph_nodes(id),
  target_node_id UUID NOT NULL REFERENCES knowledge_graph_nodes(id),
  relation VARCHAR(255) NOT NULL,
  properties JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for common queries
CREATE INDEX idx_kg_nodes_tenant_project ON knowledge_graph_nodes(tenant_id, project_id);
CREATE INDEX idx_kg_nodes_node_id ON knowledge_graph_nodes(node_id);
CREATE INDEX idx_kg_edges_tenant_project ON knowledge_graph_edges(tenant_id, project_id);
CREATE INDEX idx_kg_edges_source ON knowledge_graph_edges(source_node_id);
CREATE INDEX idx_kg_edges_target ON knowledge_graph_edges(target_node_id);
CREATE INDEX idx_kg_edges_relation ON knowledge_graph_edges(relation);

ALTER TABLE knowledge_graph_nodes ENABLE ROW LEVEL SECURITY;
ALTER TABLE knowledge_graph_edges ENABLE ROW LEVEL SECURITY;

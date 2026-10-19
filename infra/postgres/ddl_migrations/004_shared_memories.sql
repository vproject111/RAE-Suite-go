CREATE TABLE shared_memories (
  memory_id UUID REFERENCES memories(id) ON DELETE CASCADE,
  owner_tenant_id VARCHAR(255) NOT NULL,
  shared_with_tenant_id VARCHAR(255) NOT NULL,
  permissions VARCHAR(32) NOT NULL DEFAULT 'read',
  PRIMARY KEY (memory_id, shared_with_tenant_id)
);

-- Przykładowa polityka (SELECT) — do dostosowania wraz z resztą
-- CREATE POLICY tenant_and_shared_isolation ON memories ... (zdefiniuj wg potrzeb w migracji)

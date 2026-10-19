ALTER TABLE memories ENABLE ROW LEVEL SECURITY;
-- Legacy tables no longer exist in current schema
-- ALTER TABLE episodic_memories ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE semantic_memories ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE procedural_memories ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_memories ON memories
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));

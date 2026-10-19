CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS vector;

-- Drop old tables if they exist to apply the new schema
DROP TABLE IF EXISTS procedural_memories CASCADE;
DROP TABLE IF EXISTS semantic_memories CASCADE;
DROP TABLE IF EXISTS episodic_memories CASCADE;
DROP TABLE IF EXISTS memories CASCADE;

-- Create the new, standardized memories table
CREATE TABLE memories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id VARCHAR(255) NOT NULL,
  content TEXT NOT NULL,
  source VARCHAR(255),
  importance REAL DEFAULT 0.5,
  layer VARCHAR(50), -- e.g., stm, ltm, rm (short-term, long-term, reflective)
  tags TEXT[],
  timestamp TIMESTAMPTZ DEFAULT NOW(),
  created_at TIMESTAMPTZ DEFAULT NOW(),

  -- Fields for scoring and heuristics
  last_accessed_at TIMESTAMPTZ DEFAULT NOW(),
  usage_count INTEGER DEFAULT 0,
  project VARCHAR(255) -- project/namespace for memory isolation
);
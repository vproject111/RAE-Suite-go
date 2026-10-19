-- ============================================================================
-- Cost Control Schema for RAE
-- ============================================================================
-- This schema implements enterprise-grade cost and token tracking for LLM usage
-- Including budgets, cost logs, and governance capabilities
-- ============================================================================

-- ============================================================================
-- 1. BUDGETS TABLE
-- ============================================================================
-- Tracks cost and token budgets per tenant/project with daily and monthly limits
CREATE TABLE IF NOT EXISTS budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    -- USD Cost Limits
    daily_limit_usd DECIMAL(10, 4) DEFAULT NULL,
    monthly_limit_usd DECIMAL(10, 4) DEFAULT NULL,

    -- USD Usage Tracking
    daily_usage_usd DECIMAL(10, 4) DEFAULT 0.0 NOT NULL,
    monthly_usage_usd DECIMAL(10, 4) DEFAULT 0.0 NOT NULL,

    -- Token Limits (NEW - as per COST.md requirement)
    daily_tokens_limit BIGINT DEFAULT NULL,
    monthly_tokens_limit BIGINT DEFAULT NULL,

    -- Token Usage Tracking (NEW)
    daily_tokens_used BIGINT DEFAULT 0 NOT NULL,
    monthly_tokens_used BIGINT DEFAULT 0 NOT NULL,

    -- Timestamps for tracking and resets
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_usage_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_token_update_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_daily_reset TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_monthly_reset TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    -- Constraints
    CONSTRAINT budgets_tenant_project_unique UNIQUE (tenant_id, project_id),
    CONSTRAINT budgets_daily_limit_positive CHECK (daily_limit_usd IS NULL OR daily_limit_usd >= 0),
    CONSTRAINT budgets_monthly_limit_positive CHECK (monthly_limit_usd IS NULL OR monthly_limit_usd >= 0),
    CONSTRAINT budgets_daily_tokens_limit_positive CHECK (daily_tokens_limit IS NULL OR daily_tokens_limit >= 0),
    CONSTRAINT budgets_monthly_tokens_limit_positive CHECK (monthly_tokens_limit IS NULL OR monthly_tokens_limit >= 0),
    CONSTRAINT budgets_usage_non_negative CHECK (daily_usage_usd >= 0 AND monthly_usage_usd >= 0),
    CONSTRAINT budgets_tokens_used_non_negative CHECK (daily_tokens_used >= 0 AND monthly_tokens_used >= 0)
);

-- Indexes for fast budget lookups
CREATE INDEX IF NOT EXISTS idx_budgets_tenant_project ON budgets(tenant_id, project_id);
CREATE INDEX IF NOT EXISTS idx_budgets_tenant ON budgets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_budgets_last_usage ON budgets(last_usage_at);

-- Enable RLS (Row Level Security)
ALTER TABLE budgets ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Tenants can only see their own budgets
CREATE POLICY budgets_tenant_isolation ON budgets
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE));

COMMENT ON TABLE budgets IS 'Stores cost and token budgets for multi-tenant LLM usage control';
COMMENT ON COLUMN budgets.daily_limit_usd IS 'Daily spending limit in USD (NULL = unlimited)';
COMMENT ON COLUMN budgets.monthly_limit_usd IS 'Monthly spending limit in USD (NULL = unlimited)';
COMMENT ON COLUMN budgets.daily_tokens_limit IS 'Daily token limit (input + output) (NULL = unlimited)';
COMMENT ON COLUMN budgets.monthly_tokens_limit IS 'Monthly token limit (input + output) (NULL = unlimited)';
COMMENT ON COLUMN budgets.daily_usage_usd IS 'Current daily spending in USD (resets at midnight)';
COMMENT ON COLUMN budgets.monthly_usage_usd IS 'Current monthly spending in USD (resets on 1st of month)';
COMMENT ON COLUMN budgets.daily_tokens_used IS 'Current daily tokens used (resets at midnight)';
COMMENT ON COLUMN budgets.monthly_tokens_used IS 'Current monthly tokens used (resets on 1st of month)';

-- ============================================================================
-- 2. COST_LOGS TABLE
-- ============================================================================
-- Comprehensive audit log of all LLM API calls with token and cost tracking
CREATE TABLE IF NOT EXISTS cost_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    -- LLM Call Metadata
    model VARCHAR(255) NOT NULL,
    provider VARCHAR(100) NOT NULL, -- 'openai', 'anthropic', 'google', 'ollama'
    operation VARCHAR(100) NOT NULL, -- 'query', 'reflection', 'embedding', 'entity_extraction', etc.

    -- Token Tracking
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER GENERATED ALWAYS AS (input_tokens + output_tokens) STORED,

    -- Cost Tracking (per million tokens)
    input_cost_per_million DECIMAL(10, 4) NOT NULL DEFAULT 0,
    output_cost_per_million DECIMAL(10, 4) NOT NULL DEFAULT 0,
    total_cost_usd DECIMAL(10, 6) NOT NULL DEFAULT 0,

    -- Cache Tracking (for cost avoidance calculation)
    cache_hit BOOLEAN DEFAULT FALSE,
    cache_tokens_saved INTEGER DEFAULT 0,

    -- Request Context
    request_id VARCHAR(255), -- For tracing
    user_id VARCHAR(255), -- Optional user attribution

    -- Additional Metadata
    latency_ms INTEGER, -- Response time in milliseconds
    error BOOLEAN DEFAULT FALSE,
    error_message TEXT,

    -- Timestamps
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    -- Constraints
    CONSTRAINT cost_logs_tokens_non_negative CHECK (input_tokens >= 0 AND output_tokens >= 0),
    CONSTRAINT cost_logs_cost_non_negative CHECK (total_cost_usd >= 0)
);

-- Indexes for fast queries and aggregations
CREATE INDEX IF NOT EXISTS idx_cost_logs_tenant_project ON cost_logs(tenant_id, project_id);
CREATE INDEX IF NOT EXISTS idx_cost_logs_tenant ON cost_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_cost_logs_timestamp ON cost_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_cost_logs_model ON cost_logs(model);
CREATE INDEX IF NOT EXISTS idx_cost_logs_operation ON cost_logs(operation);
CREATE INDEX IF NOT EXISTS idx_cost_logs_tenant_timestamp ON cost_logs(tenant_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_cost_logs_cache_hit ON cost_logs(cache_hit) WHERE cache_hit = TRUE;

-- Enable RLS
ALTER TABLE cost_logs ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Tenants can only see their own logs
CREATE POLICY cost_logs_tenant_isolation ON cost_logs
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE));

COMMENT ON TABLE cost_logs IS 'Audit log of all LLM API calls with detailed token and cost tracking';
COMMENT ON COLUMN cost_logs.model IS 'LLM model name (e.g., gpt-4o, claude-3.5-sonnet)';
COMMENT ON COLUMN cost_logs.provider IS 'LLM provider (openai, anthropic, google, ollama)';
COMMENT ON COLUMN cost_logs.operation IS 'Type of operation (query, reflection, embedding, etc.)';
COMMENT ON COLUMN cost_logs.total_tokens IS 'Computed column: input_tokens + output_tokens';
COMMENT ON COLUMN cost_logs.cache_hit IS 'TRUE if response was served from cache (cost avoidance)';
COMMENT ON COLUMN cost_logs.cache_tokens_saved IS 'Number of tokens saved by cache hit';

-- ============================================================================
-- 3. HELPER FUNCTIONS
-- ============================================================================

-- Function to automatically reset daily/monthly usage counters
CREATE OR REPLACE FUNCTION reset_budget_counters()
RETURNS TRIGGER AS $$
DECLARE
    current_date TIMESTAMP WITH TIME ZONE := NOW();
BEGIN
    -- Reset daily counters if it's a new day
    IF DATE(NEW.last_daily_reset) < DATE(current_date) THEN
        NEW.daily_usage_usd := 0.0;
        NEW.daily_tokens_used := 0;
        NEW.last_daily_reset := current_date;
    END IF;

    -- Reset monthly counters if it's a new month
    IF (EXTRACT(YEAR FROM NEW.last_monthly_reset) < EXTRACT(YEAR FROM current_date)) OR
       (EXTRACT(YEAR FROM NEW.last_monthly_reset) = EXTRACT(YEAR FROM current_date) AND
        EXTRACT(MONTH FROM NEW.last_monthly_reset) < EXTRACT(MONTH FROM current_date)) THEN
        NEW.monthly_usage_usd := 0.0;
        NEW.monthly_tokens_used := 0;
        NEW.last_monthly_reset := current_date;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically reset counters
CREATE TRIGGER trigger_reset_budget_counters
    BEFORE UPDATE ON budgets
    FOR EACH ROW
    EXECUTE FUNCTION reset_budget_counters();

COMMENT ON FUNCTION reset_budget_counters() IS 'Automatically resets daily/monthly budget counters at day/month boundaries';

-- ============================================================================
-- 4. AGGREGATION VIEWS
-- ============================================================================

-- View for daily cost statistics per tenant
CREATE OR REPLACE VIEW daily_cost_stats AS
SELECT
    tenant_id,
    project_id,
    DATE(timestamp) as date,
    COUNT(*) as total_calls,
    SUM(input_tokens) as total_input_tokens,
    SUM(output_tokens) as total_output_tokens,
    SUM(total_tokens) as total_tokens,
    SUM(total_cost_usd) as total_cost_usd,
    AVG(total_cost_usd) as avg_cost_per_call,
    SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END) as cache_hits,
    SUM(CASE WHEN cache_hit THEN cache_tokens_saved ELSE 0 END) as total_tokens_saved
FROM cost_logs
WHERE error = FALSE
GROUP BY tenant_id, project_id, DATE(timestamp);

COMMENT ON VIEW daily_cost_stats IS 'Daily aggregated cost and token statistics per tenant/project';

-- View for monthly cost statistics per tenant
CREATE OR REPLACE VIEW monthly_cost_stats AS
SELECT
    tenant_id,
    project_id,
    DATE_TRUNC('month', timestamp) as month,
    COUNT(*) as total_calls,
    SUM(input_tokens) as total_input_tokens,
    SUM(output_tokens) as total_output_tokens,
    SUM(total_tokens) as total_tokens,
    SUM(total_cost_usd) as total_cost_usd,
    AVG(total_cost_usd) as avg_cost_per_call,
    SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END) as cache_hits,
    SUM(CASE WHEN cache_hit THEN cache_tokens_saved ELSE 0 END) as total_tokens_saved
FROM cost_logs
WHERE error = FALSE
GROUP BY tenant_id, project_id, DATE_TRUNC('month', timestamp);

COMMENT ON VIEW monthly_cost_stats IS 'Monthly aggregated cost and token statistics per tenant/project';

-- View for per-model cost breakdown
CREATE OR REPLACE VIEW model_cost_breakdown AS
SELECT
    tenant_id,
    project_id,
    model,
    provider,
    COUNT(*) as total_calls,
    SUM(total_tokens) as total_tokens,
    SUM(total_cost_usd) as total_cost_usd,
    AVG(total_cost_usd) as avg_cost_per_call,
    AVG(latency_ms) as avg_latency_ms
FROM cost_logs
WHERE error = FALSE
GROUP BY tenant_id, project_id, model, provider;

COMMENT ON VIEW model_cost_breakdown IS 'Cost breakdown by model and provider for analysis';

-- ============================================================================
-- 5. SAMPLE DATA (for testing)
-- ============================================================================

-- Insert default budget for demo tenant (optional - comment out for production)
-- INSERT INTO budgets (tenant_id, project_id, daily_limit_usd, monthly_limit_usd, daily_tokens_limit, monthly_tokens_limit)
-- VALUES ('demo-tenant', 'default-project', 10.00, 100.00, 1000000, 10000000)
-- ON CONFLICT (tenant_id, project_id) DO NOTHING;

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================

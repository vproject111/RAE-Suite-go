-- ============================================================================
-- Event Triggers & Automations Schema
-- Version: 1.0
-- Description: Event-driven automation system with rules engine
-- ============================================================================

-- ============================================================================
-- Enums
-- ============================================================================

-- Event types that can trigger automations
CREATE TYPE event_type AS ENUM (
    'memory_created',
    'memory_updated',
    'memory_deleted',
    'memory_accessed',
    'reflection_generated',
    'semantic_node_created',
    'semantic_node_degraded',
    'search_executed',
    'query_analyzed',
    'drift_detected',
    'quality_degraded',
    'threshold_exceeded',
    'schedule_triggered'
);

-- Action types that can be executed
CREATE TYPE action_type AS ENUM (
    'send_notification',
    'send_email',
    'send_webhook',
    'create_memory',
    'update_memory',
    'delete_memory',
    'generate_reflection',
    'extract_semantics',
    'apply_decay',
    'reinforce_node',
    'create_snapshot',
    'run_evaluation',
    'execute_workflow',
    'trigger_another_rule'
);

-- Trigger status
CREATE TYPE trigger_status AS ENUM (
    'active',
    'inactive',
    'paused',
    'error'
);

-- Execution status
CREATE TYPE execution_status AS ENUM (
    'pending',
    'running',
    'completed',
    'failed',
    'skipped'
);

-- Condition operators
CREATE TYPE condition_operator AS ENUM (
    'equals',
    'not_equals',
    'greater_than',
    'less_than',
    'greater_equal',
    'less_equal',
    'contains',
    'not_contains',
    'in',
    'not_in',
    'matches_regex',
    'is_null',
    'is_not_null'
);


-- ============================================================================
-- Core Tables
-- ============================================================================

-- Workflows definition (created first because trigger_rules references it)
CREATE TABLE IF NOT EXISTS workflows (
    -- Identification
    workflow_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    -- Workflow metadata
    workflow_name VARCHAR(200) NOT NULL,
    description TEXT,

    -- Steps (stored as JSONB array of WorkflowStep)
    steps JSONB NOT NULL DEFAULT '[]'::JSONB,

    -- Execution mode
    stop_on_failure BOOLEAN DEFAULT TRUE,
    parallel_execution BOOLEAN DEFAULT FALSE,

    -- Metadata
    tags TEXT[] DEFAULT '{}',

    -- Ownership and timestamps
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT workflows_tenant_project_name_idx UNIQUE (tenant_id, project_id, workflow_name)
);

-- Indexes for workflows
CREATE INDEX idx_workflows_tenant_project ON workflows(tenant_id, project_id);
CREATE INDEX idx_workflows_tags ON workflows USING GIN(tags);


-- Trigger rules definition (created after workflows because it references workflows)
CREATE TABLE IF NOT EXISTS trigger_rules (
    -- Identification
    trigger_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,

    -- Rule metadata
    rule_name VARCHAR(200) NOT NULL,
    description TEXT,

    -- Trigger conditions (stored as JSONB for flexibility)
    event_types event_type[] NOT NULL,
    condition_group JSONB,  -- Stores ConditionGroup with nested logic

    -- Time-based conditions
    time_window_seconds INTEGER,
    cooldown_seconds INTEGER,

    -- Rate limiting
    max_executions_per_hour INTEGER,
    executions_this_hour INTEGER DEFAULT 0,
    hour_window_start TIMESTAMPTZ DEFAULT NOW(),

    -- Actions to execute (stored as JSONB array)
    actions JSONB NOT NULL DEFAULT '[]'::JSONB,
    workflow_id UUID REFERENCES workflows(workflow_id) ON DELETE SET NULL,

    -- Priority and control
    priority INTEGER NOT NULL DEFAULT 5 CHECK (priority BETWEEN 1 AND 10),
    status trigger_status NOT NULL DEFAULT 'active',
    is_enabled BOOLEAN DEFAULT TRUE,

    -- Execution tracking
    execution_count INTEGER DEFAULT 0,
    last_executed_at TIMESTAMPTZ,
    last_error_at TIMESTAMPTZ,
    last_error_message TEXT,

    -- Metadata
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}'::JSONB,

    -- Ownership and timestamps
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Indexes
    CONSTRAINT trigger_rules_tenant_project_idx UNIQUE (tenant_id, project_id, rule_name)
);

-- Indexes for trigger_rules
CREATE INDEX idx_trigger_rules_tenant_project ON trigger_rules(tenant_id, project_id);
CREATE INDEX idx_trigger_rules_status ON trigger_rules(status) WHERE status = 'active';
CREATE INDEX idx_trigger_rules_event_types ON trigger_rules USING GIN(event_types);
CREATE INDEX idx_trigger_rules_priority ON trigger_rules(priority DESC);
CREATE INDEX idx_trigger_rules_tags ON trigger_rules USING GIN(tags);


-- Action execution history
CREATE TABLE IF NOT EXISTS action_executions (
    -- Identification
    execution_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger_id UUID NOT NULL REFERENCES trigger_rules(trigger_id) ON DELETE CASCADE,

    -- Event that triggered this
    event_id UUID NOT NULL,
    event_type event_type NOT NULL,
    event_payload JSONB DEFAULT '{}'::JSONB,

    -- Action details
    action_type action_type NOT NULL,
    action_config JSONB DEFAULT '{}'::JSONB,

    -- Execution
    status execution_status NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,

    -- Results
    success BOOLEAN DEFAULT FALSE,
    result JSONB,
    error_message TEXT,
    error_traceback TEXT,

    -- Retry info
    attempt_number INTEGER NOT NULL DEFAULT 1 CHECK (attempt_number >= 1),
    max_attempts INTEGER NOT NULL DEFAULT 1 CHECK (max_attempts >= 1),

    -- Context
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL
);

-- Indexes for action_executions
CREATE INDEX idx_action_executions_trigger ON action_executions(trigger_id);
CREATE INDEX idx_action_executions_event ON action_executions(event_id);
CREATE INDEX idx_action_executions_status ON action_executions(status);
CREATE INDEX idx_action_executions_tenant_project ON action_executions(tenant_id, project_id);
CREATE INDEX idx_action_executions_started_at ON action_executions(started_at DESC);


-- Workflow execution history
CREATE TABLE IF NOT EXISTS workflow_executions (
    -- Identification
    execution_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(workflow_id) ON DELETE CASCADE,
    trigger_id UUID REFERENCES trigger_rules(trigger_id) ON DELETE SET NULL,

    -- Event context
    event_id UUID,
    event_type event_type,

    -- Execution
    status execution_status NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,

    -- Results
    success BOOLEAN DEFAULT FALSE,
    steps_completed INTEGER DEFAULT 0,
    steps_failed INTEGER DEFAULT 0,
    steps_skipped INTEGER DEFAULT 0,

    error_message TEXT,

    -- Context
    tenant_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL
);

-- Indexes for workflow_executions
CREATE INDEX idx_workflow_executions_workflow ON workflow_executions(workflow_id);
CREATE INDEX idx_workflow_executions_trigger ON workflow_executions(trigger_id);
CREATE INDEX idx_workflow_executions_status ON workflow_executions(status);
CREATE INDEX idx_workflow_executions_started_at ON workflow_executions(started_at DESC);


-- Workflow step execution tracking
CREATE TABLE IF NOT EXISTS workflow_step_executions (
    -- Identification
    step_execution_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_execution_id UUID NOT NULL REFERENCES workflow_executions(execution_id) ON DELETE CASCADE,

    -- Step details
    step_id VARCHAR(100) NOT NULL,
    step_name VARCHAR(200) NOT NULL,
    step_order INTEGER NOT NULL,

    -- Action
    action_type action_type NOT NULL,
    action_config JSONB DEFAULT '{}'::JSONB,

    -- Execution
    status execution_status NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,

    -- Results
    success BOOLEAN DEFAULT FALSE,
    result JSONB,
    error_message TEXT,

    -- Dependencies
    depends_on TEXT[] DEFAULT '{}',
    dependency_met BOOLEAN DEFAULT TRUE
);

-- Indexes for workflow_step_executions
CREATE INDEX idx_workflow_step_executions_workflow ON workflow_step_executions(workflow_execution_id);
CREATE INDEX idx_workflow_step_executions_status ON workflow_step_executions(status);


-- ============================================================================
-- Helper Functions
-- ============================================================================

-- Function to record trigger execution
CREATE OR REPLACE FUNCTION record_trigger_execution(
    p_trigger_id UUID,
    p_success BOOLEAN DEFAULT TRUE
) RETURNS VOID AS $$
BEGIN
    UPDATE trigger_rules
    SET execution_count = execution_count + 1,
        executions_this_hour = executions_this_hour + 1,
        last_executed_at = NOW(),
        updated_at = NOW()
    WHERE trigger_id = p_trigger_id;

    -- If failed, record error timestamp
    IF NOT p_success THEN
        UPDATE trigger_rules
        SET last_error_at = NOW(),
            status = 'error'
        WHERE trigger_id = p_trigger_id;
    END IF;
END;
$$ LANGUAGE plpgsql;


-- Function to reset hourly execution counter
CREATE OR REPLACE FUNCTION reset_trigger_rate_limit(
    p_trigger_id UUID
) RETURNS VOID AS $$
BEGIN
    UPDATE trigger_rules
    SET executions_this_hour = 0,
        hour_window_start = NOW(),
        updated_at = NOW()
    WHERE trigger_id = p_trigger_id;
END;
$$ LANGUAGE plpgsql;


-- Function to check if trigger can execute (rate limiting + cooldown)
CREATE OR REPLACE FUNCTION can_trigger_execute(
    p_trigger_id UUID
) RETURNS BOOLEAN AS $$
DECLARE
    v_trigger RECORD;
    v_time_since_last_execution INTERVAL;
    v_time_since_window_start INTERVAL;
BEGIN
    SELECT * INTO v_trigger
    FROM trigger_rules
    WHERE trigger_id = p_trigger_id;

    -- Check if trigger exists and is enabled
    IF NOT FOUND OR NOT v_trigger.is_enabled OR v_trigger.status != 'active' THEN
        RETURN FALSE;
    END IF;

    -- Check cooldown
    IF v_trigger.cooldown_seconds IS NOT NULL AND v_trigger.last_executed_at IS NOT NULL THEN
        v_time_since_last_execution := NOW() - v_trigger.last_executed_at;
        IF EXTRACT(EPOCH FROM v_time_since_last_execution) < v_trigger.cooldown_seconds THEN
            RETURN FALSE;
        END IF;
    END IF;

    -- Check rate limit
    IF v_trigger.max_executions_per_hour IS NOT NULL THEN
        v_time_since_window_start := NOW() - v_trigger.hour_window_start;

        -- Reset counter if new hour
        IF EXTRACT(EPOCH FROM v_time_since_window_start) >= 3600 THEN
            PERFORM reset_trigger_rate_limit(p_trigger_id);
            RETURN TRUE;
        END IF;

        -- Check if within limit
        IF v_trigger.executions_this_hour >= v_trigger.max_executions_per_hour THEN
            RETURN FALSE;
        END IF;
    END IF;

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;


-- Function to activate trigger
CREATE OR REPLACE FUNCTION activate_trigger(
    p_trigger_id UUID
) RETURNS VOID AS $$
BEGIN
    UPDATE trigger_rules
    SET status = 'active',
        is_enabled = TRUE,
        updated_at = NOW()
    WHERE trigger_id = p_trigger_id;
END;
$$ LANGUAGE plpgsql;


-- Function to deactivate trigger
CREATE OR REPLACE FUNCTION deactivate_trigger(
    p_trigger_id UUID
) RETURNS VOID AS $$
BEGIN
    UPDATE trigger_rules
    SET status = 'inactive',
        is_enabled = FALSE,
        updated_at = NOW()
    WHERE trigger_id = p_trigger_id;
END;
$$ LANGUAGE plpgsql;


-- Function to pause trigger (temporary disable)
CREATE OR REPLACE FUNCTION pause_trigger(
    p_trigger_id UUID
) RETURNS VOID AS $$
BEGIN
    UPDATE trigger_rules
    SET status = 'paused',
        updated_at = NOW()
    WHERE trigger_id = p_trigger_id;
END;
$$ LANGUAGE plpgsql;


-- Function to get trigger execution statistics
CREATE OR REPLACE FUNCTION get_trigger_execution_stats(
    p_trigger_id UUID,
    p_period_days INTEGER DEFAULT 7
) RETURNS TABLE(
    total_executions BIGINT,
    successful_executions BIGINT,
    failed_executions BIGINT,
    avg_duration_ms NUMERIC,
    min_duration_ms INTEGER,
    max_duration_ms INTEGER,
    last_execution_at TIMESTAMPTZ,
    last_success_at TIMESTAMPTZ,
    last_failure_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::BIGINT AS total_executions,
        COUNT(*) FILTER (WHERE success = TRUE)::BIGINT AS successful_executions,
        COUNT(*) FILTER (WHERE success = FALSE)::BIGINT AS failed_executions,
        AVG(duration_ms)::NUMERIC AS avg_duration_ms,
        MIN(duration_ms)::INTEGER AS min_duration_ms,
        MAX(duration_ms)::INTEGER AS max_duration_ms,
        MAX(completed_at) AS last_execution_at,
        MAX(completed_at) FILTER (WHERE success = TRUE) AS last_success_at,
        MAX(completed_at) FILTER (WHERE success = FALSE) AS last_failure_at
    FROM action_executions
    WHERE trigger_id = p_trigger_id
        AND started_at >= NOW() - (p_period_days || ' days')::INTERVAL;
END;
$$ LANGUAGE plpgsql;


-- Function to get workflow execution statistics
CREATE OR REPLACE FUNCTION get_workflow_execution_stats(
    p_workflow_id UUID,
    p_period_days INTEGER DEFAULT 7
) RETURNS TABLE(
    total_executions BIGINT,
    successful_executions BIGINT,
    failed_executions BIGINT,
    avg_duration_ms NUMERIC,
    avg_steps_completed NUMERIC,
    last_execution_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::BIGINT AS total_executions,
        COUNT(*) FILTER (WHERE success = TRUE)::BIGINT AS successful_executions,
        COUNT(*) FILTER (WHERE success = FALSE)::BIGINT AS failed_executions,
        AVG(duration_ms)::NUMERIC AS avg_duration_ms,
        AVG(steps_completed)::NUMERIC AS avg_steps_completed,
        MAX(completed_at) AS last_execution_at
    FROM workflow_executions
    WHERE workflow_id = p_workflow_id
        AND started_at >= NOW() - (p_period_days || ' days')::INTERVAL;
END;
$$ LANGUAGE plpgsql;


-- Function to cleanup old execution records
CREATE OR REPLACE FUNCTION cleanup_old_executions(
    p_retention_days INTEGER DEFAULT 90
) RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    -- Delete old action executions
    WITH deleted AS (
        DELETE FROM action_executions
        WHERE started_at < NOW() - (p_retention_days || ' days')::INTERVAL
        RETURNING 1
    )
    SELECT COUNT(*) INTO v_deleted_count FROM deleted;

    -- Delete old workflow executions (cascades to step executions)
    DELETE FROM workflow_executions
    WHERE started_at < NOW() - (p_retention_days || ' days')::INTERVAL;

    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;


-- ============================================================================
-- Views
-- ============================================================================

-- View for trigger statistics
CREATE OR REPLACE VIEW trigger_statistics AS
SELECT
    tr.trigger_id,
    tr.tenant_id,
    tr.project_id,
    tr.rule_name,
    tr.status,
    tr.is_enabled,
    tr.priority,
    tr.execution_count,
    tr.last_executed_at,

    -- Recent execution stats (last 24 hours)
    COUNT(ae.execution_id) FILTER (
        WHERE ae.started_at >= NOW() - INTERVAL '24 hours'
    ) AS executions_last_24h,

    COUNT(ae.execution_id) FILTER (
        WHERE ae.started_at >= NOW() - INTERVAL '24 hours'
        AND ae.success = TRUE
    ) AS successful_executions_last_24h,

    COUNT(ae.execution_id) FILTER (
        WHERE ae.started_at >= NOW() - INTERVAL '24 hours'
        AND ae.success = FALSE
    ) AS failed_executions_last_24h,

    -- Average duration (last 100 executions)
    (
        SELECT AVG(duration_ms)
        FROM (
            SELECT duration_ms
            FROM action_executions
            WHERE trigger_id = tr.trigger_id
            ORDER BY started_at DESC
            LIMIT 100
        ) recent
    ) AS avg_duration_ms_recent,

    tr.created_at,
    tr.updated_at
FROM trigger_rules tr
LEFT JOIN action_executions ae ON ae.trigger_id = tr.trigger_id
GROUP BY tr.trigger_id;


-- View for workflow statistics
CREATE OR REPLACE VIEW workflow_statistics AS
SELECT
    w.workflow_id,
    w.tenant_id,
    w.project_id,
    w.workflow_name,

    -- Execution stats (all time)
    COUNT(we.execution_id) AS total_executions,
    COUNT(we.execution_id) FILTER (WHERE we.success = TRUE) AS successful_executions,
    COUNT(we.execution_id) FILTER (WHERE we.success = FALSE) AS failed_executions,

    -- Recent stats (last 7 days)
    COUNT(we.execution_id) FILTER (
        WHERE we.started_at >= NOW() - INTERVAL '7 days'
    ) AS executions_last_7d,

    -- Performance
    AVG(we.duration_ms) AS avg_duration_ms,
    AVG(we.steps_completed) AS avg_steps_completed,

    -- Timestamps
    MAX(we.completed_at) AS last_execution_at,
    w.created_at,
    w.updated_at
FROM workflows w
LEFT JOIN workflow_executions we ON we.workflow_id = w.workflow_id
GROUP BY w.workflow_id;


-- View for recent execution summary
CREATE OR REPLACE VIEW recent_executions_summary AS
SELECT
    ae.execution_id,
    ae.trigger_id,
    tr.rule_name,
    ae.event_type,
    ae.action_type,
    ae.status,
    ae.success,
    ae.started_at,
    ae.completed_at,
    ae.duration_ms,
    ae.error_message,
    ae.tenant_id,
    ae.project_id
FROM action_executions ae
JOIN trigger_rules tr ON tr.trigger_id = ae.trigger_id
WHERE ae.started_at >= NOW() - INTERVAL '7 days'
ORDER BY ae.started_at DESC;


-- ============================================================================
-- Database Triggers (for automatic updates)
-- ============================================================================

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_rules_updated_at
    BEFORE UPDATE ON trigger_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER workflows_updated_at
    BEFORE UPDATE ON workflows
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


-- Trigger to automatically calculate execution duration
CREATE OR REPLACE FUNCTION calculate_execution_duration()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.completed_at IS NOT NULL AND NEW.started_at IS NOT NULL THEN
        NEW.duration_ms := EXTRACT(EPOCH FROM (NEW.completed_at - NEW.started_at)) * 1000;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER action_executions_duration
    BEFORE UPDATE OF completed_at ON action_executions
    FOR EACH ROW
    EXECUTE FUNCTION calculate_execution_duration();

CREATE TRIGGER workflow_executions_duration
    BEFORE UPDATE OF completed_at ON workflow_executions
    FOR EACH ROW
    EXECUTE FUNCTION calculate_execution_duration();

CREATE TRIGGER workflow_step_executions_duration
    BEFORE UPDATE OF completed_at ON workflow_step_executions
    FOR EACH ROW
    EXECUTE FUNCTION calculate_execution_duration();


-- ============================================================================
-- Row-Level Security (RLS)
-- ============================================================================

-- Enable RLS on all tables
ALTER TABLE trigger_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE action_executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflows ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_step_executions ENABLE ROW LEVEL SECURITY;

-- Policies for trigger_rules
CREATE POLICY trigger_rules_tenant_isolation ON trigger_rules
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::VARCHAR);

-- Policies for action_executions
CREATE POLICY action_executions_tenant_isolation ON action_executions
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::VARCHAR);

-- Policies for workflows
CREATE POLICY workflows_tenant_isolation ON workflows
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::VARCHAR);

-- Policies for workflow_executions
CREATE POLICY workflow_executions_tenant_isolation ON workflow_executions
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::VARCHAR);

-- Policies for workflow_step_executions (through workflow_executions)
CREATE POLICY workflow_step_executions_tenant_isolation ON workflow_step_executions
    FOR ALL
    USING (
        workflow_execution_id IN (
            SELECT execution_id
            FROM workflow_executions
            WHERE tenant_id = current_setting('app.current_tenant_id', TRUE)::VARCHAR
        )
    );


-- ============================================================================
-- Indexes for Performance
-- ============================================================================

-- Composite indexes for common queries
CREATE INDEX idx_trigger_rules_tenant_project_status ON trigger_rules(tenant_id, project_id, status)
    WHERE status = 'active' AND is_enabled = TRUE;

CREATE INDEX idx_action_executions_trigger_started ON action_executions(trigger_id, started_at DESC);

CREATE INDEX idx_action_executions_tenant_project_started ON action_executions(tenant_id, project_id, started_at DESC);

CREATE INDEX idx_workflow_executions_workflow_started ON workflow_executions(workflow_id, started_at DESC);


-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON TABLE trigger_rules IS 'Event trigger rules with conditions and actions';
COMMENT ON TABLE action_executions IS 'History of action executions triggered by events';
COMMENT ON TABLE workflows IS 'Workflow definitions with multiple steps';
COMMENT ON TABLE workflow_executions IS 'Workflow execution history';
COMMENT ON TABLE workflow_step_executions IS 'Individual workflow step execution tracking';

COMMENT ON COLUMN trigger_rules.condition_group IS 'Nested condition groups with AND/OR logic (JSONB)';
COMMENT ON COLUMN trigger_rules.actions IS 'Array of action configurations (JSONB)';
COMMENT ON COLUMN trigger_rules.cooldown_seconds IS 'Minimum seconds between executions';
COMMENT ON COLUMN trigger_rules.max_executions_per_hour IS 'Rate limit per hour';

COMMENT ON FUNCTION can_trigger_execute IS 'Check if trigger can execute (rate limit + cooldown)';
COMMENT ON FUNCTION record_trigger_execution IS 'Record trigger execution and update counters';
COMMENT ON FUNCTION get_trigger_execution_stats IS 'Get execution statistics for a trigger';
COMMENT ON FUNCTION cleanup_old_executions IS 'Cleanup execution records older than retention period';


-- ============================================================================
-- Initial Data / Examples
-- ============================================================================

-- Example: Quality degradation alert trigger
-- (Commented out - would be created via API)
/*
INSERT INTO trigger_rules (
    tenant_id, project_id, rule_name, description,
    event_types, condition_group, actions,
    priority, created_by
) VALUES (
    'example-tenant', 'example-project',
    'Quality Degradation Alert',
    'Send notification when quality drops below threshold',
    ARRAY['quality_degraded']::event_type[],
    '{
        "operator": "AND",
        "conditions": [
            {"field": "payload.quality_score", "operator": "less_than", "value": 0.7}
        ]
    }'::JSONB,
    '[
        {
            "action_type": "send_notification",
            "config": {"channel": "email", "template": "quality_alert"}
        }
    ]'::JSONB,
    9,
    'system'
);
*/

-- ============================================================================
-- Maintenance
-- ============================================================================

-- Create scheduled job to cleanup old executions (if pg_cron is available)
-- SELECT cron.schedule('cleanup-old-executions', '0 2 * * *', 'SELECT cleanup_old_executions(90)');

-- Create scheduled job to reset rate limits for stale triggers
-- SELECT cron.schedule('reset-stale-rate-limits', '*/5 * * * *',
--     'UPDATE trigger_rules SET executions_this_hour = 0, hour_window_start = NOW()
--      WHERE EXTRACT(EPOCH FROM (NOW() - hour_window_start)) >= 3600');

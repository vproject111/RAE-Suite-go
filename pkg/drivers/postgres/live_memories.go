package postgres

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

// NewPGWireClientFromEnv constructs a PGWireClient using POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER,
// POSTGRES_PASSWORD, and POSTGRES_DB environment variables (defaulting to 127.0.0.1:5432, rae, rae_password, rae).
func NewPGWireClientFromEnv() *PGWireClient {
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "rae"
	}
	pwd := os.Getenv("POSTGRES_PASSWORD")
	if pwd == "" {
		pwd = "rae_password"
	}
	db := os.Getenv("POSTGRES_DB")
	if db == "" {
		db = "rae"
	}
	return NewPGWireClient(fmt.Sprintf("%s:%s", host, port), user, pwd, db)
}

func sqlEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func sqlQuote(s string) string {
	return "'" + sqlEscape(s) + "'"
}

// UpsertMemoryRow inserts or updates a MemoryItem in the live PostgreSQL `memories` table.
func (c *PGWireClient) UpsertMemoryRow(m *models.MemoryItem) error {
	if m == nil {
		return nil
	}
	tenantID := m.TenantID
	if len(tenantID) != 36 {
		tenantID = "f51d8b92-2fb1-524c-86e4-c6f8f6f59872"
	}
	id := m.ID
	if len(id) != 36 {
		id = models.GenerateUUIDv7()
		m.ID = id
	}
	agentID := m.AgentID
	if agentID == "" {
		agentID = "default"
	}
	infoClass := string(m.InfoClass)
	if infoClass == "" {
		infoClass = "internal"
	}
	layer := string(m.Layer)
	if layer == "" {
		layer = "working"
	}
	vals := strings.Join([]string{
		sqlQuote(id),
		sqlQuote(tenantID),
		sqlQuote(m.Content),
		strconv.FormatFloat(m.Importance, 'f', 6, 64),
		sqlQuote(layer),
		sqlQuote(m.Project),
		sqlQuote(agentID),
		sqlQuote(m.SessionID),
		sqlQuote(m.Source),
		sqlQuote(infoClass),
		strconv.Itoa(m.UsageCount),
		strconv.FormatFloat(m.Strength, 'f', 6, 64),
		sqlQuote(m.ContentHash),
	}, ", ")
	sql := strings.Join([]string{
		"SET app.current_user_role = 'admin'; SET app.current_tenant_id =",
		sqlQuote(tenantID),
		"; INSERT INTO memories (id, tenant_id, content, importance, layer, project, agent_id, session_id, source, info_class, usage_count, strength, content_hash) VALUES (",
		vals,
		") ON CONFLICT (id) DO UPDATE SET content = EXCLUDED.content, importance = EXCLUDED.importance, layer = EXCLUDED.layer, usage_count = EXCLUDED.usage_count, last_accessed_at = now();",
	}, " ")
	_, err := c.Query(sql)
	return err
}

// DeleteMemoryRow deletes a memory row by ID from the live PostgreSQL `memories` table.
func (c *PGWireClient) DeleteMemoryRow(id string) error {
	quotedID := sqlQuote(id)
	stmt := strings.Join([]string{"SET app.current_user_role = 'admin'; DELETE FROM memories WHERE id =", quotedID, ";"}, " ")
	_, err := c.Query(stmt)
	return err
}

// LoadAllMemories reads up to `limit` memories from the live PostgreSQL `memories` table.
func (c *PGWireClient) LoadAllMemories(limit int) ([]models.MemoryItem, error) {
	if limit <= 0 {
		limit = 5000
	}
	limitStr := strconv.Itoa(limit)
	sql := strings.Join([]string{
		"SET app.current_user_role = 'admin';",
		"SELECT id::text, tenant_id::text, content, COALESCE(layer, 'working') AS layer,",
		"COALESCE(project, 'default') AS project, COALESCE(importance, 0.5)::text AS importance,",
		"COALESCE(agent_id, 'default') AS agent_id, COALESCE(session_id, '') AS session_id,",
		"COALESCE(info_class, 'internal') AS info_class, COALESCE(usage_count, 0)::text AS usage_count",
		"FROM memories ORDER BY created_at DESC LIMIT",
		limitStr,
		";",
	}, " ")
	rows, err := c.Query(sql)
	if err != nil {
		return nil, err
	}
	items := make([]models.MemoryItem, 0, len(rows))
	now := time.Now().UTC()
	for _, r := range rows {
		id, _ := r["id"].(string)
		tenantID, _ := r["tenant_id"].(string)
		content, _ := r["content"].(string)
		layerStr, _ := r["layer"].(string)
		project, _ := r["project"].(string)
		impStr, _ := r["importance"].(string)
		agentID, _ := r["agent_id"].(string)
		sessionID, _ := r["session_id"].(string)
		infoStr, _ := r["info_class"].(string)
		usageStr, _ := r["usage_count"].(string)

		imp, err := strconv.ParseFloat(impStr, 64)
		if err != nil || imp <= 0 {
			imp = 0.5
		}
		usage, _ := strconv.Atoi(usageStr)

		items = append(items, models.MemoryItem{
			ID:             id,
			TenantID:       tenantID,
			Content:        content,
			Layer:          models.NormalizeLayer(layerStr),
			MemoryType:     models.TypeText,
			InfoClass:      models.InformationClass(strings.ToLower(infoStr)),
			Project:        project,
			AgentID:        agentID,
			SessionID:      sessionID,
			Importance:     imp,
			Strength:       1.0,
			UsageCount:     usage,
			CreatedAt:      now,
			LastAccessedAt: now,
		})
	}
	return items, nil
}

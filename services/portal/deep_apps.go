package portal

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raesdk"
)

// PDKnowledgeHubApp implements packages/rae-agentic-memory/rae-portal/apps/pd_knowledge_hub.py.
type PDKnowledgeHubApp struct {
	mu     sync.RWMutex
	chunks map[string][]string
}

func NewPDKnowledgeHubApp() *PDKnowledgeHubApp {
	return &PDKnowledgeHubApp{
		chunks: make(map[string][]string),
	}
}

func (app *PDKnowledgeHubApp) ExtractDocumentText(raw []byte) string {
	return strings.TrimSpace(string(raw))
}

func (app *PDKnowledgeHubApp) ChunkText(text string, chunkSize int) []string {
	if chunkSize <= 0 {
		chunkSize = 200
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var out []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len()+len(w)+1 > chunkSize && cur.Len() > 0 {
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		}
		if cur.Len() > 0 {
			cur.WriteString(" ")
		}
		cur.WriteString(w)
	}
	if cur.Len() > 0 {
		out = append(out, strings.TrimSpace(cur.String()))
	}
	return out
}

func (app *PDKnowledgeHubApp) HandleFileUpload(docID string, raw []byte) map[string]interface{} {
	text := app.ExtractDocumentText(raw)
	chunks := app.ChunkText(text, 200)
	app.mu.Lock()
	app.chunks[docID] = chunks
	app.mu.Unlock()
	return map[string]interface{}{
		"doc_id":      docID,
		"chunk_count": len(chunks),
		"status":      "INGESTED",
	}
}

func (app *PDKnowledgeHubApp) DeleteDocumentChunks(docID string) bool {
	app.mu.Lock()
	defer app.mu.Unlock()
	_, ok := app.chunks[docID]
	delete(app.chunks, docID)
	return ok
}

func (app *PDKnowledgeHubApp) SearchISAPAndWeb(query string) []map[string]interface{} {
	q := strings.ToLower(query)
	app.mu.RLock()
	defer app.mu.RUnlock()
	results := make([]map[string]interface{}, 0)
	for docID, chList := range app.chunks {
		for idx, ch := range chList {
			if q == "" || strings.Contains(strings.ToLower(ch), q) {
				results = append(results, map[string]interface{}{
					"doc_id":      docID,
					"chunk_index": idx,
					"content":     ch,
					"source":      "ISAP_KNOWLEDGE_HUB",
				})
			}
		}
	}
	return results
}

// CentralAuditorApp implements packages/rae-agentic-memory/rae-portal/apps/central_auditor.py.
type CentralAuditorApp struct {
	SecretKey string
}

func NewCentralAuditorApp(secretKey string) *CentralAuditorApp {
	if secretKey == "" {
		secretKey = "rae-sentinel-hmac-key"
	}
	return &CentralAuditorApp{SecretKey: secretKey}
}

func (c *CentralAuditorApp) SignEvent(payload string) string {
	mac := hmac.New(sha256.New, []byte(c.SecretKey))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *CentralAuditorApp) VerifyEventSignature(payload, sig string) bool {
	expected := c.SignEvent(payload)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func (c *CentralAuditorApp) DetectSecrets(text string) []string {
	var found []string
	lower := strings.ToLower(text)
	if strings.Contains(lower, "sk-") {
		found = append(found, "OPENAI_KEY_PATTERN")
	}
	if strings.Contains(lower, "ghp_") {
		found = append(found, "GITHUB_PAT_PATTERN")
	}
	if strings.Contains(lower, "password=") {
		found = append(found, "HARDCODED_PASSWORD")
	}
	return found
}

func (c *CentralAuditorApp) TriggerSentinelAudit(code string) map[string]interface{} {
	secrets := c.DetectSecrets(code)
	sig := c.SignEvent(code)
	return map[string]interface{}{
		"passed":        len(secrets) == 0,
		"secrets_found": secrets,
		"signature":     sig,
	}
}

func (c *CentralAuditorApp) Render() string {
	return "<section id='central-auditor'>Central Sentinel Auditor Active</section>"
}

// PhoenixRepairApp implements packages/rae-agentic-memory/rae-portal/apps/phoenix_repair.py.
type PhoenixRepairApp struct {
	mu            sync.Mutex
	activeIncident string
	timerTicks    int
}

func NewPhoenixRepairApp() *PhoenixRepairApp {
	return &PhoenixRepairApp{}
}

func (p *PhoenixRepairApp) TriggerSimulatedIncident(incident string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.activeIncident = incident
	return "INCIDENT_ACTIVE:" + incident
}

func (p *PhoenixRepairApp) TickIncidentTimer() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.timerTicks++
	return p.timerTicks
}

func (p *PhoenixRepairApp) SimulateRepair(code string) map[string]interface{} {
	return map[string]interface{}{
		"original": code,
		"patched":  "// Repaired by Phoenix\n" + code,
		"status":   "SIMULATED_OK",
	}
}

func (p *PhoenixRepairApp) ApplyRepair(code string) map[string]interface{} {
	p.mu.Lock()
	p.activeIncident = ""
	p.mu.Unlock()
	return map[string]interface{}{
		"applied": true,
		"code":    "// Repaired by Phoenix\n" + code,
	}
}

func (p *PhoenixRepairApp) Render() string {
	return "<section id='phoenix-repair'>Phoenix Closed-Loop Repair Console</section>"
}

// MissionControlApp implements packages/rae-agentic-memory/rae-portal/apps/mission_control.py.
type MissionControlApp struct{}

func NewMissionControlApp() *MissionControlApp { return &MissionControlApp{} }

func (m *MissionControlApp) FetchData() map[string]interface{} {
	return map[string]interface{}{
		"nodes_online": 3,
		"mesh_status":  "SYNCED",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}
}

func (m *MissionControlApp) RenderContent() string {
	data := m.FetchData()
	return fmt.Sprintf("Mission Control: nodes=%v mesh=%v", data["nodes_online"], data["mesh_status"])
}

func (m *MissionControlApp) Render() string {
	return "<main id='mission-control'>" + m.RenderContent() + "</main>"
}

// RAESuitePortal implements packages/rae-agentic-memory/rae-portal/main.py.
type RAESuitePortal struct {
	CurrentPage   string
	KnowledgeHub  *PDKnowledgeHubApp
	Auditor       *CentralAuditorApp
	PhoenixApp    *PhoenixRepairApp
	MissionCtrl   *MissionControlApp
	TrayApp       *raesdk.DesktopTrayApp
	NiceGUIDash   *raesdk.NiceGUIDashboardController
}

func NewRAESuitePortal() *RAESuitePortal {
	return &RAESuitePortal{
		CurrentPage:  "mission_control",
		KnowledgeHub: NewPDKnowledgeHubApp(),
		Auditor:      NewCentralAuditorApp(""),
		PhoenixApp:   NewPhoenixRepairApp(),
		MissionCtrl:  NewMissionControlApp(),
		TrayApp:      raesdk.NewDesktopTrayApp("http://127.0.0.1:8000"),
		NiceGUIDash:  raesdk.NewNiceGUIDashboardController(),
	}
}

func (p *RAESuitePortal) DetectSystem() map[string]string {
	return map[string]string{"runtime": "go", "suite": "RAE-Suite-go v5.2.0"}
}

func (p *RAESuitePortal) SetPage(page string) {
	p.CurrentPage = page
}

func (p *RAESuitePortal) OpenInspector(memoryID string) map[string]interface{} {
	return map[string]interface{}{"inspector_open": true, "memory_id": memoryID}
}

func (p *RAESuitePortal) StartWebSocketClient() bool {
	return true
}

func (p *RAESuitePortal) Render() string {
	switch p.CurrentPage {
	case "auditor":
		return p.Auditor.Render()
	case "phoenix":
		return p.PhoenixApp.Render()
	case "nicegui":
		if p.NiceGUIDash != nil {
			return p.NiceGUIDash.RenderLayoutHTML()
		}
		return p.MissionCtrl.Render()
	default:
		return p.MissionCtrl.Render()
	}
}


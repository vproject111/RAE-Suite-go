package raesdk

import (
	"fmt"
	"sync"
	"time"
)

// DesktopTrayApp implements packages/rae-agentic-memory/rae-windows (Windows/Linux System Tray & Desktop Control Agent).
type DesktopTrayApp struct {
	mu           sync.RWMutex
	AppName      string
	Running      bool
	TrayIcon     string
	APIEndpoint  string
	MenuActions  []string
	Notifications []map[string]interface{}
}

func NewDesktopTrayApp(apiEndpoint string) *DesktopTrayApp {
	if apiEndpoint == "" {
		apiEndpoint = "http://127.0.0.1:8000"
	}
	return &DesktopTrayApp{
		AppName:     "RAE Desktop Tray Controller",
		Running:     true,
		TrayIcon:    "rae-oracle-active",
		APIEndpoint: apiEndpoint,
		MenuActions: []string{
			"Open RAE Portal Dashboard",
			"Sync RAE-Mesh Now",
			"Run 9/5 Benchmarks",
			"Toggle Lite/Full Mode",
			"Exit Tray Agent",
		},
		Notifications: make([]map[string]interface{}, 0),
	}
}

func (t *DesktopTrayApp) Notify(title, message string) map[string]interface{} {
	t.mu.Lock()
	defer t.mu.Unlock()
	ev := map[string]interface{}{
		"title":     title,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	t.Notifications = append(t.Notifications, ev)
	return ev
}

func (t *DesktopTrayApp) TriggerMenuAction(action string) map[string]interface{} {
	t.mu.Lock()
	defer t.mu.Unlock()
	if action == "Exit Tray Agent" {
		t.Running = false
	}
	return map[string]interface{}{
		"action":       action,
		"running":      t.Running,
		"api_endpoint": t.APIEndpoint,
		"executed_at":  time.Now().UTC().Format(time.RFC3339),
	}
}

func (t *DesktopTrayApp) GetTrayStatus() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return map[string]interface{}{
		"app_name":            t.AppName,
		"running":             t.Running,
		"tray_icon":           t.TrayIcon,
		"api_endpoint":        t.APIEndpoint,
		"menu_actions":        t.MenuActions,
		"notifications_count": len(t.Notifications),
	}
}

// NiceGUIDashboardController implements tools/memory-dashboard-nicegui & tools/memory-dashboard (Streamlit/NiceGUI state engine).
type NiceGUIDashboardController struct {
	mu            sync.RWMutex
	ActiveTab     string
	SelectedLayer string
	SearchQuery   string
	AutoRefresh   bool
	Widgets       map[string]interface{}
}

func NewNiceGUIDashboardController() *NiceGUIDashboardController {
	return &NiceGUIDashboardController{
		ActiveTab:     "memories_explorer",
		SelectedLayer: "all",
		AutoRefresh:   true,
		Widgets: map[string]interface{}{
			"kpi_mrr":          1.0,
			"kpi_hit_rate":     1.0,
			"graph_visualizer": "active",
			"timeline_slider":  "24h",
		},
	}
}

func (d *NiceGUIDashboardController) SelectTab(tab string) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if tab != "" {
		d.ActiveTab = tab
	}
	return d.ActiveTab
}

func (d *NiceGUIDashboardController) UpdateWidget(id string, value interface{}) map[string]interface{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Widgets[id] = value
	return map[string]interface{}{
		"widget_id": id,
		"value":     value,
		"status":    "UPDATED",
	}
}

func (d *NiceGUIDashboardController) RenderLayoutHTML() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return fmt.Sprintf(`<div id="rae-nicegui-root" class="app-root" data-tab="%s" data-layer="%s"><h1>RAE Interactive Dashboard</h1><section class="widgets-grid">Widgets: %d</section></div>`,
		d.ActiveTab, d.SelectedLayer, len(d.Widgets))
}

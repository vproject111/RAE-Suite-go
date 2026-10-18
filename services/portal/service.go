package portal

import (
	"encoding/json"
	"net/http"
)

// AllPortalViews returns the 14 canonical RAE-Portal views (including RaeCrl).
func AllPortalViews() []string {
	return []string{
		"MissionControl",
		"SearchConsole",
		"DecayRetention",
		"EvolutionLab",
		"PhoenixRepair",
		"HiveSandbox",
		"OpenClawEscalation",
		"RaeCrl",
		"MeshFederation",
		"CentralAuditor",
		"PDKnowledgeHub",
		"ComputeFabric",
		"HITLApproval",
		"ISOCompliance",
	}
}

// Service implements Module 8: RAE-Portal (14 Views, PWA Offline Read-Only, WebSockets & HITL).
type Service struct{}

// NewService constructs a new RAE-Portal service.
func NewService() *Service {
	return &Service{}
}

// RegisterRoutes attaches RAE-Portal PWA, manifest, service worker, and dashboard routes (:8082).
func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"views":  len(AllPortalViews()),
		})
	})

	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":             "RAE Suite Command Center",
			"short_name":       "RAE Portal",
			"start_url":        "/",
			"display":          "standalone",
			"background_color": "#0f172a",
			"theme_color":      "#1e293b",
			"icons":            []string{},
		})
	})

	mux.HandleFunc("/sw.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = w.Write([]byte(`const CACHE_NAME = 'rae-portal-v4';
self.addEventListener('install', (e) => self.skipWaiting());
self.addEventListener('activate', (e) => e.waitUntil(self.clients.claim()));
self.addEventListener('fetch', (e) => {
  e.respondWith(fetch(e.request).catch(() => caches.match(e.request)));
});
`))
	})

	mux.HandleFunc("/v2/portal/views", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"views": AllPortalViews(),
		})
	})
}

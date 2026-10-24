package ops

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type ctxKey string

const (
	ctxKeyTraceID   ctxKey = "rae_trace_id"
	ctxKeySpanID    ctxKey = "rae_span_id"
	ctxKeySampled   ctxKey = "rae_sampled"
	ctxKeyRequestID ctxKey = "rae_request_id"
	ctxKeyTenantID  ctxKey = "rae_tenant_id"
	ctxKeyAgentID   ctxKey = "rae_agent_id"
)

// TelemetryContext holds W3C trace context and RAE multi-tenant correlation attributes.
type TelemetryContext struct {
	TraceID   string `json:"trace_id"`
	SpanID    string `json:"span_id"`
	Sampled   bool   `json:"sampled"`
	RequestID string `json:"request_id"`
	TenantID  string `json:"tenant_id"`
	AgentID   string `json:"agent_id"`
}

// GenAITelemetry holds canonical OpenTelemetry GenAI semantic attributes and RAE domain attributes.
type GenAITelemetry struct {
	System        string  `json:"gen_ai.system"`
	RequestModel  string  `json:"gen_ai.request.model"`
	InputTokens   int     `json:"gen_ai.usage.input_tokens"`
	OutputTokens  int     `json:"gen_ai.usage.output_tokens"`
	CostUSD       float64 `json:"gen_ai.cost_usd"`
	CascadeTier   string  `json:"rae.cascade.tier"`
	MemoryLayer   string  `json:"rae.memory.layer"`
	SecurityClass string  `json:"rae.security.info_class"`
}

// Attrs converts GenAITelemetry into canonical slog.Attr items.
func (g GenAITelemetry) Attrs() []slog.Attr {
	return []slog.Attr{
		slog.String("gen_ai.system", g.System),
		slog.String("gen_ai.request.model", g.RequestModel),
		slog.Int("gen_ai.usage.input_tokens", g.InputTokens),
		slog.Int("gen_ai.usage.output_tokens", g.OutputTokens),
		slog.Float64("gen_ai.cost_usd", g.CostUSD),
		slog.String("rae.cascade.tier", g.CascadeTier),
		slog.String("rae.memory.layer", g.MemoryLayer),
		slog.String("rae.security.info_class", g.SecurityClass),
	}
}

// WithTelemetryContext injects TelemetryContext into a context.Context.
func WithTelemetryContext(ctx context.Context, tc TelemetryContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if tc.TraceID == "" {
		tc.TraceID = generateHexID(16)
	}
	if tc.SpanID == "" {
		tc.SpanID = generateHexID(8)
	}
	if tc.RequestID == "" {
		tc.RequestID = "req-" + tc.SpanID
	}
	ctx = context.WithValue(ctx, ctxKeyTraceID, tc.TraceID)
	ctx = context.WithValue(ctx, ctxKeySpanID, tc.SpanID)
	ctx = context.WithValue(ctx, ctxKeySampled, tc.Sampled)
	ctx = context.WithValue(ctx, ctxKeyRequestID, tc.RequestID)
	ctx = context.WithValue(ctx, ctxKeyTenantID, tc.TenantID)
	ctx = context.WithValue(ctx, ctxKeyAgentID, tc.AgentID)
	return ctx
}

// ExtractTelemetryContext extracts TelemetryContext from a context.Context.
func ExtractTelemetryContext(ctx context.Context) TelemetryContext {
	if ctx == nil {
		return TelemetryContext{}
	}
	strVal := func(k ctxKey) string {
		if v, ok := ctx.Value(k).(string); ok {
			return v
		}
		return ""
	}
	sampled, _ := ctx.Value(ctxKeySampled).(bool)
	return TelemetryContext{
		TraceID:   strVal(ctxKeyTraceID),
		SpanID:    strVal(ctxKeySpanID),
		Sampled:   sampled,
		RequestID: strVal(ctxKeyRequestID),
		TenantID:  strVal(ctxKeyTenantID),
		AgentID:   strVal(ctxKeyAgentID),
	}
}

func generateHexID(numBytes int) string {
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return strings.Repeat("a", numBytes*2)
	}
	return hex.EncodeToString(b)
}

// ParseW3CTraceparent parses a W3C traceparent header ("00-<32hex>-<16hex>-<2hex>") into traceID, spanID, sampled, and ok.
func ParseW3CTraceparent(header string) (traceID, spanID string, sampled, ok bool) {
	parts := strings.Split(strings.TrimSpace(header), "-")
	if len(parts) != 4 || parts[0] != "00" {
		return "", "", false, false
	}
	if len(parts[1]) != 32 || len(parts[2]) != 16 || len(parts[3]) != 2 {
		return "", "", false, false
	}
	if nonHexRegex.MatchString(parts[1]) || nonHexRegex.MatchString(parts[2]) || nonHexRegex.MatchString(parts[3]) {
		return "", "", false, false
	}
	if parts[1] == strings.Repeat("0", 32) || parts[2] == strings.Repeat("0", 16) {
		return "", "", false, false
	}
	tID := strings.ToLower(parts[1])
	sID := strings.ToLower(parts[2])
	fl := strings.ToLower(parts[3])
	return tID, sID, fl == "01", true
}

var (
	sensitiveKeys = map[string]bool{
		"password":      true,
		"secret":        true,
		"token":         true,
		"authorization": true,
		"api_key":       true,
		"apikey":        true,
		"pesel":         true,
		"nip":           true,
		"jwt":           true,
		"private_key":   true,
	}
	emailRegex = regexp.MustCompile(`([a-zA-Z0-9._%+\-])[a-zA-Z0-9._%+\-]*@([a-zA-Z0-9.\-]+\.[a-zA-Z]{2,})`)
	peselRegex = regexp.MustCompile(`\b\d{11}\b`)
	bearerRx   = regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-._~+/]+=*`)
)

// MaskPIIString redacts emails, Bearer tokens, and 11-digit PESEL numbers inside free-form strings.
func MaskPIIString(input string) string {
	out := bearerRx.ReplaceAllString(input, "Bearer [REDACTED]")
	out = emailRegex.ReplaceAllString(out, "$1***@$2")
	out = peselRegex.ReplaceAllString(out, "[REDACTED_PESEL]")
	return out
}

// RedactAttr is a slog ReplaceAttr function that masks sensitive keys and PII values.
func RedactAttr(_ []string, a slog.Attr) slog.Attr {
	keyLow := strings.ToLower(strings.TrimSpace(a.Key))
	for k := range sensitiveKeys {
		if keyLow == k || strings.Contains(keyLow, k) {
			return slog.String(a.Key, "[REDACTED]")
		}
	}
	if a.Value.Kind() == slog.KindString {
		return slog.String(a.Key, MaskPIIString(a.Value.String()))
	}
	return a
}

// ContextJSONHandler wraps slog.JSONHandler to automatically attach TelemetryContext from context.Context.
type ContextJSONHandler struct {
	inner slog.Handler
}

// Enabled delegates to the underlying JSONHandler.
func (h *ContextJSONHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle enriches the slog.Record with trace_id, span_id, request_id, tenant_id, and agent_id from ctx.
func (h *ContextJSONHandler) Handle(ctx context.Context, r slog.Record) error {
	tc := ExtractTelemetryContext(ctx)
	if tc.TraceID != "" {
		r.AddAttrs(slog.String("trace_id", tc.TraceID))
	}
	if tc.SpanID != "" {
		r.AddAttrs(slog.String("span_id", tc.SpanID))
	}
	if tc.RequestID != "" {
		r.AddAttrs(slog.String("request_id", tc.RequestID))
	}
	if tc.TenantID != "" {
		r.AddAttrs(slog.String("tenant_id", tc.TenantID))
	}
	if tc.AgentID != "" {
		r.AddAttrs(slog.String("agent_id", tc.AgentID))
	}
	return h.inner.Handle(ctx, r)
}

// WithAttrs delegates to the underlying handler.
func (h *ContextJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextJSONHandler{inner: h.inner.WithAttrs(attrs)}
}

// WithGroup delegates to the underlying handler.
func (h *ContextJSONHandler) WithGroup(name string) slog.Handler {
	return &ContextJSONHandler{inner: h.inner.WithGroup(name)}
}

// NewCanonicalLogger constructs a canonical RAE JSON logger writing 1-line JSON records with PII redaction and base service metadata.
func NewCanonicalLogger(service, env, version string, w io.Writer, level slog.Level) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	if service == "" {
		service = "rae-suite"
	}
	if env == "" {
		env = "production"
	}
	if version == "" {
		version = "5.3.6"
	}
	w = GlobalObservabilityHub().WrapWriter(w, service, version, env)
	baseHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: RedactAttr,
	})
	withMeta := baseHandler.WithAttrs([]slog.Attr{
		slog.String("service", service),
		slog.String("env", env),
		slog.String("version", version),
	})
	return slog.New(&ContextJSONHandler{inner: withMeta})
}

// LogGenAIInvocation emits a structured canonical log entry with OpenTelemetry GenAI attributes.
func LogGenAIInvocation(ctx context.Context, logger *slog.Logger, msg string, telemetry GenAITelemetry, extraAttrs ...slog.Attr) {
	if logger == nil {
		return
	}
	attrs := append(telemetry.Attrs(), extraAttrs...)
	anyAttrs := make([]any, 0, len(attrs))
	for _, a := range attrs {
		anyAttrs = append(anyAttrs, a)
	}
	logger.Log(ctx, slog.LevelInfo, msg, anyAttrs...)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if sr.status == 0 {
		sr.status = http.StatusOK
	}
	n, err := sr.ResponseWriter.Write(b)
	sr.bytes += n
	return n, err
}

func isHealthEndpoint(path string) bool {
	p := strings.ToLower(strings.TrimSpace(path))
	switch p {
	case "/health", "/healthz", "/livez", "/readyz", "/system-logs", "/logs":
		return true
	default:
		return strings.HasPrefix(p, "/_meta/") || strings.HasPrefix(p, "/api/_meta/")
	}
}

// RequestLoggerMiddleware extracts/propagates W3C traceparent, attaches TelemetryContext, and logs HTTP requests (suppressing INFO noise on health endpoints).
func RequestLoggerMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ServeObservabilityEndpoint(w, r) {
			return
		}
		start := time.Now()
		tpHeader := r.Header.Get("traceparent")
		traceID, parentSpanID, sampled, ok := ParseW3CTraceparent(tpHeader)
		if !ok {
			traceID = generateHexID(16)
			sampled = true
		}
		spanID := generateHexID(8)
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = "req-" + spanID
		}
		tenantID := r.Header.Get("X-Tenant-ID")
		agentID := r.Header.Get("X-Agent-ID")

		tc := TelemetryContext{
			TraceID:   traceID,
			SpanID:    spanID,
			Sampled:   sampled,
			RequestID: reqID,
			TenantID:  tenantID,
			AgentID:   agentID,
		}
		ctx := WithTelemetryContext(r.Context(), tc)
		w.Header().Set("traceparent", InjectW3CTraceContext(traceID, spanID, sampled))
		w.Header().Set("X-Request-ID", reqID)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r.WithContext(ctx))

		durMs := float64(time.Since(start).Microseconds()) / 1000.0
		routePattern := NormalizeRoutePattern(r.URL.Path)
		if logger != nil {
			if isHealthEndpoint(r.URL.Path) && rec.status < 400 {
				logger.Log(ctx, slog.LevelDebug, "health_check_completed",
					slog.String("event", "http_request_completed"),
					slog.String("log_category", "infrastructure"),
					slog.String("method", r.Method),
					slog.String("http_method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("http_path", r.URL.Path),
					slog.String("route", routePattern),
					slog.String("http_route", routePattern),
					slog.Int("status", rec.status),
					slog.Int("http_status", rec.status),
					slog.Int("status_code", rec.status),
					slog.Float64("duration_ms", durMs),
				)
				return
			}
			lvl := slog.LevelInfo
			if rec.status >= 500 {
				lvl = slog.LevelError
			} else if rec.status >= 400 {
				lvl = slog.LevelWarn
			}
			logger.Log(ctx, lvl, "http_request_completed",
				slog.String("event", "http_request_completed"),
				slog.String("log_category", "access"),
				slog.String("method", r.Method),
				slog.String("http_method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("http_path", r.URL.Path),
				slog.String("route", routePattern),
				slog.String("http_route", routePattern),
				slog.Int("status", rec.status),
				slog.Int("http_status", rec.status),
				slog.Int("status_code", rec.status),
				slog.Int("bytes", rec.bytes),
				slog.Int("response_bytes", rec.bytes),
				slog.Float64("duration_ms", durMs),
				slog.String("parent_span_id", parentSpanID),
			)
		}
	})
}

// RecoveryMiddleware catches panics, logs an ERROR with stack_trace and W3C correlation, and returns HTTP 500 JSON.
func RecoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := string(debug.Stack())
				if logger != nil {
					logger.Log(r.Context(), slog.LevelError, "http_handler_panic_recovered",
						slog.String("event", "http_handler_panic_recovered"),
						slog.String("error_code", "ERR_INTERNAL_PANIC"),
						slog.String("panic_value", fmt.Sprintf("%v", rec)),
						slog.String("stack_trace", stack),
					)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error_code": "ERR_INTERNAL_PANIC",
					"status":     "INTERNAL_SERVER_ERROR",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// TestLogRecorder captures structured JSON log lines in memory for unit/contract test assertions.
type TestLogRecorder struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// NewTestLogRecorder creates a new thread-safe TestLogRecorder.
func NewTestLogRecorder() *TestLogRecorder {
	return &TestLogRecorder{}
}

func (r *TestLogRecorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.Write(p)
}

// RawString returns the raw newline-delimited JSON output.
func (r *TestLogRecorder) RawString() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

// Entries parses each non-empty line as a JSON object.
func (r *TestLogRecorder) Entries() ([]map[string]interface{}, error) {
	r.mu.Lock()
	raw := r.buf.String()
	r.mu.Unlock()

	lines := strings.Split(strings.TrimSpace(raw), "\n")
	var out []map[string]interface{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			return nil, fmt.Errorf("invalid JSON log line %q: %w", line, err)
		}
		out = append(out, obj)
	}
	return out, nil
}

// AssertContainsEvent verifies that at least one recorded JSON log entry matches the given msg or event name.
func (r *TestLogRecorder) AssertContainsEvent(eventOrMsg string) (map[string]interface{}, bool) {
	entries, err := r.Entries()
	if err != nil {
		return nil, false
	}
	for _, e := range entries {
		if e["msg"] == eventOrMsg || e["event"] == eventOrMsg {
			return e, true
		}
	}
	return nil, false
}

// AssertNoSecretsLeaked checks that none of the provided raw secret strings appear anywhere in the recorded logs.
func (r *TestLogRecorder) AssertNoSecretsLeaked(secrets ...string) error {
	raw := r.RawString()
	for _, s := range secrets {
		if s != "" && strings.Contains(raw, s) {
			return fmt.Errorf("secret value %q leaked in logs", s)
		}
	}
	return nil
}

// Package handler implements the HTTP endpoints of the metrics server:
// text/plain and JSON metric updates, value lookups, batch updates,
// an HTML dashboard, and an optional /ping for the database backend.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/efer92/go-yandex-practicum-metrics/internal/audit"
	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Pinger is the optional database-connection check used by the /ping endpoint.
type Pinger interface {
	Ping(ctx context.Context) error
}

// MetricHandler exposes the metric HTTP endpoints on top of a MetricService.
type MetricHandler struct {
	svc       *service.MetricService
	log       *zap.Logger
	pinger    Pinger
	publisher *audit.Publisher
}

// NewMetricHandler returns a MetricHandler. The pinger and publisher are optional and may be nil.
func NewMetricHandler(svc *service.MetricService, log *zap.Logger, pinger Pinger, publisher *audit.Publisher) *MetricHandler {
	return &MetricHandler{svc: svc, log: log, pinger: pinger, publisher: publisher}
}

// clientIP returns X-Real-IP, the first X-Forwarded-For entry, or the host from RemoteAddr.
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// emitAudit dispatches the audit event asynchronously so that slow observers
// (e.g. a stalled remote audit sink) cannot block the response. The request
// context is intentionally not propagated — it is cancelled the moment the
// response is sent, which would abort an in-flight HTTP audit POST.
func (h *MetricHandler) emitAudit(r *http.Request, metrics []string) {
	if h.publisher == nil {
		return
	}
	e := audit.Event{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: clientIP(r),
	}
	go h.publisher.Notify(context.Background(), e)
}

// Routes returns a chi.Router with all metric endpoints mounted.
func (h *MetricHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListMetrics)
	r.Get("/ping", h.Ping)

	r.Post("/update", h.UpdateMetricJSON)
	r.Post("/update/", h.UpdateMetricJSON)
	r.Post("/value", h.GetValueJSON)
	r.Post("/value/", h.GetValueJSON)
	r.Post("/updates/", h.UpdateMetricsBatch)

	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)
	r.Get("/value/{type}/{name}", h.GetValue)

	return r
}

// internalError logs the 5xx error and responds with a safe generic message.
func (h *MetricHandler) internalError(w http.ResponseWriter, msg string, err error) {
	h.log.Error(msg, zap.Error(err))
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// httpStatusForError maps a service-level error to an HTTP status code.
func httpStatusForError(err error) int {
	switch {
	case errors.Is(err, service.ErrMetricNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrUnknownType),
		errors.Is(err, service.ErrMetricNameEmpty),
		errors.Is(err, service.ErrValueRequired),
		errors.Is(err, service.ErrDeltaRequired):
		return http.StatusBadRequest
	default:
		return http.StatusBadRequest
	}
}

// Ping handles GET /ping and verifies the database connection.
// Responds 500 when the database backend is not configured or unreachable.
func (h *MetricHandler) Ping(w http.ResponseWriter, r *http.Request) {
	if h.pinger == nil {
		http.Error(w, "database not configured", http.StatusInternalServerError)
		return
	}
	if err := h.pinger.Ping(r.Context()); err != nil {
		h.internalError(w, "db ping failed", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// ─── text/plain эндпоинты ────────────────────────────────────────────────────

// UpdateMetric handles POST /update/{type}/{name}/{value} for text/plain clients.
func (h *MetricHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	value := chi.URLParam(r, "value")

	if name == "" {
		http.NotFound(w, r)
		return
	}

	if err := h.svc.UpdateMetric(r.Context(), mType, name, value); err != nil {
		http.Error(w, err.Error(), httpStatusForError(err))
		return
	}

	h.emitAudit(r, []string{name})
	w.WriteHeader(http.StatusOK)
}

// GetValue handles GET /value/{type}/{name} and returns the metric value as text.
func (h *MetricHandler) GetValue(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	if name == "" {
		http.NotFound(w, r)
		return
	}

	val, err := h.svc.GetValue(r.Context(), mType, name)
	if err != nil {
		http.Error(w, err.Error(), httpStatusForError(err))
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(val))
}

// ─── JSON эндпоинты ──────────────────────────────────────────────────────────

// UpdateMetricJSON handles POST /update with a JSON-encoded model.Metrics body
// and responds with the updated metric as JSON.
func (h *MetricHandler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	var m model.Metrics

	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	if m.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	if err := h.svc.UpdateMetricFromModel(r.Context(), m); err != nil {
		http.Error(w, err.Error(), httpStatusForError(err))
		return
	}

	updated, err := h.svc.GetMetric(r.Context(), m.MType, m.ID)
	if err != nil {
		h.internalError(w, "failed to get metric after update", err)
		return
	}

	h.emitAudit(r, []string{m.ID})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updated); err != nil {
		h.log.Error("failed to encode response", zap.Error(err))
	}
}

// GetValueJSON handles POST /value with a JSON-encoded model.Metrics descriptor
// (only ID and MType are read) and responds with the full metric as JSON.
func (h *MetricHandler) GetValueJSON(w http.ResponseWriter, r *http.Request) {
	var req model.Metrics

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	m, err := h.svc.GetMetric(r.Context(), req.MType, req.ID)
	if err != nil {
		http.Error(w, err.Error(), httpStatusForError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(m); err != nil {
		h.log.Error("failed to encode response", zap.Error(err))
	}
}

// UpdateMetricsBatch handles POST /updates/ with a JSON-encoded []model.Metrics
// and applies all updates atomically.
func (h *MetricHandler) UpdateMetricsBatch(w http.ResponseWriter, r *http.Request) {
	var metrics []model.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.svc.UpdateBatch(r.Context(), metrics); err != nil {
		h.internalError(w, "batch update failed", err)
		return
	}

	names := make([]string, 0, len(metrics))
	for _, m := range metrics {
		names = append(names, m.ID)
	}
	h.emitAudit(r, names)

	w.WriteHeader(http.StatusOK)
}

// ─── HTML dashboard ──────────────────────────────────────────────────────────

// MetricView is the row model used to render the HTML dashboard.
type MetricView struct {
	Type  string
	Name  string
	Value string
}

var listTemplate = template.Must(template.New("metrics").Parse(htmlTemplate))

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Metrics Dashboard</title>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            margin: 0;
            padding: 40px 20px;
            min-height: 100vh;
        }
        .container {
            max-width: 1000px;
            margin: 0 auto;
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px;
            text-align: center;
        }
        .header h1 { margin: 0 0 10px 0; font-size: 2.5em; font-weight: 300; letter-spacing: 1px; }
        .subtitle { opacity: 0.9; font-size: 1.1em; }
        .stats { display: flex; justify-content: center; gap: 20px; margin-top: 25px; flex-wrap: wrap; }
        .stat-card {
            background: rgba(255,255,255,0.15);
            backdrop-filter: blur(10px);
            padding: 12px 24px;
            border-radius: 25px;
            font-size: 0.9em;
            border: 1px solid rgba(255,255,255,0.2);
        }
        .content { padding: 0; }
        table { width: 100%; border-collapse: collapse; }
        thead { background: #f8f9fa; position: sticky; top: 0; z-index: 10; }
        th {
            padding: 20px; text-align: left; font-weight: 600; color: #495057;
            border-bottom: 2px solid #dee2e6; text-transform: uppercase; font-size: 0.8em; letter-spacing: 1px;
        }
        td { padding: 16px 20px; border-bottom: 1px solid #e9ecef; vertical-align: middle; }
        tr { transition: all 0.2s; }
        tr:hover { background-color: #f8f9fa; transform: scale(1.01); box-shadow: 0 2px 8px rgba(0,0,0,0.05); }
        .badge {
            display: inline-block; padding: 6px 14px; border-radius: 20px;
            font-size: 0.75em; font-weight: 700; text-transform: uppercase;
            letter-spacing: 0.5px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .badge-gauge { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; }
        .badge-counter { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); color: white; }
        .metric-name { font-weight: 600; color: #2d3748; font-size: 1.05em; }
        .metric-value {
            font-family: 'Courier New', Consolas, monospace; font-weight: 700;
            color: #1a202c; background: #edf2f7; padding: 6px 12px;
            border-radius: 6px; display: inline-block; font-size: 0.95em;
        }
        .empty-state { text-align: center; padding: 80px 40px; color: #718096; }
        .empty-state h3 { margin: 0 0 10px 0; color: #2d3748; font-size: 1.5em; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 Metrics Dashboard</h1>
            <div class="subtitle">Real-time monitoring</div>
            <div class="stats">
                <div class="stat-card">Total: <strong>{{len .}}</strong> metrics</div>
                <div class="stat-card">Gauge Metrics</div>
                <div class="stat-card">Counter Metrics</div>
            </div>
        </div>
        <div class="content">
            {{if .}}
            <table>
                <thead>
                    <tr>
                        <th width="120">Type</th>
                        <th>Metric Name</th>
                        <th width="200">Value</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .}}
                    <tr>
                        <td>
                            {{if eq .Type "gauge"}}
                            <span class="badge badge-gauge">Gauge</span>
                            {{else}}
                            <span class="badge badge-counter">Counter</span>
                            {{end}}
                        </td>
                        <td class="metric-name">{{.Name}}</td>
                        <td><span class="metric-value">{{.Value}}</span></td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
            {{else}}
            <div class="empty-state">
                <h3>No metrics available</h3>
                <p>Metrics will appear here once the agent sends data.</p>
            </div>
            {{end}}
        </div>
    </div>
</body>
</html>
`

// ListMetrics handles GET / and renders an HTML dashboard of all known metrics.
func (h *MetricHandler) ListMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.svc.GetAllMetrics(r.Context())

	data := make([]MetricView, 0, len(metrics))
	for key, val := range metrics {
		i := strings.IndexByte(key, '/')
		if i < 0 {
			continue
		}
		data = append(data, MetricView{
			Type:  key[:i],
			Name:  key[i+1:],
			Value: val,
		})
	}

	sort.Slice(data, func(i, j int) bool {
		if data[i].Type != data[j].Type {
			return data[i].Type < data[j].Type
		}
		return data[i].Name < data[j].Name
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := listTemplate.Execute(w, data); err != nil {
		h.log.Error("failed to execute template", zap.Error(err))
	}
}

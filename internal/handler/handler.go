package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"sort"
	"strings"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

type MetricHandler struct {
	svc *service.MetricService
}

func NewMetricHandler(svc *service.MetricService) *MetricHandler {
	return &MetricHandler{svc: svc}
}

func (h *MetricHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// text/plain эндпоинты (старые)
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)
	r.Get("/value/{type}/{name}", h.GetValue)

	// JSON эндпоинты (новые)
	r.Post("/update", h.UpdateMetricJSON)
	r.Post("/value", h.GetValueJSON)

	r.Get("/", h.ListMetrics)
	return r
}

// ─── text/plain эндпоинты ────────────────────────────────────────────────────

func (h *MetricHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	value := chi.URLParam(r, "value")

	if name == "" {
		http.NotFound(w, r)
		return
	}

	if err := h.svc.UpdateMetric(mType, name, value); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *MetricHandler) GetValue(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	if name == "" {
		http.NotFound(w, r)
		return
	}

	val, err := h.svc.GetValue(mType, name)
	if err != nil {
		if err.Error() == "metric not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(val))
}

// ─── JSON эндпоинты ──────────────────────────────────────────────────────────

// UpdateMetricJSON — POST /update — принимает метрику в JSON, сохраняет и возвращает её обратно.
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

	if err := h.svc.UpdateMetricFromModel(m); err != nil {
		if err.Error() == "unknown metric type" {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	// Возвращаем актуальное состояние метрики (для counter — накопленное значение)
	updated, err := h.svc.GetMetric(m.MType, m.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updated)
}

// GetValueJSON — POST /value — принимает {id, type}, возвращает метрику с заполненным значением.
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

	m, err := h.svc.GetMetric(req.MType, req.ID)
	if err != nil {
		if err.Error() == "metric not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(m)
}

// ─── HTML dashboard ──────────────────────────────────────────────────────────

type MetricView struct {
	Type  string
	Name  string
	Value string
}

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

func (h *MetricHandler) ListMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.svc.GetAllMetrics()

	var data []MetricView
	for key, val := range metrics {
		parts := strings.SplitN(key, "/", 2)
		if len(parts) == 2 {
			data = append(data, MetricView{
				Type:  parts[0],
				Name:  parts[1],
				Value: val,
			})
		}
	}

	sort.Slice(data, func(i, j int) bool {
		if data[i].Type != data[j].Type {
			return data[i].Type < data[j].Type
		}
		return data[i].Name < data[j].Name
	})

	tmpl, err := template.New("metrics").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, data)
}

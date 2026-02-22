package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/internal/repository"
	"github.com/efer92/go-yandex-practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHandler() *MetricHandler {
	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)
	return NewMetricHandler(svc)
}

func newTestServer(h *MetricHandler) *httptest.Server {
	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	return httptest.NewServer(r)
}

// ─── POST /update (JSON) ─────────────────────────────────────────────────────

func TestUpdateMetricJSON_Gauge(t *testing.T) {
	h := newTestHandler()
	srv := newTestServer(h)
	defer srv.Close()

	val := 42.5
	m := model.Metrics{ID: "TestGauge", MType: model.Gauge, Value: &val}
	body, _ := json.Marshal(m)

	resp, err := http.Post(srv.URL+"/update", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var result model.Metrics
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "TestGauge", result.ID)
	assert.Equal(t, model.Gauge, result.MType)
	require.NotNil(t, result.Value)
	assert.Equal(t, 42.5, *result.Value)
}

func TestUpdateMetricJSON_Counter(t *testing.T) {
	h := newTestHandler()
	srv := newTestServer(h)
	defer srv.Close()

	delta := int64(10)
	m := model.Metrics{ID: "TestCounter", MType: model.Counter, Delta: &delta}
	body, _ := json.Marshal(m)

	// Отправляем дважды — счётчик должен накопиться
	firstResp, err := http.Post(srv.URL+"/update", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	firstResp.Body.Close()
	body, _ = json.Marshal(m)
	resp, err := http.Post(srv.URL+"/update", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result model.Metrics
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.NotNil(t, result.Delta)
	assert.Equal(t, int64(20), *result.Delta) // 10 + 10
}

func TestUpdateMetricJSON_UnknownType(t *testing.T) {
	h := newTestHandler()
	srv := newTestServer(h)
	defer srv.Close()

	m := model.Metrics{ID: "X", MType: "unknown"}
	body, _ := json.Marshal(m)

	resp, err := http.Post(srv.URL+"/update", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateMetricJSON_EmptyID(t *testing.T) {
	h := newTestHandler()
	srv := newTestServer(h)
	defer srv.Close()

	val := 1.0
	m := model.Metrics{ID: "", MType: model.Gauge, Value: &val}
	body, _ := json.Marshal(m)

	resp, err := http.Post(srv.URL+"/update", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateMetricJSON_InvalidBody(t *testing.T) {
	h := newTestHandler()
	srv := newTestServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update", "application/json", bytes.NewBufferString("not json"))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── POST /value (JSON) ──────────────────────────────────────────────────────

func TestGetValueJSON_Gauge(t *testing.T) {
	h := newTestHandler()
	srv := newTestServer(h)
	defer srv.Close()

	// Сначала сохраняем
	val := 99.9
	body, _ := json.Marshal(model.Metrics{ID: "Alloc", MType: model.Gauge, Value: &val})
	setupResp, err := http.Post(srv.URL+"/update", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	setupResp.Body.Close()

	// Запрашиваем
	body, _ = json.Marshal(model.Metrics{ID: "Alloc", MType: model.Gauge})
	resp, err := http.Post(srv.URL+"/value", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var result model.Metrics
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.NotNil(t, result.Value)
	assert.Equal(t, 99.9, *result.Value)
}

func TestGetValueJSON_NotFound(t *testing.T) {
	h := newTestHandler()
	srv := newTestServer(h)
	defer srv.Close()

	body, _ := json.Marshal(model.Metrics{ID: "NonExistent", MType: model.Gauge})
	resp, err := http.Post(srv.URL+"/value", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetValueJSON_Counter(t *testing.T) {
	h := newTestHandler()
	srv := newTestServer(h)
	defer srv.Close()

	delta := int64(5)
	body, _ := json.Marshal(model.Metrics{ID: "PollCount", MType: model.Counter, Delta: &delta})
	setupResp, err := http.Post(srv.URL+"/update", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	setupResp.Body.Close()

	body, _ = json.Marshal(model.Metrics{ID: "PollCount", MType: model.Counter})
	resp, err := http.Post(srv.URL+"/value", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result model.Metrics
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.NotNil(t, result.Delta)
	assert.Equal(t, int64(5), *result.Delta)
}

// ─── Старые text/plain эндпоинты ─────────────────────────────────────────────

func TestUpdateMetric_TextPlain(t *testing.T) {
	h := newTestHandler()
	srv := newTestServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update/gauge/TestMetric/42.5", "text/plain", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetValue_TextPlain(t *testing.T) {
	h := newTestHandler()
	srv := newTestServer(h)
	defer srv.Close()

	setupResp, err := http.Post(srv.URL+"/update/gauge/TestMetric/42.5", "text/plain", nil)
	require.NoError(t, err)
	setupResp.Body.Close()

	resp, err := http.Get(srv.URL + "/value/gauge/TestMetric")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/efer92/go-yandex-practicum-metrics/internal/audit"
	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/internal/repository"
	"github.com/efer92/go-yandex-practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestHandler() *MetricHandler {
	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)
	return NewMetricHandler(svc, zap.NewNop(), nil, nil)
}

func newTestServer(h *MetricHandler) *httptest.Server {
	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	return httptest.NewServer(r)
}

// ─── mock Pinger ─────────────────────────────────────────────────────────────

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(_ context.Context) error {
	return m.err
}

// ─── GET /ping ────────────────────────────────────────────────────────────────

func TestPing_NoPinger(t *testing.T) {
	h := newTestHandler() // pinger == nil
	srv := newTestServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ping")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestPing_Success(t *testing.T) {
	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)
	h := NewMetricHandler(svc, zap.NewNop(), &mockPinger{err: nil}, nil)
	srv := newTestServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ping")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPing_Failure(t *testing.T) {
	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)
	h := NewMetricHandler(svc, zap.NewNop(), &mockPinger{err: assert.AnError}, nil)
	srv := newTestServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ping")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
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
	assert.Equal(t, int64(20), *result.Delta)
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

	val := 99.9
	body, _ := json.Marshal(model.Metrics{ID: "Alloc", MType: model.Gauge, Value: &val})
	setupResp, err := http.Post(srv.URL+"/update", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	setupResp.Body.Close()

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

// ─── audit ───────────────────────────────────────────────────────────────────

type recordingSink struct {
	events []audit.Event
}

func (s *recordingSink) Receive(_ context.Context, e audit.Event) error {
	s.events = append(s.events, e)
	return nil
}

func newHandlerWithSink(t *testing.T) (*MetricHandler, *recordingSink) {
	t.Helper()
	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)
	sink := &recordingSink{}
	pub := audit.NewPublisher(zap.NewNop(), sink)
	return NewMetricHandler(svc, zap.NewNop(), nil, pub), sink
}

func TestUpdateMetricJSON_AuditEmits(t *testing.T) {
	h, sink := newHandlerWithSink(t)
	srv := newTestServer(h)
	defer srv.Close()

	val := 1.5
	body, _ := json.Marshal(model.Metrics{ID: "Alloc", MType: model.Gauge, Value: &val})
	resp, err := http.Post(srv.URL+"/update", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, sink.events, 1)
	assert.Equal(t, []string{"Alloc"}, sink.events[0].Metrics)
	assert.NotEmpty(t, sink.events[0].IPAddress)
	assert.NotZero(t, sink.events[0].TS)
}

func TestUpdateMetricsBatch_AuditEmitsAllNames(t *testing.T) {
	h, sink := newHandlerWithSink(t)
	srv := newTestServer(h)
	defer srv.Close()

	v1 := 1.0
	d1 := int64(3)
	body, _ := json.Marshal([]model.Metrics{
		{ID: "Alloc", MType: model.Gauge, Value: &v1},
		{ID: "PollCount", MType: model.Counter, Delta: &d1},
	})
	resp, err := http.Post(srv.URL+"/updates/", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, sink.events, 1)
	assert.Equal(t, []string{"Alloc", "PollCount"}, sink.events[0].Metrics)
}

func TestUpdateMetric_TextPlain_AuditEmits(t *testing.T) {
	h, sink := newHandlerWithSink(t)
	srv := newTestServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update/gauge/Alloc/3.14", "text/plain", nil)
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, sink.events, 1)
	assert.Equal(t, []string{"Alloc"}, sink.events[0].Metrics)
}

func TestUpdateMetricJSON_AuditNotEmittedOnError(t *testing.T) {
	h, sink := newHandlerWithSink(t)
	srv := newTestServer(h)
	defer srv.Close()

	body, _ := json.Marshal(model.Metrics{ID: "Bad", MType: "unknown"})
	resp, err := http.Post(srv.URL+"/update", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	resp.Body.Close()

	assert.Empty(t, sink.events)
}

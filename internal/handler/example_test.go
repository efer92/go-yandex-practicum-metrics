package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/efer92/go-yandex-practicum-metrics/internal/handler"
	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	"github.com/efer92/go-yandex-practicum-metrics/internal/repository"
	"github.com/efer92/go-yandex-practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func exampleServer() *httptest.Server {
	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)
	h := handler.NewMetricHandler(svc, zap.NewNop(), nil, nil)
	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	return httptest.NewServer(r)
}

func mustPost(url, contentType string, body []byte) *http.Response {
	resp, err := http.Post(url, contentType, bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	return resp
}

func mustGet(url string) *http.Response {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	return resp
}

// ExampleMetricHandler_UpdateMetric demonstrates the text/plain update endpoint
// POST /update/{type}/{name}/{value}.
func ExampleMetricHandler_UpdateMetric() {
	srv := exampleServer()
	defer srv.Close()

	resp := mustPost(srv.URL+"/update/gauge/Alloc/3.14", "text/plain", nil)
	defer resp.Body.Close()

	fmt.Println("status:", resp.StatusCode)

	// Output:
	// status: 200
}

// ExampleMetricHandler_GetValue demonstrates fetching a gauge value via
// GET /value/{type}/{name} after it has been set.
func ExampleMetricHandler_GetValue() {
	srv := exampleServer()
	defer srv.Close()

	mustPost(srv.URL+"/update/gauge/Alloc/3.14", "text/plain", nil).Body.Close()

	resp := mustGet(srv.URL + "/value/gauge/Alloc")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	fmt.Println("status:", resp.StatusCode)
	fmt.Println("body:  ", string(body))

	// Output:
	// status: 200
	// body:   3.14
}

// ExampleMetricHandler_UpdateMetricJSON demonstrates the JSON update endpoint
// POST /update which returns the updated metric as JSON.
func ExampleMetricHandler_UpdateMetricJSON() {
	srv := exampleServer()
	defer srv.Close()

	delta := int64(5)
	body, _ := json.Marshal(model.Metrics{ID: "PollCount", MType: model.Counter, Delta: &delta})

	resp := mustPost(srv.URL+"/update", "application/json", body)
	defer resp.Body.Close()

	var updated model.Metrics
	_ = json.NewDecoder(resp.Body).Decode(&updated)

	fmt.Println("status:", resp.StatusCode)
	fmt.Println("id:    ", updated.ID)
	fmt.Println("type:  ", updated.MType)
	fmt.Println("delta: ", *updated.Delta)

	// Output:
	// status: 200
	// id:     PollCount
	// type:   counter
	// delta:  5
}

// ExampleMetricHandler_GetValueJSON demonstrates POST /value with a JSON
// descriptor: only ID and MType are read; the full metric is returned.
func ExampleMetricHandler_GetValueJSON() {
	srv := exampleServer()
	defer srv.Close()

	val := 42.0
	put, _ := json.Marshal(model.Metrics{ID: "Alloc", MType: model.Gauge, Value: &val})
	mustPost(srv.URL+"/update", "application/json", put).Body.Close()

	q, _ := json.Marshal(model.Metrics{ID: "Alloc", MType: model.Gauge})
	resp := mustPost(srv.URL+"/value", "application/json", q)
	defer resp.Body.Close()

	var got model.Metrics
	_ = json.NewDecoder(resp.Body).Decode(&got)

	fmt.Println("status:", resp.StatusCode)
	fmt.Println("value: ", *got.Value)

	// Output:
	// status: 200
	// value:  42
}

// ExampleMetricHandler_UpdateMetricsBatch demonstrates the batch update endpoint
// POST /updates/ that accepts a JSON array of metrics.
func ExampleMetricHandler_UpdateMetricsBatch() {
	srv := exampleServer()
	defer srv.Close()

	v := 1.5
	d := int64(2)
	body, _ := json.Marshal([]model.Metrics{
		{ID: "Alloc", MType: model.Gauge, Value: &v},
		{ID: "PollCount", MType: model.Counter, Delta: &d},
	})

	resp := mustPost(srv.URL+"/updates/", "application/json", body)
	defer resp.Body.Close()

	fmt.Println("status:", resp.StatusCode)

	// Output:
	// status: 200
}

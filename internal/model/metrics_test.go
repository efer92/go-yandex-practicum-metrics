package model

import (
	"encoding/json"
	"testing"
)

func TestMetrics_JSONMarshal(t *testing.T) {
	delta := int64(10)
	value := 123.45

	m := Metrics{
		ID:    "test",
		MType: Gauge,
		Delta: &delta,
		Value: &value,
		Hash:  "abc123",
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Проверяем что поля на месте
	jsonStr := string(data)
	if !contains(jsonStr, `"id":"test"`) {
		t.Error("missing id in json")
	}
	if !contains(jsonStr, `"type":"gauge"`) {
		t.Error("missing type in json")
	}
	if !contains(jsonStr, `"hash":"abc123"`) {
		t.Error("missing hash in json")
	}
}

func TestMetrics_JSONUnmarshal(t *testing.T) {
	jsonData := `{
        "id": "Alloc",
        "type": "gauge",
        "value": 100.5,
        "hash": ""
    }`

	var m Metrics
	err := json.Unmarshal([]byte(jsonData), &m)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if m.ID != "Alloc" {
		t.Errorf("expected ID=Alloc, got %s", m.ID)
	}
	if m.MType != Gauge {
		t.Errorf("expected MType=gauge, got %s", m.MType)
	}
	if m.Value == nil || *m.Value != 100.5 {
		t.Error("wrong value")
	}
}

func TestMetrics_OMitempty(t *testing.T) {
	// Тестируем что nil поля не попадают в JSON
	m := Metrics{
		ID:    "test",
		MType: Counter,
		// Delta и Value nil - не должны быть в JSON
	}

	data, _ := json.Marshal(m)
	jsonStr := string(data)

	if contains(jsonStr, "delta") {
		t.Error("delta should be omitted when nil")
	}
	if contains(jsonStr, "value") {
		t.Error("value should be omitted when nil")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

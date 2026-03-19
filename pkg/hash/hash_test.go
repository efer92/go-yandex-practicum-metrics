package hash

import (
	"testing"
)

func TestSign(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		key  string
		want string
	}{
		{
			name: "same input produces same hash",
			body: []byte(`{"id":"Alloc","type":"gauge","value":1024.5}`),
			key:  "secret",
		},
		{
			name: "different keys produce different hashes",
			body: []byte(`test`),
			key:  "key1",
		},
		{
			name: "empty body",
			body: []byte{},
			key:  "secret",
		},
		{
			name: "empty key",
			body: []byte(`test`),
			key:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h1 := Sign(tt.body, tt.key)
			h2 := Sign(tt.body, tt.key)
			if h1 != h2 {
				t.Errorf("Sign() is not deterministic: %s != %s", h1, h2)
			}
			if len(h1) != 64 { // SHA256 = 32 bytes = 64 hex chars
				t.Errorf("Sign() wrong length: got %d, want 64", len(h1))
			}
		})
	}
}

func TestSign_DifferentKeysProduceDifferentHashes(t *testing.T) {
	body := []byte(`test body`)
	h1 := Sign(body, "key1")
	h2 := Sign(body, "key2")
	if h1 == h2 {
		t.Error("different keys should produce different hashes")
	}
}

func TestSign_DifferentBodiesProduceDifferentHashes(t *testing.T) {
	key := "secret"
	h1 := Sign([]byte(`body1`), key)
	h2 := Sign([]byte(`body2`), key)
	if h1 == h2 {
		t.Error("different bodies should produce different hashes")
	}
}

func TestEqual(t *testing.T) {
	body := []byte(`{"id":"test","type":"gauge","value":42}`)
	key := "mysecret"
	sig := Sign(body, key)

	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{
			name: "equal hashes",
			a:    sig,
			b:    sig,
			want: true,
		},
		{
			name: "different hashes",
			a:    sig,
			b:    Sign([]byte(`other`), key),
			want: false,
		},
		{
			name: "invalid hex a",
			a:    "not-hex!!",
			b:    sig,
			want: false,
		},
		{
			name: "invalid hex b",
			a:    sig,
			b:    "not-hex!!",
			want: false,
		},
		{
			name: "both empty strings",
			a:    "",
			b:    "",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Equal(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("Equal(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

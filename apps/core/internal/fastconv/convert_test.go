package fastconv

import (
	"testing"
)

func TestBytesToString_EdgeCases(t *testing.T) {
	empty := []byte{}
	if str := BytesToString(empty); str != "" {
		t.Errorf("Expected empty string, got '%s'", str)
	}

	raw := []byte("budment-benchmark-token")
	str := BytesToString(raw)
	if str != "budment-benchmark-token" {
		t.Errorf("String mismatch: got '%s'", str)
	}
}

func TestStringToBytes_EdgeCases(t *testing.T) {
	if b := StringToBytes(""); b != nil {
		t.Errorf("Expected nil slice for empty string, got %v", b)
	}

	raw := "engine-payload"
	b := StringToBytes(raw)
	if string(b) != raw {
		t.Errorf("Bytes mismatch: got '%s'", string(b))
	}
}

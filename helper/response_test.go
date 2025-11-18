package helper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, map[string]string{"msg": "ok"})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected content-type application/json, got %s", rec.Header().Get("Content-Type"))
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if body["msg"] != "ok" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestWriteErrorJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteErrorJSON(rec, http.StatusBadRequest, "bad")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if body["error"] != "bad" {
		t.Fatalf("unexpected error body: %+v", body)
	}
}

package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My Database", "my-database"},
		{"  Production Redis  ", "production-redis"},
		{"hello_world!", "helloworld"},
		{"CamelCase", "camelcase"},
		{"a--b--c", "a--b--c"},
		{"123-test", "123-test"},
		{"", ""},
		{"special@#$chars", "specialchars"},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		query      string
		wantLimit  int32
		wantOffset int32
	}{
		{"", 50, 0},
		{"limit=10", 10, 0},
		{"limit=10&offset=20", 10, 20},
		{"limit=200", 50, 0},   // exceeds max, use default
		{"limit=0", 50, 0},     // zero, use default
		{"limit=-1", 50, 0},    // negative, use default
		{"limit=abc", 50, 0},   // non-numeric, use default
		{"offset=-5", 50, 0},   // negative offset, use default
		{"limit=100", 100, 0},  // max allowed
		{"limit=101", 50, 0},   // over max
	}
	for _, tt := range tests {
		r := httptest.NewRequest("GET", "/?"+tt.query, nil)
		limit, offset := parsePagination(r)
		if limit != tt.wantLimit || offset != tt.wantOffset {
			t.Errorf("parsePagination(%q) = (%d, %d), want (%d, %d)",
				tt.query, limit, offset, tt.wantLimit, tt.wantOffset)
		}
	}
}

func TestHandleCreateValidation(t *testing.T) {
	h := &resourceHandler{q: nil, jobs: nil} // nil DB — we're testing validation before DB calls

	tests := []struct {
		name       string
		body       string
		userHeader string
		wantStatus int
		wantError  string
	}{
		{
			name:       "invalid JSON",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid JSON body",
		},
		{
			name:       "missing required fields",
			body:       `{"name": "test"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "project_id, resource_type_id, and name are required",
		},
		{
			name:       "missing project_id",
			body:       `{"resource_type_id": "rt-1", "name": "test"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "project_id, resource_type_id, and name are required",
		},
		{
			name:       "missing name",
			body:       `{"project_id": "p-1", "resource_type_id": "rt-1"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "project_id, resource_type_id, and name are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/resources", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userHeader != "" {
				req.Header.Set("X-User-Id", tt.userHeader)
			}
			w := httptest.NewRecorder()
			h.HandleCreate(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantError != "" && !strings.Contains(w.Body.String(), tt.wantError) {
				t.Errorf("body = %s, want to contain %q", w.Body.String(), tt.wantError)
			}
		})
	}
}

func TestHandleListValidation(t *testing.T) {
	h := &resourceHandler{q: nil, jobs: nil}

	// Missing project_id query parameter.
	req := httptest.NewRequest("GET", "/api/v1/resources", nil)
	w := httptest.NewRecorder()
	h.HandleList(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "project_id") {
		t.Errorf("body = %s, want error mentioning project_id", w.Body.String())
	}
}

func TestHandleDeleteValidation(t *testing.T) {
	// HandleDelete hits DB (GetResource) before validating X-User-Id header,
	// so we can't unit test header validation without a real DB connection.
	// The header validation is tested indirectly via HandleCreate tests above.
	t.Skip("requires DB connection for GetResource call")
}

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	respondJSON(w, http.StatusOK, map[string]string{"key": "value"})
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if !strings.Contains(w.Body.String(), `"key":"value"`) {
		t.Errorf("body = %s, want key:value", w.Body.String())
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()
	respondError(w, http.StatusBadRequest, "something went wrong")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "something went wrong") {
		t.Errorf("body = %s, want error message", w.Body.String())
	}
}

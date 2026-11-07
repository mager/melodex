package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWhoSampledEndpointIsRemoved(t *testing.T) {
	router := NewRouter(nil, nil, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("POST", "/whosampled", strings.NewReader(`not json`)))
	if w.Code != 404 {
		t.Fatalf("retired endpoint returned %d, want 404", w.Code)
	}
}

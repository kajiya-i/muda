package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kajiya-i/muda/backend/internal/handler"
	"github.com/kajiya-i/muda/backend/internal/infrastructure/httpapi/apigen"
)

func newHandler() http.Handler {
	return apigen.HandlerFromMux(apigen.NewStrictHandler(handler.New(), nil), http.NewServeMux())
}

func TestGetHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	newHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGetHealthRejectsOtherMethods(t *testing.T) {
	rec := httptest.NewRecorder()
	newHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/healthz", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

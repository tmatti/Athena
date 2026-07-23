package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tmatti/athena/internal/store"
)

func TestBearerAuth(t *testing.T) {
	const key = "secret-key-0123456789"
	var gotSubject string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSubject = SubjectFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	handler := BearerAuth(key, nil, "")(next)

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"missing", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic " + key, http.StatusUnauthorized},
		{"wrong token", "Bearer nope", http.StatusUnauthorized},
		{"valid", "Bearer " + key, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/memories", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, tc.want, rec.Code)
			if tc.want == http.StatusOK {
				require.Equal(t, store.SubjectOwner, gotSubject)
			}
		})
	}
}

func TestBearerAuthTokenValidator(t *testing.T) {
	const key = "secret-key-0123456789"
	validate := func(_ context.Context, token string) (string, error) {
		if token == "athat_good" {
			return store.SubjectOwner, nil
		}
		return "", errors.New("unknown token")
	}

	var gotSubject string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSubject = SubjectFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	const metadataURL = "https://athena.example.com/.well-known/oauth-protected-resource/mcp"
	handler := BearerAuth(key, validate, metadataURL)(next)

	// A valid OAuth token is accepted and resolves the subject.
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer athat_good")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, store.SubjectOwner, gotSubject)

	// The static key still works alongside the validator.
	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// A rejected token gets 401 with the RFC 9728 discovery challenge.
	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer athat_bad")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, `Bearer resource_metadata="`+metadataURL+`"`, rec.Header().Get("WWW-Authenticate"))
}

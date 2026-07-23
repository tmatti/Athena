package oauth

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type registrationRequest struct {
	RedirectURIs []string `json:"redirect_uris"`
	ClientName   string   `json:"client_name"`
}

// handleRegister implements RFC 7591 dynamic client registration. Only public
// clients are supported: no secret is issued and the token endpoint requires
// PKCE instead. Requested auth methods are overridden to "none" (the RFC
// allows the server to substitute registration values).
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registrationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "request body must be a JSON client metadata document")
		return
	}
	if len(req.RedirectURIs) == 0 {
		writeOAuthError(w, http.StatusBadRequest, "invalid_redirect_uri", "redirect_uris is required")
		return
	}
	for _, u := range req.RedirectURIs {
		if !validRedirectURI(u) {
			writeOAuthError(w, http.StatusBadRequest, "invalid_redirect_uri", "redirect URIs must be https (or http on localhost): "+u)
			return
		}
	}

	client, err := s.store.CreateOAuthClient(r.Context(), req.ClientName, req.RedirectURIs)
	if err != nil {
		s.log.Error("register oauth client", "error", err)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "could not register client")
		return
	}
	s.log.Info("registered oauth client", "client_id", client.ID, "name", client.Name, "redirect_uris", client.RedirectURIs)

	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  client.ID,
		"client_id_issued_at":        client.CreatedAt.Unix(),
		"client_name":                client.Name,
		"redirect_uris":              client.RedirectURIs,
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
	})
}

// validRedirectURI accepts absolute https URLs, and plain http only on
// loopback hosts (CLI-style clients that run a local callback server).
func validRedirectURI(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	switch u.Scheme {
	case "https":
		return true
	case "http":
		h := u.Hostname()
		return h == "localhost" || h == "127.0.0.1" || h == "::1"
	default:
		return false
	}
}

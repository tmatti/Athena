package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePublicBaseURL(t *testing.T) {
	valid := map[string]string{
		"https://athena.example.com":  "https://athena.example.com",
		"https://athena.example.com/": "https://athena.example.com",
		"http://localhost:8080":       "http://localhost:8080",
		"http://127.0.0.1:8080":       "http://127.0.0.1:8080",
	}
	for in, want := range valid {
		got, err := validatePublicBaseURL(in)
		require.NoError(t, err, in)
		require.Equal(t, want, got)
	}

	invalid := []string{
		"http://athena.example.com",      // plain http off loopback
		"https://athena.example.com/api", // path not allowed
		"athena.example.com",             // no scheme
		"ftp://athena.example.com",
	}
	for _, in := range invalid {
		_, err := validatePublicBaseURL(in)
		require.Error(t, err, in)
	}
}

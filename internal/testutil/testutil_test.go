package testutil

import (
	"net/http"
	"os"
	"testing"

	"github.com/antiwork/gumroad-cli/internal/config"
)

func TestSetup_ClearsEnvAccessToken(t *testing.T) {
	t.Setenv(config.EnvAccessToken, "should-be-cleared")

	Setup(t, func(w http.ResponseWriter, r *http.Request) {})

	if got := os.Getenv(config.EnvAccessToken); got != "" {
		t.Fatalf("Setup should clear %s, got %q", config.EnvAccessToken, got)
	}

	info, err := config.ResolveToken()
	if err != nil {
		t.Fatalf("ResolveToken failed: %v", err)
	}
	if info.Value != "test-token" {
		t.Fatalf("expected config-file token, got %q", info.Value)
	}
	if info.Source != config.TokenSourceConfig {
		t.Fatalf("expected source %q, got %q", config.TokenSourceConfig, info.Source)
	}
}

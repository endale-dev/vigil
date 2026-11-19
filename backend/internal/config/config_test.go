package config

import (
	"strings"
	"testing"
	"time"
)

func env(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := load(env(map[string]string{
		"APP_BASE_URL": "http://localhost:8080",
		"MONGO_URI":    "mongodb://localhost:27017",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppEnv != "development" || cfg.AppAddr != ":8080" || cfg.MongoDB != "vigil" {
		t.Fatalf("unexpected string defaults: %+v", cfg)
	}
	if cfg.CookieSecure || cfg.AllowPrivateTargets || cfg.TrustProxyHeaders {
		t.Fatalf("boolean defaults should be false: %+v", cfg)
	}
	if cfg.SessionTTL != 720*time.Hour || cfg.SchedulerTick != time.Second || cfg.CheckRetention != 720*time.Hour {
		t.Fatalf("unexpected duration defaults: %+v", cfg)
	}
	if cfg.CheckWorkers != 50 || cfg.FailureThreshold != 2 || cfg.MaxMonitorsPerUser != 25 || cfg.SMTPPort != 0 {
		t.Fatalf("unexpected numeric defaults: %+v", cfg)
	}
}

func TestLoadCookieSecureDefaultsAndOverride(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{name: "production default", env: map[string]string{"APP_ENV": "production", "APP_BASE_URL": "https://vigil.example", "MONGO_URI": "mongodb://localhost:27017", "SMTP_HOST": "mail.example", "SMTP_USER": "user", "SMTP_PASS": "pass", "SMTP_FROM": "vigil@example.com"}, want: true},
		{name: "development override", env: map[string]string{"APP_ENV": "development", "COOKIE_SECURE": "true", "APP_BASE_URL": "http://localhost:8080", "MONGO_URI": "mongodb://localhost:27017"}, want: true},
		{name: "production override", env: map[string]string{"APP_ENV": "production", "COOKIE_SECURE": "false", "APP_BASE_URL": "https://vigil.example", "MONGO_URI": "mongodb://localhost:27017", "SMTP_HOST": "mail.example", "SMTP_USER": "user", "SMTP_PASS": "pass", "SMTP_FROM": "vigil@example.com"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := load(env(tt.env))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.CookieSecure != tt.want {
				t.Fatalf("CookieSecure = %t, want %t", cfg.CookieSecure, tt.want)
			}
		})
	}
}

func TestLoadSMTPPortDefault(t *testing.T) {
	cfg, err := load(env(map[string]string{
		"APP_BASE_URL": "http://localhost:8080",
		"MONGO_URI":    "mongodb://localhost:27017",
		"SMTP_HOST":    "mail.example",
		"SMTP_USER":    "user",
		"SMTP_PASS":    "pass",
		"SMTP_FROM":    "vigil@example.com",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SMTPPort != 587 {
		t.Fatalf("SMTPPort = %d, want 587", cfg.SMTPPort)
	}
}

func TestLoadProductionRequirements(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr []string
	}{
		{
			name:    "production requires HTTPS and SMTP",
			env:     map[string]string{"APP_ENV": "production", "APP_BASE_URL": "http://vigil.example", "MONGO_URI": "mongodb://localhost:27017"},
			wantErr: []string{"APP_BASE_URL must use HTTPS", "SMTP_HOST is required"},
		},
		{
			name: "production accepts complete SMTP and HTTPS",
			env:  map[string]string{"APP_ENV": "production", "APP_BASE_URL": "https://vigil.example", "MONGO_URI": "mongodb://localhost:27017", "SMTP_HOST": "mail.example", "SMTP_USER": "user", "SMTP_PASS": "pass", "SMTP_FROM": "vigil@example.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := load(env(tt.env))
			if len(tt.wantErr) == 0 {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected configuration error")
			}
			for _, part := range tt.wantErr {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("error %q does not contain %q", err, part)
				}
			}
		})
	}
}

func TestLoadGoogleCredentialsMustBePaired(t *testing.T) {
	for _, values := range []map[string]string{
		{"GOOGLE_CLIENT_ID": "id"},
		{"GOOGLE_CLIENT_SECRET": "secret"},
	} {
		values["APP_BASE_URL"] = "http://localhost:8080"
		values["MONGO_URI"] = "mongodb://localhost:27017"
		_, err := load(env(values))
		if err == nil || !strings.Contains(err.Error(), "GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must be set together") {
			t.Fatalf("expected paired Google credential error, got %v", err)
		}
	}
}

func TestLoadReportsAllInvalidValues(t *testing.T) {
	_, err := load(env(map[string]string{
		"APP_ENV":               "test",
		"APP_ADDR":              ":bad",
		"APP_BASE_URL":          "localhost/path",
		"MONGO_URI":             "https://example.com",
		"MONGO_DB":              " ",
		"COOKIE_SECURE":         "sometimes",
		"SESSION_TTL":           "not-a-duration",
		"CHECK_WORKERS":         "0",
		"SCHEDULER_TICK":        "0s",
		"FAILURE_THRESHOLD":     "-1",
		"MAX_MONITORS_PER_USER": "x",
		"CHECK_RETENTION":       "-1h",
		"ALLOW_PRIVATE_TARGETS": "maybe",
		"TRUST_PROXY_HEADERS":   "maybe",
		"SMTP_HOST":             "mail.example",
		"SMTP_PORT":             "70000",
	}))
	if err == nil {
		t.Fatal("expected invalid configuration")
	}
	for _, key := range []string{"APP_ENV", "APP_ADDR", "APP_BASE_URL", "MONGO_URI", "MONGO_DB", "COOKIE_SECURE", "SESSION_TTL", "CHECK_WORKERS", "SCHEDULER_TICK", "FAILURE_THRESHOLD", "MAX_MONITORS_PER_USER", "CHECK_RETENTION", "ALLOW_PRIVATE_TARGETS", "TRUST_PROXY_HEADERS", "SMTP_PORT", "SMTP_USER"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("combined error %q does not contain %s", err, key)
		}
	}
}

func TestLoadRequiresPublicOriginAndMongoURI(t *testing.T) {
	_, err := load(env(nil))
	if err == nil {
		t.Fatal("expected required-value errors")
	}
	for _, key := range []string{"APP_BASE_URL is required", "MONGO_URI is required"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q does not contain %q", err, key)
		}
	}
}

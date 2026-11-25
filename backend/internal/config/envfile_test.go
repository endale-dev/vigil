package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvFileLoadsTelegramAndPreservesEnvironmentOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	contents := "# Local configuration\nAPP_BASE_URL=http://localhost:3000\nMONGO_URI='mongodb://localhost:27017'\nexport TELEGRAM_BOT_TOKEN=\"123:local-test-token\" # bot\nMONGO_DB=file_database\n"
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadWithEnvFile(path, env(map[string]string{"MONGO_DB": "environment_database"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TelegramBotToken != "123:local-test-token" || cfg.MongoDB != "environment_database" {
		t.Fatal("file token or environment override was not loaded")
	}
	cfg, err = loadWithEnvFile(path, env(map[string]string{"TELEGRAM_BOT_TOKEN": ""}))
	if err != nil || cfg.TelegramBotToken != "" {
		t.Fatal("explicit empty environment value must disable Telegram")
	}
}

func TestEnvFileMissingUsesEnvironment(t *testing.T) {
	_, err := loadWithEnvFile(filepath.Join(t.TempDir(), ".env"), env(map[string]string{"APP_BASE_URL": "http://localhost:3000", "MONGO_URI": "mongodb://localhost:27017"}))
	if err != nil {
		t.Fatal(err)
	}
}

func TestEnvFileErrorsDoNotExposeSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("TELEGRAM_BOT_TOKEN=\"SECRET_VALUE"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := loadWithEnvFile(path, env(nil))
	if err == nil || strings.Contains(err.Error(), "SECRET_VALUE") || !strings.Contains(err.Error(), "line 1") {
		t.Fatal("expected safe line-number diagnostic")
	}
}

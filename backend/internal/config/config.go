package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv              string
	AppAddr             string
	AppBaseURL          string
	MongoURI            string
	MongoDB             string
	CookieSecure        bool
	SessionTTL          time.Duration
	GoogleClientID      string
	GoogleClientSecret  string
	TelegramBotToken    string
	SMTPHost            string
	SMTPPort            int
	SMTPUser            string
	SMTPPass            string
	SMTPFrom            string
	CheckWorkers        int
	SchedulerTick       time.Duration
	FailureThreshold    int
	MaxMonitorsPerUser  int
	CheckRetention      time.Duration
	AllowPrivateTargets bool
	TrustProxyHeaders   bool
}

func Load() (Config, error) {
	return load(os.LookupEnv)
}

func load(lookup func(string) (string, bool)) (Config, error) {
	var problems []error
	value := func(key, fallback string) string {
		if got, ok := lookup(key); ok {
			return strings.TrimSpace(got)
		}
		return fallback
	}
	required := func(key string) string {
		got := value(key, "")
		if got == "" {
			problems = append(problems, fmt.Errorf("%s is required", key))
		}
		return got
	}
	parseBool := func(key string, fallback bool) bool {
		got := value(key, strconv.FormatBool(fallback))
		parsed, err := strconv.ParseBool(got)
		if err != nil {
			problems = append(problems, fmt.Errorf("%s must be a boolean", key))
			return fallback
		}
		return parsed
	}
	parseDuration := func(key, fallback string) time.Duration {
		got := value(key, fallback)
		parsed, err := time.ParseDuration(got)
		if err != nil || parsed <= 0 {
			problems = append(problems, fmt.Errorf("%s must be a positive duration", key))
			return 0
		}
		return parsed
	}
	parsePositiveInt := func(key, fallback string) int {
		got := value(key, fallback)
		parsed, err := strconv.Atoi(got)
		if err != nil || parsed <= 0 {
			problems = append(problems, fmt.Errorf("%s must be a positive integer", key))
			return 0
		}
		return parsed
	}

	cfg := Config{
		AppEnv:              value("APP_ENV", "development"),
		AppAddr:             value("APP_ADDR", ":8080"),
		AppBaseURL:          required("APP_BASE_URL"),
		MongoURI:            required("MONGO_URI"),
		MongoDB:             value("MONGO_DB", "vigil"),
		SessionTTL:          parseDuration("SESSION_TTL", "720h"),
		GoogleClientID:      value("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:  value("GOOGLE_CLIENT_SECRET", ""),
		TelegramBotToken:    value("TELEGRAM_BOT_TOKEN", ""),
		SMTPHost:            value("SMTP_HOST", ""),
		SMTPUser:            value("SMTP_USER", ""),
		SMTPPass:            value("SMTP_PASS", ""),
		SMTPFrom:            value("SMTP_FROM", ""),
		CheckWorkers:        parsePositiveInt("CHECK_WORKERS", "50"),
		SchedulerTick:       parseDuration("SCHEDULER_TICK", "1s"),
		FailureThreshold:    parsePositiveInt("FAILURE_THRESHOLD", "2"),
		MaxMonitorsPerUser:  parsePositiveInt("MAX_MONITORS_PER_USER", "25"),
		CheckRetention:      parseDuration("CHECK_RETENTION", "720h"),
		AllowPrivateTargets: parseBool("ALLOW_PRIVATE_TARGETS", false),
		TrustProxyHeaders:   parseBool("TRUST_PROXY_HEADERS", false),
	}

	if cfg.AppEnv != "development" && cfg.AppEnv != "production" {
		problems = append(problems, errors.New("APP_ENV must be development or production"))
	}
	_, appPort, addrErr := net.SplitHostPort(cfg.AppAddr)
	port, portErr := strconv.Atoi(appPort)
	if addrErr != nil || portErr != nil || port < 0 || port > 65535 {
		problems = append(problems, errors.New("APP_ADDR must be a valid host:port address"))
	}
	if cfg.AppBaseURL != "" {
		parsed, err := url.Parse(cfg.AppBaseURL)
		if err != nil || !parsed.IsAbs() || parsed.Hostname() == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			problems = append(problems, errors.New("APP_BASE_URL must be an absolute HTTP or HTTPS origin without a path, query, or fragment"))
		} else if cfg.AppEnv == "production" && parsed.Scheme != "https" {
			problems = append(problems, errors.New("APP_BASE_URL must use HTTPS in production"))
		}
	}
	if cfg.MongoURI != "" {
		parsed, err := url.Parse(cfg.MongoURI)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "mongodb" && parsed.Scheme != "mongodb+srv") {
			problems = append(problems, errors.New("MONGO_URI must be a valid MongoDB connection URI"))
		}
	}
	if cfg.MongoDB == "" {
		problems = append(problems, errors.New("MONGO_DB must not be empty"))
	}
	if _, ok := lookup("COOKIE_SECURE"); ok {
		cfg.CookieSecure = parseBool("COOKIE_SECURE", false)
	} else {
		cfg.CookieSecure = cfg.AppEnv == "production"
	}
	if (cfg.GoogleClientID == "") != (cfg.GoogleClientSecret == "") {
		problems = append(problems, errors.New("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must be set together"))
	}
	if cfg.SMTPHost != "" {
		cfg.SMTPPort = parsePositiveInt("SMTP_PORT", "587")
		if cfg.SMTPPort > 65535 {
			problems = append(problems, errors.New("SMTP_PORT must be between 1 and 65535"))
		}
		if cfg.SMTPUser == "" || cfg.SMTPPass == "" || cfg.SMTPFrom == "" {
			problems = append(problems, errors.New("SMTP_USER, SMTP_PASS, and SMTP_FROM are required when SMTP_HOST is set"))
		}
	} else {
		if _, ok := lookup("SMTP_PORT"); ok {
			port := value("SMTP_PORT", "")
			if port != "" {
				problems = append(problems, errors.New("SMTP_PORT requires SMTP_HOST"))
			}
		}
		if cfg.AppEnv == "production" {
			problems = append(problems, errors.New("SMTP_HOST is required in production"))
		}
	}

	if err := errors.Join(problems...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

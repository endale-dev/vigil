package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

func TestTelegramNotifierSendsDownAndRecoveryMessages(t *testing.T) {
	requests := make(chan struct {
		path string
		body struct {
			ChatID string `json:"chat_id"`
			Text   string `json:"text"`
		}
	}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var body struct {
			ChatID string `json:"chat_id"`
			Text   string `json:"text"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		requests <- struct {
			path string
			body struct {
				ChatID string `json:"chat_id"`
				Text   string `json:"text"`
			}
		}{path: request.URL.Path, body: body}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewTelegramNotifier("12345:BOT_SECRET")
	notifier.endpoint = server.URL
	user := domain.User{Notify: domain.NotifySettings{TelegramChatID: "chat-42"}}
	if notifier.Name() != domain.ChannelTelegram || !notifier.Enabled(user) {
		t.Fatal("Telegram notifier name or enabled state is incorrect")
	}
	alerts := []struct {
		alert service.Alert
		want  string
	}{
		{
			alert: service.Alert{Monitor: domain.Monitor{Name: "API", Target: "https://example.test/health"}, Status: domain.StatusDown, Detail: "HTTP 503"},
			want:  "🔴 DOWN — API (https://example.test/health)\nHTTP 503",
		},
		{
			alert: service.Alert{Monitor: domain.Monitor{Name: "API"}, Status: domain.StatusUp, Detail: "HTTP 200"},
			want:  "🟢 RECOVERED — API\nHTTP 200",
		},
	}
	for _, test := range alerts {
		if err := notifier.Notify(context.Background(), user, test.alert); err != nil {
			t.Fatalf("Notify() error = %v", err)
		}
		request := <-requests
		if request.path != "/bot12345:BOT_SECRET/sendMessage" || request.body.ChatID != user.Notify.TelegramChatID || request.body.Text != test.want {
			t.Fatalf("request = %+v, want token endpoint, chat ID, and text %q", request, test.want)
		}
	}
}

func TestTelegramNotifierSkipsDisabledAndUnknownAlerts(t *testing.T) {
	called := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called <- struct{}{} }))
	defer server.Close()
	notifier := NewTelegramNotifier("token")
	notifier.endpoint = server.URL

	if NewTelegramNotifier("").Enabled(domain.User{Notify: domain.NotifySettings{TelegramChatID: "chat"}}) {
		t.Fatal("notifier enabled without a bot token")
	}
	if notifier.Enabled(domain.User{}) {
		t.Fatal("notifier enabled without a chat ID")
	}
	if err := notifier.Notify(context.Background(), domain.User{}, service.Alert{Status: domain.StatusDown}); err != nil {
		t.Fatalf("disabled Notify() error = %v", err)
	}
	user := domain.User{Notify: domain.NotifySettings{TelegramChatID: "chat"}}
	if err := notifier.Notify(context.Background(), user, service.Alert{Status: domain.StatusUnknown}); err != nil {
		t.Fatalf("unknown alert Notify() error = %v", err)
	}
	select {
	case <-called:
		t.Fatal("disabled or unknown alert made an HTTP request")
	default:
	}
}

func TestTelegramNotifierSanitizesHTTPFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "failed for https://api.telegram.org/bot12345:BOT_SECRET/sendMessage", http.StatusBadGateway)
	}))
	defer server.Close()
	notifier := NewTelegramNotifier("12345:BOT_SECRET")
	notifier.endpoint = server.URL
	user := domain.User{Notify: domain.NotifySettings{TelegramChatID: "chat"}}
	err := notifier.Notify(context.Background(), user, service.Alert{
		Monitor: domain.Monitor{Name: "API"}, Status: domain.StatusDown,
	})
	if err == nil || strings.Contains(err.Error(), "BOT_SECRET") || strings.Contains(err.Error(), "api.telegram.org") {
		t.Fatalf("Notify() error = %v, want sanitized status error", err)
	}
}

func TestTelegramNotifierHonorsRequestCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
		case <-time.After(100 * time.Millisecond):
		}
	}))
	defer server.Close()
	notifier := NewTelegramNotifier("12345:BOT_SECRET")
	notifier.endpoint = server.URL
	user := domain.User{Notify: domain.NotifySettings{TelegramChatID: "chat"}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := notifier.Notify(ctx, user, service.Alert{Monitor: domain.Monitor{Name: "API"}, Status: domain.StatusDown})
	if err != context.DeadlineExceeded || strings.Contains(err.Error(), "BOT_SECRET") {
		t.Fatalf("Notify() error = %v, want context deadline without token", err)
	}
}

package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

const telegramAPIEndpoint = "https://api.telegram.org"

type TelegramNotifier struct {
	links    telegramLinks
	botToken string
	endpoint string
	client   *http.Client
}

func NewTelegramNotifier(botToken string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		endpoint: telegramAPIEndpoint,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (notifier *TelegramNotifier) Name() domain.Channel {
	return domain.ChannelTelegram
}

func (notifier *TelegramNotifier) Enabled(user domain.User) bool {
	return notifier.botToken != "" && user.Notify.TelegramChatID != "" && !user.Notify.TelegramDisabled
}

func (notifier *TelegramNotifier) Notify(ctx context.Context, user domain.User, alert service.Alert) error {
	if !notifier.Enabled(user) {
		return nil
	}
	message, ok := telegramMessage(alert)
	if !ok {
		return nil
	}
	body, err := json.Marshal(struct {
		ChatID string `json:"chat_id"`
		Text   string `json:"text"`
	}{ChatID: user.Notify.TelegramChatID, Text: message})
	if err != nil {
		return errors.New("telegram notification failed")
	}
	endpoint := strings.TrimRight(notifier.endpoint, "/") + "/bot" + notifier.botToken + "/sendMessage"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return errors.New("telegram notification failed")
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := notifier.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("telegram notification failed")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("telegram notification failed with status %d", response.StatusCode)
	}
	return nil
}

func telegramMessage(alert service.Alert) (string, bool) {
	switch alert.Status {
	case domain.StatusDown:
		return fmt.Sprintf("🔴 DOWN — %s (%s)\n%s", alert.Monitor.Name, alert.Monitor.Target, alert.Detail), true
	case domain.StatusUp:
		return fmt.Sprintf("🟢 RECOVERED — %s\n%s", alert.Monitor.Name, alert.Detail), true
	default:
		return "", false
	}
}

var _ service.Notifier = (*TelegramNotifier)(nil)

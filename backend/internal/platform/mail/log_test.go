package mail

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/endale-dev/vigil/internal/service"
)

func TestLogMailerLogsMessageAtInfo(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	mailer := NewLogMailer(logger)
	message := service.Mail{To: "user@example.test", Subject: "Welcome", Body: "Hello from Vigil"}
	if err := mailer.Send(context.Background(), message); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	for _, want := range []string{"level=INFO", "development email", message.To, message.Subject, message.Body} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("log output %q does not contain %q", output.String(), want)
		}
	}
}

func TestLogMailerHonorsCanceledContext(t *testing.T) {
	var output bytes.Buffer
	mailer := NewLogMailer(slog.New(slog.NewTextHandler(&output, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := mailer.Send(ctx, service.Mail{To: "user@example.test"}); err == nil {
		t.Fatal("Send() returned no error for canceled context")
	}
	if output.Len() != 0 {
		t.Fatalf("canceled email was logged: %q", output.String())
	}
}

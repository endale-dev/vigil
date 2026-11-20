package mail

import (
	"context"
	"log/slog"

	"github.com/endale-dev/vigil/internal/service"
)

type LogMailer struct {
	logger *slog.Logger
}

func NewLogMailer(logger *slog.Logger) *LogMailer {
	return &LogMailer{logger: logger}
}

func (mailer *LogMailer) Send(ctx context.Context, message service.Mail) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	mailer.logger.InfoContext(ctx, "development email", "to", message.To, "subject", message.Subject, "body", message.Body)
	return nil
}

var _ service.Mailer = (*LogMailer)(nil)

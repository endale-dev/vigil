package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/endale-dev/vigil/internal/service"
)

type SMTPMailer struct {
	host      string
	address   string
	username  string
	password  string
	from      string
	tlsConfig *tls.Config
}

func NewSMTPMailer(host string, port int, username, password, from string) *SMTPMailer {
	return &SMTPMailer{
		host:     host,
		address:  net.JoinHostPort(host, strconv.Itoa(port)),
		username: username,
		password: password,
		from:     from,
	}
}

func (mailer *SMTPMailer) Send(ctx context.Context, message service.Mail) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", mailer.address)
	if err != nil {
		return sanitizedSMTPError(ctx)
	}
	defer connection.Close()
	stopCancellation := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stopCancellation()
	if deadline, ok := ctx.Deadline(); ok {
		if err := connection.SetDeadline(deadline); err != nil {
			return sanitizedSMTPError(ctx)
		}
	}

	client, err := smtp.NewClient(connection, mailer.host)
	if err != nil {
		return sanitizedSMTPError(ctx)
	}
	defer client.Close()
	if supported, _ := client.Extension("STARTTLS"); !supported {
		return errors.New("SMTP server does not support STARTTLS")
	}
	if err := client.StartTLS(mailer.startTLSConfig()); err != nil {
		return sanitizedSMTPError(ctx)
	}
	if err := client.Auth(smtp.PlainAuth("", mailer.username, mailer.password, mailer.host)); err != nil {
		return sanitizedSMTPError(ctx)
	}

	from := stripHeaderBreaks(mailer.from)
	to := stripHeaderBreaks(message.To)
	if err := client.Mail(from); err != nil {
		return sanitizedSMTPError(ctx)
	}
	if err := client.Rcpt(to); err != nil {
		return sanitizedSMTPError(ctx)
	}
	writer, err := client.Data()
	if err != nil {
		return sanitizedSMTPError(ctx)
	}
	if _, err := io.WriteString(writer, formatMessage(from, to, message.Subject, message.Body)); err != nil {
		_ = writer.Close()
		return sanitizedSMTPError(ctx)
	}
	if err := writer.Close(); err != nil {
		return sanitizedSMTPError(ctx)
	}
	if err := client.Quit(); err != nil {
		return sanitizedSMTPError(ctx)
	}
	return nil
}

func (mailer *SMTPMailer) startTLSConfig() *tls.Config {
	config := &tls.Config{ServerName: mailer.host}
	if mailer.tlsConfig != nil {
		config = mailer.tlsConfig.Clone()
		config.ServerName = mailer.host
	}
	config.InsecureSkipVerify = false
	return config
}

func formatMessage(from, to, subject, body string) string {
	subject = mime.QEncoding.Encode("UTF-8", stripHeaderBreaks(subject))
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")
	body = strings.ReplaceAll(body, "\n", "\r\n")
	return fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		stripHeaderBreaks(from), stripHeaderBreaks(to), subject, body,
	)
}

func stripHeaderBreaks(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, value)
}

func sanitizedSMTPError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return errors.New("email delivery failed")
}

var _ service.Mailer = (*SMTPMailer)(nil)

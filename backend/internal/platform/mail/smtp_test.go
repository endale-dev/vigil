package mail

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/service"
)

func TestSMTPMailerUsesStartTLSAndSanitizesHeaders(t *testing.T) {
	serverName, certificate, roots := testSMTPServerCertificate(t)
	address, serverResult, messageResult := startSMTPTestServer(t, certificate)
	mailer := NewSMTPMailer(serverName, 587, "smtp-user", "smtp-secret", "sender@example.test")
	mailer.address = address
	mailer.tlsConfig = &tls.Config{RootCAs: roots}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := mailer.Send(ctx, service.Mail{
		To:      "user@example.test\r\nBcc: injected@example.test",
		Subject: "Hello\r\nBcc: injected@example.test",
		Body:    "first line\n.second line",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("SMTP server error = %v", err)
	}
	message := <-messageResult
	if !strings.Contains(message, "From: sender@example.test\r\n") ||
		!strings.Contains(message, "To: user@example.testBcc: injected@example.test\r\n") ||
		!strings.Contains(message, "Subject: HelloBcc: injected@example.test\r\n") ||
		!strings.Contains(message, "Content-Type: text/plain; charset=UTF-8\r\n") {
		t.Fatalf("message headers were not formed and sanitized: %q", message)
	}
	if strings.Contains(message, "\r\nBcc:") {
		t.Fatalf("recipient or subject injected a header: %q", message)
	}
	if !strings.Contains(message, "first line\r\n.second line") {
		t.Fatalf("body line endings or dot escaping are incorrect: %q", message)
	}
}

func TestSMTPMailerSanitizesTransportAndTLSFailures(t *testing.T) {
	serverName, certificate, _ := testSMTPServerCertificate(t)
	address, serverResult, _ := startSMTPTestServer(t, certificate)
	mailer := NewSMTPMailer(serverName, 587, "smtp-user", "smtp-secret", "sender@example.test")
	mailer.address = address
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mailer.Send(ctx, service.Mail{To: "user@example.test", Subject: "Test"})
	if err == nil || strings.Contains(err.Error(), "smtp-secret") || strings.Contains(err.Error(), serverName) {
		t.Fatalf("untrusted TLS error = %v, want sanitized failure", err)
	}
	if err := <-serverResult; err == nil {
		t.Fatal("test SMTP server expected client certificate verification to fail")
	}
}

func TestSMTPMailerReturnsContextCancellation(t *testing.T) {
	mailer := NewSMTPMailer("smtp.invalid", 587, "user", "secret", "from@example.test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := mailer.Send(ctx, service.Mail{}); err != context.Canceled {
		t.Fatalf("Send() error = %v, want context.Canceled", err)
	}
}

func TestFormatMessageUsesUTF8TextAndStripsHeaderBreaks(t *testing.T) {
	message := formatMessage("from@example.test", "to@example.test", "hello\r\nBcc: x@example.test", "a\nb")
	if strings.Contains(message, "\r\nBcc:") {
		t.Fatalf("header break was not removed: %q", message)
	}
	if !strings.Contains(message, "Content-Type: text/plain; charset=UTF-8\r\n") {
		t.Fatalf("missing UTF-8 content type: %q", message)
	}
}

func testSMTPServerCertificate(t *testing.T) (string, tls.Certificate, *x509.CertPool) {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(server.Close)
	certificate := server.Certificate()
	if len(certificate.DNSNames) == 0 {
		t.Fatal("test server certificate has no DNS name")
	}
	roots := x509.NewCertPool()
	roots.AddCert(certificate)
	return certificate.DNSNames[0], server.TLS.Certificates[0], roots
}

func startSMTPTestServer(t *testing.T, certificate tls.Certificate) (string, <-chan error, <-chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	serverResult := make(chan error, 1)
	messageResult := make(chan string, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverResult <- err
			return
		}
		defer connection.Close()
		_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
		message, err := serveTestSMTP(connection, certificate)
		if err != nil {
			serverResult <- err
			return
		}
		messageResult <- message
		serverResult <- nil
	}()
	return listener.Addr().String(), serverResult, messageResult
}

func serveTestSMTP(connection net.Conn, certificate tls.Certificate) (string, error) {
	reader := bufio.NewReader(connection)
	writer := bufio.NewWriter(connection)
	write := func(line string) error {
		if _, err := io.WriteString(writer, line+"\r\n"); err != nil {
			return err
		}
		return writer.Flush()
	}
	read := func() (string, error) {
		line, err := reader.ReadString('\n')
		return strings.TrimRight(line, "\r\n"), err
	}
	if err := write("220 smtp.test ESMTP"); err != nil {
		return "", err
	}
	if line, err := read(); err != nil || !strings.HasPrefix(line, "EHLO ") {
		return "", fmt.Errorf("initial EHLO = %q: %w", line, err)
	}
	if err := write("250-smtp.test"); err != nil {
		return "", err
	}
	if err := write("250-STARTTLS"); err != nil {
		return "", err
	}
	if err := write("250 SIZE 100000"); err != nil {
		return "", err
	}
	if line, err := read(); err != nil || line != "STARTTLS" {
		return "", fmt.Errorf("STARTTLS command = %q: %w", line, err)
	}
	if err := write("220 begin TLS"); err != nil {
		return "", err
	}
	tlsConnection := tls.Server(connection, &tls.Config{Certificates: []tls.Certificate{certificate}})
	if err := tlsConnection.Handshake(); err != nil {
		return "", err
	}
	reader = bufio.NewReader(tlsConnection)
	writer = bufio.NewWriter(tlsConnection)
	if line, err := read(); err != nil || !strings.HasPrefix(line, "EHLO ") {
		return "", fmt.Errorf("post-TLS EHLO = %q: %w", line, err)
	}
	if err := write("250-smtp.test"); err != nil {
		return "", err
	}
	if err := write("250 AUTH PLAIN"); err != nil {
		return "", err
	}
	line, err := read()
	if err != nil || !strings.HasPrefix(line, "AUTH PLAIN ") {
		return "", fmt.Errorf("AUTH command = %q: %w", line, err)
	}
	credentials, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "AUTH PLAIN "))
	if err != nil || string(credentials) != "\x00smtp-user\x00smtp-secret" {
		return "", fmt.Errorf("unexpected SMTP credentials")
	}
	if err := write("235 authenticated"); err != nil {
		return "", err
	}
	if line, err := read(); err != nil || !strings.HasPrefix(line, "MAIL FROM:<sender@example.test>") {
		return "", fmt.Errorf("MAIL command = %q: %w", line, err)
	}
	if err := write("250 sender accepted"); err != nil {
		return "", err
	}
	if line, err := read(); err != nil || !strings.HasPrefix(line, "RCPT TO:<user@example.testBcc: injected@example.test>") {
		return "", fmt.Errorf("RCPT command = %q: %w", line, err)
	}
	if err := write("250 recipient accepted"); err != nil {
		return "", err
	}
	if line, err := read(); err != nil || line != "DATA" {
		return "", fmt.Errorf("DATA command = %q: %w", line, err)
	}
	if err := write("354 send message"); err != nil {
		return "", err
	}
	var message strings.Builder
	for {
		line, err := read()
		if err != nil {
			return "", err
		}
		if line == "." {
			break
		}
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}
		message.WriteString(line)
		message.WriteString("\r\n")
	}
	if err := write("250 message accepted"); err != nil {
		return "", err
	}
	if line, err := read(); err != nil || line != "QUIT" {
		return "", fmt.Errorf("QUIT command = %q: %w", line, err)
	}
	if err := write("221 closing connection"); err != nil {
		return "", err
	}
	return message.String(), nil
}

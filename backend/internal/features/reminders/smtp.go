package reminders

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type SMTPOptions struct {
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
	FromName    string
	RequireTLS  bool
}

type SMTPProvider struct {
	address    string
	host       string
	auth       smtp.Auth
	from       mail.Address
	envelope   string
	requireTLS bool
}

func NewSMTPProvider(options SMTPOptions) (*SMTPProvider, error) {
	from, err := mail.ParseAddress(options.FromAddress)
	if err != nil {
		return nil, fmt.Errorf("parse SMTP from address: %w", err)
	}
	if strings.ContainsAny(options.Host, "\r\n") || options.Host == "" || options.Port < 1 || options.Port > 65535 {
		return nil, errors.New("invalid SMTP host or port")
	}
	if options.FromName != "" {
		from.Name = options.FromName
	}
	var auth smtp.Auth
	if options.Username != "" {
		auth = smtp.PlainAuth("", options.Username, options.Password, options.Host)
	}
	return &SMTPProvider{
		address:    options.Host + ":" + strconv.Itoa(options.Port),
		host:       options.Host,
		auth:       auth,
		from:       *from,
		envelope:   from.Address,
		requireTLS: options.RequireTLS,
	}, nil
}

func (p *SMTPProvider) Send(ctx context.Context, message EmailMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	recipient, err := mail.ParseAddress(message.To)
	if err != nil {
		return "", fmt.Errorf("parse recipient: %w", err)
	}
	messageID := message.MessageID
	if messageID == "" {
		messageID, err = newMessageID()
		if err != nil {
			return "", fmt.Errorf("create message ID: %w", err)
		}
	}
	if strings.ContainsAny(messageID, "\r\n") || !strings.HasPrefix(messageID, "<") || !strings.HasSuffix(messageID, ">") {
		return "", errors.New("invalid message ID")
	}
	body := strings.ReplaceAll(message.Body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\n", "\r\n")
	raw := strings.Join([]string{
		"From: " + p.from.String(),
		"To: " + recipient.String(),
		"Subject: " + mime.QEncoding.Encode("UTF-8", sanitizeHeader(message.Subject)),
		"Message-ID: " + messageID,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", p.address)
	if err != nil {
		return "", fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer connection.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	} else {
		_ = connection.SetDeadline(time.Now().Add(30 * time.Second))
	}
	client, err := smtp.NewClient(connection, p.host)
	if err != nil {
		return "", fmt.Errorf("start SMTP session: %w", err)
	}
	defer client.Close()
	if supported, _ := client.Extension("STARTTLS"); supported {
		if err := client.StartTLS(&tls.Config{ServerName: p.host, MinVersion: tls.VersionTLS12}); err != nil {
			return "", fmt.Errorf("secure SMTP session: %w", err)
		}
	} else if p.requireTLS {
		return "", errors.New("SMTP server does not offer STARTTLS")
	}
	if p.auth != nil {
		if err := client.Auth(p.auth); err != nil {
			return "", fmt.Errorf("authenticate with SMTP server: %w", err)
		}
	}
	if err := client.Mail(p.envelope); err != nil {
		return "", fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(recipient.Address); err != nil {
		return "", fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return "", fmt.Errorf("open SMTP message: %w", err)
	}
	if _, err := writer.Write([]byte(raw)); err != nil {
		_ = writer.Close()
		return "", fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("finish SMTP message: %w", err)
	}
	_ = client.Quit()
	return messageID, nil
}

func sanitizeHeader(value string) string {
	return strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ").Replace(value))
}

func newMessageID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "<" + hex.EncodeToString(value) + "@ledgerops.local>", nil
}

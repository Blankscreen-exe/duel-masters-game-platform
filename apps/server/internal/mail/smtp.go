package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// sendTimeout is the longest one email may take, from connecting to the server
// accepting it.
const sendTimeout = 15 * time.Second

// dialer connects to a mail server, encrypted when the settings say so from the
// start. It is a field of Service so tests can trust their own certificate.
type dialer func(ctx context.Context, settings Settings) (net.Conn, *tls.Config, error)

// dialServer connects to the real mail server, checking its certificate.
func dialServer(ctx context.Context, settings Settings) (net.Conn, *tls.Config, error) {
	config := &tls.Config{ServerName: settings.Host, MinVersion: tls.VersionTLS12}
	address := net.JoinHostPort(settings.Host, strconv.Itoa(settings.Port))

	if settings.Security == ImplicitTLS {
		conn, err := (&tls.Dialer{Config: config}).DialContext(ctx, "tcp", address)
		return conn, config, err
	}

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	return conn, config, err
}

// Send sends one plain text email. It returns ErrNotConfigured before an admin
// has set up a mail server, and ErrUnreadable while the stored settings cannot
// be read.
func (s *Service) Send(ctx context.Context, to, subject, body string) error {
	settings, _, found, err := s.read(ctx, s.db)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotConfigured
	}
	if err := CheckAddress(to); err != nil {
		return fmt.Errorf("sending to %q: %w", to, err)
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	message, err := composeMessage(settings, to, subject, body, time.Now())
	if err != nil {
		return err
	}

	return s.deliver(ctx, settings, to, message)
}

// deliver hands a message to the mail server.
func (s *Service) deliver(ctx context.Context, settings Settings, to string, message []byte) error {
	conn, config, err := s.dial(ctx, settings)
	if err != nil {
		return fmt.Errorf("connecting to the mail server: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return fmt.Errorf("connecting to the mail server: %w", err)
		}
	}

	client, err := smtp.NewClient(conn, settings.Host)
	if err != nil {
		return fmt.Errorf("talking to the mail server: %w", err)
	}
	defer func() { _ = client.Close() }()

	if settings.Security == StartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("the mail server does not offer STARTTLS, so the connection would not be encrypted")
		}
		if err := client.StartTLS(config); err != nil {
			return fmt.Errorf("encrypting the connection to the mail server: %w", err)
		}
	}

	if settings.Username != "" || settings.Password != "" {
		if err := client.Auth(authFor(client, settings)); err != nil {
			return fmt.Errorf("logging in to the mail server: %w", err)
		}
	}

	if err := client.Mail(settings.FromAddress); err != nil {
		return fmt.Errorf("the mail server refused the sender: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("the mail server refused the recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("the mail server refused the message: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		return fmt.Errorf("writing the message to the mail server: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("the mail server refused the message: %w", err)
	}

	return client.Quit()
}

// authFor picks how to log in: PLAIN where the server offers it, otherwise
// LOGIN, the only one some servers offer. Both refuse to send the password over
// an unencrypted connection, except to this machine.
func authFor(client *smtp.Client, settings Settings) smtp.Auth {
	plain := smtp.PlainAuth("", settings.Username, settings.Password, settings.Host)

	if ok, mechanisms := client.Extension("AUTH"); ok && !offers(mechanisms, "PLAIN") && offers(mechanisms, "LOGIN") {
		return &loginAuth{plain: plain, username: settings.Username, password: settings.Password}
	}

	return plain
}

func offers(mechanisms, wanted string) bool {
	for _, mechanism := range strings.Fields(mechanisms) {
		if strings.EqualFold(mechanism, wanted) {
			return true
		}
	}

	return false
}

// loginAuth is the LOGIN way of logging in, which net/smtp does not have.
type loginAuth struct {
	// plain makes the same refusal to send a password in the clear.
	plain              smtp.Auth
	username, password string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	// PLAIN's check of the connection is all that is wanted from it. It looks
	// at whether the server offers PLAIN too, and this one does not, so it is
	// told it does.
	checked := *server
	checked.Auth = append([]string{"PLAIN"}, server.Auth...)
	if _, _, err := a.plain.Start(&checked); err != nil {
		return "", nil, err
	}

	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(prompt []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}

	switch strings.ToLower(strings.TrimSpace(string(prompt))) {
	case "username:":
		return []byte(a.username), nil
	case "password:":
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("the mail server asked for %q while logging in", prompt)
	}
}

// composeMessage writes a plain text email with its headers.
func composeMessage(settings Settings, to, subject, body string, now time.Time) ([]byte, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("making a message id: %w", err)
	}

	from := mail.Address{Name: settings.FromName, Address: settings.FromAddress}
	_, domain, _ := strings.Cut(settings.FromAddress, "@")

	var message bytes.Buffer
	for _, header := range [][2]string{
		{"From", from.String()},
		{"To", (&mail.Address{Address: to}).String()},
		{"Subject", mime.QEncoding.Encode("utf-8", subject)},
		{"Date", now.UTC().Format(time.RFC1123Z)},
		{"Message-ID", fmt.Sprintf("<%s@%s>", id, domain)},
		{"MIME-Version", "1.0"},
		{"Content-Type", `text/plain; charset="utf-8"`},
		{"Content-Transfer-Encoding", "8bit"},
	} {
		message.WriteString(header[0] + ": " + header[1] + "\r\n")
	}
	message.WriteString("\r\n")
	message.WriteString(strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n"))
	message.WriteString("\r\n")

	return message.Bytes(), nil
}

// Package mailtest is a small SMTP server for tests, so the real sending code is
// what they exercise.
package mailtest

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// Message is one email the server accepted.
type Message struct {
	From string
	To   string
	Data string
}

// Options say what the server offers.
type Options struct {
	// Username and Password are the login it wants; empty for none.
	Username, Password string
	// Auth is the login mechanisms it offers; "PLAIN LOGIN" when empty.
	Auth string
	// StartTLS offers upgrading the connection.
	StartTLS bool
	// ImplicitTLS encrypts every connection from the start.
	ImplicitTLS bool
	// RefuseRecipient refuses every recipient.
	RefuseRecipient bool
}

// Server is a running test mail server.
type Server struct {
	// Port is the port it listens on, on 127.0.0.1.
	Port int
	// Roots trusts its certificate.
	Roots *x509.CertPool

	options  Options
	listener net.Listener
	tls      *tls.Config

	mu       sync.Mutex
	messages []Message
	wg       sync.WaitGroup
}

// Start runs a server until the test ends.
func Start(t *testing.T, options Options) *Server {
	t.Helper()

	if options.Auth == "" {
		options.Auth = "PLAIN LOGIN"
	}

	certificate, roots := selfSigned(t)
	config := &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("starting the test mail server: %v", err)
	}
	if options.ImplicitTLS {
		listener = tls.NewListener(listener, config)
	}

	server := &Server{
		Port:     listener.Addr().(*net.TCPAddr).Port,
		Roots:    roots,
		options:  options,
		listener: listener,
		tls:      config,
	}
	server.wg.Add(1)
	go server.serve()

	t.Cleanup(func() {
		_ = listener.Close()
		server.wg.Wait()
	})

	return server
}

// Messages returns the emails accepted so far.
func (s *Server) Messages() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]Message(nil), s.messages...)
}

func (s *Server) serve() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() { _ = conn.Close() }()
			_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
			s.handle(conn)
		}()
	}
}

// session is the state of one connection.
type session struct {
	reader    *bufio.Reader
	conn      net.Conn
	encrypted bool
	loggedIn  bool
	from, to  string
}

func (s *session) reply(line string) {
	_, _ = s.conn.Write([]byte(line + "\r\n"))
}

func (s *Server) handle(conn net.Conn) {
	current := &session{reader: bufio.NewReader(conn), conn: conn, encrypted: s.options.ImplicitTLS}
	current.reply("220 test mail server")

	for {
		line, err := current.reader.ReadString('\n')
		if err != nil {
			return
		}

		command, argument, _ := strings.Cut(strings.TrimRight(line, "\r\n"), " ")
		switch strings.ToUpper(command) {
		case "EHLO", "HELO":
			s.greet(current)
		case "STARTTLS":
			if !s.options.StartTLS {
				current.reply("502 not offered")
				continue
			}
			current.reply("220 go ahead")
			upgraded := tls.Server(current.conn, s.tls)
			if upgraded.Handshake() != nil {
				return
			}
			current.conn = upgraded
			current.reader = bufio.NewReader(upgraded)
			current.encrypted = true
		case "AUTH":
			s.logIn(current, argument)
		case "MAIL":
			if s.options.Username != "" && !current.loggedIn {
				current.reply("530 log in first")
				continue
			}
			current.from = addressIn(argument)
			current.reply("250 ok")
		case "RCPT":
			if s.options.RefuseRecipient {
				current.reply("550 no such user")
				continue
			}
			current.to = addressIn(argument)
			current.reply("250 ok")
		case "DATA":
			current.reply("354 go ahead")
			s.readMessage(current)
		case "QUIT":
			current.reply("221 bye")
			return
		default:
			current.reply("500 unknown command")
		}
	}
}

func (s *Server) greet(current *session) {
	lines := []string{"test mail server"}
	if s.options.StartTLS && !current.encrypted {
		lines = append(lines, "STARTTLS")
	}
	lines = append(lines, "AUTH "+s.options.Auth)

	for i, line := range lines {
		separator := "-"
		if i == len(lines)-1 {
			separator = " "
		}
		current.reply("250" + separator + line)
	}
}

func (s *Server) logIn(current *session, argument string) {
	mechanism, initial, _ := strings.Cut(argument, " ")

	var username, password string
	switch strings.ToUpper(mechanism) {
	case "PLAIN":
		decoded, _ := base64.StdEncoding.DecodeString(initial)
		parts := strings.Split(string(decoded), "\x00")
		if len(parts) == 3 {
			username, password = parts[1], parts[2]
		}
	case "LOGIN":
		username = s.ask(current, "Username:")
		password = s.ask(current, "Password:")
	default:
		current.reply("504 unknown mechanism")
		return
	}

	if username != s.options.Username || password != s.options.Password {
		current.reply("535 wrong login")
		return
	}

	current.loggedIn = true
	current.reply("235 logged in")
}

func (s *Server) ask(current *session, prompt string) string {
	current.reply("334 " + base64.StdEncoding.EncodeToString([]byte(prompt)))
	line, err := current.reader.ReadString('\n')
	if err != nil {
		return ""
	}
	decoded, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(line))
	return string(decoded)
}

func (s *Server) readMessage(current *session) {
	var data strings.Builder
	for {
		line, err := current.reader.ReadString('\n')
		if err != nil {
			return
		}
		if strings.TrimRight(line, "\r\n") == "." {
			break
		}
		data.WriteString(strings.TrimPrefix(line, "."))
	}

	s.mu.Lock()
	s.messages = append(s.messages, Message{From: current.from, To: current.to, Data: data.String()})
	s.mu.Unlock()
	current.reply("250 accepted")
}

// addressIn finds the address in "FROM:<name@example.com> BODY=8BITMIME".
func addressIn(argument string) string {
	_, rest, _ := strings.Cut(argument, "<")
	address, _, _ := strings.Cut(rest, ">")
	return address
}

// selfSigned makes a certificate for 127.0.0.1 and a pool trusting it.
func selfSigned(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("making a key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test mail server"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},

		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("making a certificate: %v", err)
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("reading a certificate: %v", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(parsed)

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, roots
}

// smtp_proxy.go - mix network client smtp submission proxy
// Copyright (C) 2017  David Anthony Stainton
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// Package util provides client utilities
package util

import (
	"bytes"
	"io"
	"net"
	"net/mail"

	"github.com/katzenpost/core/interfaces"
	"github.com/katzenpost/core/wire"
	"github.com/op/go-logging"
	"github.com/siebenmann/smtpd"
)

// logWriter is used to present the io.Reader interface
// to our SMTP library for logging. this is only required
// because of our SMTP library choice and isn't otherwise needed.
type logWriter struct {
	log *logging.Logger
}

// newLogWriter creates a new logWriter
func newLogWriter(log *logging.Logger) *logWriter {
	writer := logWriter{
		log: log,
	}
	return &writer
}

// Write writes a message to the log
func (w *logWriter) Write(p []byte) (int, error) {
	w.log.Debug(string(p))
	return len(p), nil
}

// SubmitProxy handles SMTP mail submissions. This means implementing
// Client to Provider protocol reliability as well as three layers of
// crypto:
//
//    * link layer / sphinx layer / end to end layer
//
// Note: the end to end crypto is Client to Client while the Provider
// participates in our reliability protocol receiving ciphertext on behalf
// of the recipient.
//
// The outgoing messages are persisted to disk in a cryptographically sealed file vault.
// If the client stops operating before receiving the corresponding ACK message,
// the client will later be able to retreive messages from disk and retransmit them.
type SubmitProxy struct {
	// Authenticator is an implementation of the wire.PeerAuthenticator interface,
	// in this case it's used by the client to authenticate the Provider using our
	// noise based wire protocol authentication command
	authenticator wire.PeerAuthenticator

	// RandomReader is an implementation of the io.Reader interface
	// which is used to generate ephemeral keys for our wire protocol's
	// cryptographic handshake messages
	randomReader io.Reader

	// userPKI implements the UserPKI interface
	userPKI interfaces.UserPKI

	// mixPKI implements the MixPKI interface
	mixPKI interfaces.MixPKI
}

// NewSubmitProxy
func NewSubmitProxy(authenticator wire.PeerAuthenticator, randomReader io.Reader, userPki interfaces.UserPKI, mixPki interfaces.MixPKI) *SubmitProxy {
	submissionProxy := SubmitProxy{
		authenticator: authenticator,
		randomReader:  randomReader,
		userPKI:       userPki,
		mixPKI:        mixPki,
	}
	return &submissionProxy
}

// sendMessage sends a message to a given receiver identity
//
// The sender e-mail address is used to select the client's end to end cryptographic
// identity in sending end to end messages, ACK messaging queuing identification on the provider
// and link layer authentication with the associated Provider mixnet service.
//
// TODO: implement the Stop and Wait ARQ protocol scheme here!
func (p *SubmitProxy) sendMessage(sender, receiver string, message []byte) error {
	return nil
}

// handleSMTPSubmission handles the SMTP submissions
// and proxies them to the mix network.
// TODO:
// 1. reject/abort upon invalid SMTP rcpt to:
//    and invalid mail from:
// 2. properly parse From and To fields
// 3. lowercase the sender and receiver string if using
//    them as keys to a map
func (p *SubmitProxy) handleSMTPSubmission(conn net.Conn) error {
	cfg := smtpd.Config{} // XXX
	logWriter := newLogWriter(log)
	smtpConn := smtpd.NewConn(conn, cfg, logWriter)
	for {
		event := smtpConn.Next()
		if event.What == smtpd.DONE || event.What == smtpd.ABORT {
			return nil
		}
		// XXX todo: check for other states to determine when to
		// reject a bad From or To
		if event.What == smtpd.GOTDATA {
			messageBuffer := bytes.NewBuffer([]byte(event.Arg))
			message, err := mail.ReadMessage(messageBuffer)
			if err != nil {
				return err
			}
			header := message.Header
			// XXX TODO: must lowercase and properly parse From and To
			sender := header.Get("From") // XXX
			receiver := header.Get("To") // XXX
			err = p.sendMessage(sender, receiver, []byte(event.Arg))
			if err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

package domain

import (
	"fmt"
	"strings"
)

// Bounds of a message.
//
// Explicit rather than absent: without them, a body of several megabytes would
// cross the domain all the way to the provider, where it would fail with a
// third-party library message instead of a business message.
const (
	subjectMaxLen = 300
	bodyMaxLen    = 1 << 20
)

// Channel is the means of conveyance.
//
// A named type rather than a string: that is what prevents passing a subject
// where a channel is expected, a confusion the compiler would never see between
// two `string`s.
type Channel string

// Channels SERVED by this module.
//
// The list says what EXISTS, not what is planned (ADR 014). `sms` and `push` are
// described in documentation/technique/modules-noyau.md and are not shipped:
// declaring them here would make them selectable, and a message would leave
// towards a channel nothing implements.
const (
	// ChannelEmail is the only shipped channel.
	ChannelEmail Channel = "email"
)

// Message is what one asks to convey.
//
// The CONTENT is supplied already rendered, not templated: this module knows no
// template engine, so it imposes none. Rendering belongs to the caller, who
// knows in which language and with which data to write.
type Message struct {
	Channel Channel
	To      Recipient
	Subject string
	Body    string
}

// NewMessage validates and normalises a message. The only useful construction
// path.
//
// # Why the channel is validated HERE and not at the driver
//
// A driver refusing the channel would do so after the message has crossed the
// use case, hence after a decorator could have logged it or counted it as sent.
// The most useful refusal is the earliest one.
func NewMessage(channel Channel, to Recipient, subject, body string) (Message, error) {
	if channel != ChannelEmail {
		return Message{}, fmt.Errorf("%w: %q — only %q is shipped", ErrUnknownChannel, channel, ChannelEmail)
	}
	if to.IsZero() {
		return Message{}, fmt.Errorf("%w: the recipient is required", ErrIncomplete)
	}

	trimmedSubject := strings.TrimSpace(subject)
	switch {
	case trimmedSubject == "":
		return Message{}, fmt.Errorf("%w: the subject is required", ErrIncomplete)
	case len(trimmedSubject) > subjectMaxLen:
		return Message{}, fmt.Errorf("%w: the subject exceeds %d characters", ErrIncomplete, subjectMaxLen)
	case strings.TrimSpace(body) == "":
		return Message{}, fmt.Errorf("%w: the body is required", ErrIncomplete)
	case len(body) > bodyMaxLen:
		return Message{}, fmt.Errorf("%w: the body exceeds %d bytes", ErrIncomplete, bodyMaxLen)
	}

	return Message{Channel: channel, To: to, Subject: trimmedSubject, Body: body}, nil
}

// String returns a form WITHOUT the body nor the address in clear.
//
// # What the masking prevents
//
// A notification body regularly carries a secret: confirmation link, reset
// token, one-time code. These are bearer credentials — whoever reads them can
// use them. Logging them turns a log leak into an account takeover.
//
// `Stringer` AND `GoStringer` are implemented, not out of redundancy: `%v` goes
// through the first, `%#v` through the second, and covering one leaves the other
// leaking. It took a test to discover this on `auth.Credential`.
func (m Message) String() string {
	return fmt.Sprintf(
		"Message{channel: %s, to: %s, subject: %q, body: *** (%d bytes)}",
		m.Channel, m.To.Masked(), m.Subject, len(m.Body))
}

// GoString masks the body under `%#v` too.
func (m Message) GoString() string { return m.String() }

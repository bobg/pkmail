package pkmail

import (
	"time"

	"github.com/bobg/rmime"
)

// SchemaVersion is the latest schema version as a semver string.
const SchemaVersion = "3.0.0"

// TODO: compatibility with http://schema.org/EmailMessage
// TODO: inverted index for text/* parts

type rPart struct {
	// PkmailVersion holds the value of pkmail.SchemaVersion at the time this blob was written.
	// For the original (prototype) schema version, this string is empty.
	PkmailVersion string `pk:"pkmail_version,inline"`

	CamliVersion             int               `pk:"camliVersion,inline"`
	CamliType                string            `pk:"camliType,inline"`
	ContentType              string            `pk:"content_type,inline"`
	ContentDisposition       string            `pk:"content_disposition,inline"`
	Header                   *rmime.Header     `pk:"header,inline,omitempty"`
	ContentTypeParams        map[string]string `pk:"content_type_params,omitempty,inline"`
	ContentDispositionParams map[string]string `pk:"content_disposition_params,omitempty,inline"`
	Time                     *time.Time        `pk:"time,omitempty,inline"`
	Charset                  string            `pk:"charset,omitempty,inline"`
	Subject                  string            `pk:"subject,omitempty,inline"`
	Sender                   *rAddress         `pk:"sender,omitempty"`
	Recipients               []*rAddress       `pk:"recipients,omitempty"`
	MessageID                string            `pk:"message_id,omitempty"`
	InReplyTo                []string          `pk:"in_reply_to,omitempty"`
	References               []string          `pk:"references,omitempty"`

	// Exactly one of the following is set.
	Multipart         *rMultipart           `pk:"multipart,omitempty"`
	SubMessage        *rPart                `pk:"submessage,omitempty"`
	DeliveryStatusBug *rmime.DeliveryStatus `pk:"delivery-status,omitempty"`
	DeliveryStatus    *rmime.DeliveryStatus `pk:"delivery_status,omitempty"`
	Body              string                `pk:"body,omitempty"`
}

type rAddress struct {
	Name    string `pk:"name,omitempty"`
	Address string `pk:"address,omitempty"`
}

type rMultipart struct {
	Preamble  string   `pk:"preamble,inline,omitempty"`
	Postamble string   `pk:"postamble,inline,omitempty"`
	Parts     []*rPart `pk:"parts,inline"`
}

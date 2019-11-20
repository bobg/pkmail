package pkmail

import (
	"time"
)

// SchemaVersion is the latest schema version as a semver string.
const SchemaVersion = "2.0.0"

// TODO: inverted index for text/* parts
type rPart struct {
	// PkmailVersion holds the value of pkmail.SchemaVersion at the time this blob was written.
	// For the original (prototype) schema version, this string is empty.
	PkmailVersion string `pk:"pkmail_version,inline"`

	CamliType                string            `pk:"camliType,inline"`
	ContentType              string            `pk:"content_type,inline"`
	ContentDisposition       string            `pk:"content_disposition,inline"`
	Header                   []*rField         `pk:"header,omitempty"`
	ContentTypeParams        map[string]string `pk:"content_type_params,omitempty,inline"`
	ContentDispositionParams map[string]string `pk:"content_disposition_params,omitempty,inline"`
	Time                     *time.Time        `pk:"time,omitempty,inline"`
	Charset                  string            `pk:"charset,omitempty,inline"`
	Subject                  string            `pk:"subject,omitempty,inline"`
	Sender                   *rAddress         `pk:"sender,omitempty"`
	Recipients               []*rAddress       `pk:"recipients,omitempty"`

	// Exactly one of the following is set.
	Multipart         *rMultipart      `pk:"multipart,omitempty"`
	SubMessage        *rPart           `pk:"submessage,omitempty"`
	DeliveryStatusBug *rDeliveryStatus `pk:"delivery-status,omitempty"`
	DeliveryStatus    *rDeliveryStatus `pk:"delivery_status,omitempty"`
	Body              string           `pk:"body,omitempty"`
}

type rField struct {
	N string   `pk:"name,inline"`
	V []string `pk:"value,inline"`
}

type rAddress struct {
	Name    string `pk:"name,omitempty"`
	Address string `pk:"address,omitempty"`
}

type rDeliveryStatus struct {
	Message    []*rField   `pk:"message,omitempty"`
	Recipients [][]*rField `pk:"recipients,omitempty"`
}

type rMultipart struct {
	Preamble  string   `pk:"preamble,omitempty"`
	Postamble string   `pk:"postamble,omitempty"`
	Parts     []*rPart `pk:"parts"`
}

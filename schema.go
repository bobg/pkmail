package pkmail

import (
	"encoding/json"
	"time"

	"github.com/bobg/rmime"
	"perkeep.org/pkg/blob"
)

// SchemaVersion is the latest schema version as a semver string.
const SchemaVersion = "1.0.1"

type schemaBase struct {
	// PkmailVersion holds the value of pkmail.SchemaVersion at the time this blob was written.
	// For the original (prototype) schema version, this string is empty.
	PkmailVersion string `json:"pkmail_version"`

	CamliType                string            `json:"camliType"`
	ContentType              string            `json:"content_type"`
	ContentDisposition       string            `json:"content_disposition"`
	Header                   []*rmime.Field    `json:"header,omitempty"`
	ContentTypeParams        map[string]string `json:"content_type_params,omitempty"`
	ContentDispositionParams map[string]string `json:"content_disposition_params,omitempty"`
	Time                     *time.Time        `json:"time,omitempty"`
	Charset                  string            `json:"charset,omitempty"`
	Subject                  string            `json:"subject,omitempty"`
	Sender                   *rmime.Address    `json:"sender,omitempty"`
	Recipients               []*rmime.Address  `json:"recipients,omitempty"`

	// Exactly one of the following is set.
	Subparts          []blob.Ref            `json:"subparts,omitempty"`
	SubMessage        *blob.Ref             `json:"submessage,omitempty"`
	DeliveryStatusBug *rmime.DeliveryStatus `json:"delivery-status,omitempty"`
	DeliveryStatus    *rmime.DeliveryStatus `json:"delivery_status,omitempty"`
	Body              *blob.Ref             `json:"body,omitempty"`
}

type schema schemaBase

// MarshalJSON ensures that the fields of s are marshaled in lexical order.
func (s *schema) MarshalJSON() ([]byte, error) {
	// Marshal once as the base type, getting fields in struct-definition order.
	j, err := json.Marshal((*schemaBase)(s))
	if err != nil {
		return nil, err
	}

	// Unmarshal as a dictionary and remarshal to get automatic field sorting.
	// Not the most efficient approach,
	// but it's simple and does allow us to define the type above more clearly
	// (i.e., segregating the body fields from the rest of the fields).
	var m map[string]interface{}
	err = json.Unmarshal(j, &m)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

package pkmail

import (
	"encoding/json"
	"time"

	"github.com/bobg/rmime"
	"perkeep.org/pkg/blob"
)

type partMapBase struct {
	CamliType                string            `json:"camliType"`
	ContentType              string            `json:"content_type"`
	ContentDisposition       string            `json:"content_disposition"`
	Header                   []*rmime.Field    `json:"header,omitempty"`
	ContentTypeParams        map[string]string `json:"content_type_params,omitempty"`
	ContentDispositionParams map[string]string `json:"content_disposition_params,omitempty"`
	Time                     time.Time         `json:"time,omitempty"`
	Charset                  string            `json:"charset,omitempty"`
	Subject                  string            `json:"subject,omitempty"`
	Sender                   *rmime.Address    `json:"sender,omitempty"`
	Recipients               []*rmime.Address  `json:"recipients,omitempty"`

	// Exactly one of the following is set.
	Subparts       []blob.Ref            `json:"subparts,omitempty"`
	SubMessage     *blob.Ref             `json:"submessage,omitempty"`
	DeliveryStatus *rmime.DeliveryStatus `json:"delivery_status,omitempty"`
	Body           *blob.Ref             `json:"body,omitempty"`
}

type partMap partMapBase

// MarshalJSON ensures that the fields of p are marshaled in lexical order.
func (p *partMap) MarshalJSON() ([]byte, error) {
	// Marshal once as the base type, getting fields in struct-definition order.
	j, err := json.Marshal((*partMapBase)(p))
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

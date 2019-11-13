// Package pkmail implements a schema for storing parsed e-mail messages in Perkeep.
package pkmail

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/bobg/rmime"
	"perkeep.org/pkg/blob"
	"perkeep.org/pkg/blobserver"
)

// PkPutMsg adds a message to the Perkeep server at dst. The message
// is added as a hierarchy of blobs with the root blob a schema blob
// having camliType "mime-message". See PkPutPart for other details
// of the root schema blob.
func PkPutMsg(ctx context.Context, dst blobserver.StatReceiver, msg *rmime.Message) (blob.Ref, error) {
	return pkPut(ctx, dst, (*rmime.Part)(msg), "mime-message")
}

// PkPutPart adds a message part to the Perkeep server at dst.
// The message part is added as a hierarchy of blobs with the root blob a
// schema blob having camliType "mime-part".
//
// Other fields of the root schema blob:
//   pkmail_version: semver string for the pkmail schema version in use
//   content_type: canonicalized content-type of the part
//   content_disposition: canonicalized content-disposition token ("inline" or "attachment")
//
// and optionally:
//   header: header of the part as a list of (name, list-of-values) pairs (if non-empty)
//   time: parsed date of the part
//   charset: charset for text/* parts
//   subject: decoded subject text for message parts
//   content_type_params
//   content_disposition_params
//
// Additionally, the body of the part appears as follows:
//   - for multipart/* parts, as the field "subparts",
//     a list of nested "mime-part" schema blobs
//   - for message/delivery-status parts, {"message": fields, "recipients": [fields, ...]}
//     as the field "delivery_status"
//   - for message/* parts, as the field "submessage",
//     a nested "mime-message" schema blob
//   - for other parts, as the field "body", a reference to a "bytes" schema blob.
func PkPutPart(ctx context.Context, dst blobserver.StatReceiver, p *rmime.Part) (blob.Ref, error) {
	return pkPut(ctx, dst, p, "mime-part")
}

// TODO:
//   - text/* parts get inverted index
//   - text/html parts get parsed into DOMs (?)
func pkPut(ctx context.Context, dst blobserver.StatReceiver, p *rmime.Part, camType string) (blob.Ref, error) {
	cd, cdParams := p.Disposition()
	s := &schema{
		PkmailVersion:            SchemaVersion,
		CamliType:                camType,
		ContentType:              p.Type(),
		ContentDisposition:       cd,
		Header:                   p.Fields,
		ContentTypeParams:        p.Params(),
		ContentDispositionParams: cdParams,
		Time:                     p.Time(),
		Subject:                  p.Subject(),
		Sender:                   p.Sender(),
		Recipients:               p.Recipients(),
	}
	if p.MajorType() == "text" {
		s.Charset = p.Charset()
	}

	switch p.MajorType() {
	case "multipart":
		multi := p.B.(*rmime.Multipart)
		var subpartRefs []blob.Ref
		for _, subpart := range multi.Parts {
			subpartRef, err := PkPutPart(ctx, dst, subpart)
			if err != nil {
				return blob.Ref{}, err
			}
			subpartRefs = append(subpartRefs, subpartRef)
		}
		s.Subparts = subpartRefs
		// TODO: preamble and postamble?

	case "message":
		switch p.MinorType() {
		case "rfc822", "news":
			submsg := p.B.(*rmime.Message)
			bodyRef, err := PkPutMsg(ctx, dst, submsg)
			if err != nil {
				return blob.Ref{}, err
			}
			s.SubMessage = &bodyRef

		case "delivery-status":
			s.DeliveryStatus = p.B.(*rmime.DeliveryStatus)

		default:
			return blob.Ref{}, rmime.ErrUnimplemented
		}

	default:
		bodyR, err := p.Body()
		if err != nil {
			return blob.Ref{}, err
		}
		bodyBytes, err := ioutil.ReadAll(bodyR)
		if err != nil {
			return blob.Ref{}, err
		}
		bodyRef := blob.RefFromBytes(bodyBytes)
		_, err = blobserver.ReceiveNoHash(ctx, dst, bodyRef, bytes.NewReader(bodyBytes))
		if err != nil {
			return blob.Ref{}, err
		}
		s.Body = &bodyRef
	}

	jBytes, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return blob.Ref{}, err
	}

	// Canonical form, according to mapJSON() in
	// perkeep.org/pkg/schema/schema.go.
	jStr := "{\"camliVersion\": 1,\n" + string(jBytes[2:])
	partRef := blob.RefFromString(jStr)

	_, err = blobserver.ReceiveNoHash(ctx, dst, partRef, strings.NewReader(jStr))
	return partRef, err
}

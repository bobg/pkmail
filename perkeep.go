package pkmail

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/bobg/rmime"
	"perkeep.org/pkg/blob"
	"perkeep.org/pkg/blobserver"
	"perkeep.org/pkg/schema"
)

// PkPutMsg adds a message to the Perkeep server at dst. The message
// is added as a hierarchy of blobs with the root blob a schema blob
// having camliType "mime-message". See PkPutPart for other details
// of the root schema blob.
func PkPutMsg(ctx context.Context, dst blobserver.StatReceiver, msg *rmime.Message) (blob.Ref, error) {
	return pkPut(ctx, dst, (*rmime.Part)(msg), "mime-message")
}

// PkPutPart adds a message part to the Perkeep server at dst. The
// message part is added as a hierarchy of blobs with the root blob a
// schema blob having camliType "mime-part".
//
// Other fields of the root schema blob:
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
	var (
		bodyName string
		body     interface{}
	)

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
		bodyName = "subparts"
		body = subpartRefs
		// TODO: preamble and postamble?

	case "message":
		switch p.MinorType() {
		case "rfc822":
			submsg := p.B.(*rmime.Message)
			bodyRef, err := PkPutMsg(ctx, dst, submsg)
			if err != nil {
				return blob.Ref{}, err
			}
			bodyName = "submessage"
			body = bodyRef

		case "delivery-status":
			bodyName = "delivery-status"
			body = p.B.(*rmime.DeliveryStatus)

		default:
			return blob.Ref{}, rmime.ErrUnimplemented
		}

	default:
		bodyR, err := p.Body()
		if err != nil {
			return blob.Ref{}, err
		}
		builder := schema.NewBuilder()
		builder.SetType("bytes")
		bodyRef, err := schema.WriteFileMap(ctx, dst, builder, bodyR)
		if err != nil {
			return blob.Ref{}, err
		}
		bodyName = "body"
		body = bodyRef
	}

	cd, cdParams := p.Disposition()
	m := map[string]interface{}{
		"camliType":           camType,
		"content_type":        p.Type(),
		"content_disposition": cd,
		bodyName:              body,
	}
	if len(p.Fields) > 0 {
		m["header"] = p.Fields
	}
	if params := p.Params(); len(params) > 0 {
		m["content_type_params"] = params
	}
	if len(cdParams) > 0 {
		m["content_disposition_params"] = cdParams
	}
	if t := p.Time(); t != (time.Time{}) {
		m["time"] = t
	}
	if p.MajorType() == "text" {
		m["charset"] = p.Charset()
	}
	if subj := p.Subject(); subj != "" {
		m["subject"] = subj
	}
	if sender := p.Sender(); sender != nil {
		m["sender"] = sender
	}
	if recipients := p.Recipients(); len(recipients) > 0 {
		m["recipients"] = recipients
	}

	jBytes, err := json.MarshalIndent(m, "", " ")
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

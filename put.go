// Package pkmail implements a schema for storing parsed e-mail messages in Perkeep.
package pkmail

import (
	"context"
	"io/ioutil"

	"github.com/bobg/pk"
	"github.com/bobg/rmime"
	"perkeep.org/pkg/blob"
	"perkeep.org/pkg/blobserver"
)

// PkPutMsg adds a message to the Perkeep server at dst. The message
// is added as a hierarchy of blobs with the root blob a schema blob
// having camliType "mime-message". See PkPutPart for other details
// of the root schema blob.
func PkPutMsg(ctx context.Context, dst blobserver.BlobReceiver, msg *rmime.Message) (blob.Ref, error) {
	rp, err := toRPart((*rmime.Part)(msg), "mime-message")
	if err != nil {
		return blob.Ref{}, err
	}
	return pk.Marshal(ctx, dst, rp)
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
func PkPutPart(ctx context.Context, dst blobserver.BlobReceiver, p *rmime.Part) (blob.Ref, error) {
	rp, err := toRPart(p, "mime-part")
	if err != nil {
		return blob.Ref{}, err
	}
	return pk.Marshal(ctx, dst, rp)
}

func toRPart(p *rmime.Part, camliType string) (*rPart, error) {
	cd, cdParams := p.Disposition()
	rp := &rPart{
		PkmailVersion:            SchemaVersion,
		CamliVersion:             1,
		CamliType:                camliType,
		ContentType:              p.Type(),
		ContentDisposition:       cd,
		ContentTypeParams:        p.Params(),
		ContentDispositionParams: cdParams,
		Subject:                  p.Subject(),
		MessageID:                p.MessageID(),
		InReplyTo:                p.InReplyTo(),
		References:               p.References(),
	}
	if t := p.Time(); !t.IsZero() {
		rp.Time = &t
	}
	if p.MajorType() == "text" {
		rp.Charset = p.Charset()
	}

	for _, f := range p.Header.Fields {
		rp.Header = append(rp.Header, &rField{
			N: f.N,
			V: f.V,
		})
	}
	if sender := p.Sender(); sender != nil {
		rp.Sender = &rAddress{
			Name:    sender.Name,
			Address: sender.Address,
		}
	}
	for _, recip := range p.Recipients() {
		rp.Recipients = append(rp.Recipients, &rAddress{
			Name:    recip.Name,
			Address: recip.Address,
		})
	}

	switch p.MajorType() {
	case "multipart":
		multi := p.B.(*rmime.Multipart)
		rmulti := &rMultipart{
			Preamble:  multi.Preamble,
			Postamble: multi.Postamble,
		}
		for _, subpart := range multi.Parts {
			subrp, err := toRPart(subpart, "mime-part") // xxx "mime-message" for message-type parts?
			if err != nil {
				return nil, err
			}
			rmulti.Parts = append(rmulti.Parts, subrp)
		}
		rp.Multipart = rmulti

	case "message":
		switch p.MinorType() {
		case "rfc822", "news":
			var err error

			submsg := p.B.(*rmime.Message)
			rp.SubMessage, err = toRPart((*rmime.Part)(submsg), "mime-message")
			if err != nil {
				return nil, err
			}

		case "delivery-status":
			ds := p.B.(*rmime.DeliveryStatus)
			rds := new(rDeliveryStatus)
			for _, msg := range ds.Message.Fields {
				rds.Message = append(rds.Message, &rField{
					N: msg.N,
					V: msg.V,
				})
			}
			for _, recip := range ds.Recipients {
				var rfields []*rField
				for _, f := range recip.Fields {
					rfields = append(rfields, &rField{
						N: f.N,
						V: f.V,
					})
				}
				rds.Recipients = append(rds.Recipients, rfields)
			}
			rp.DeliveryStatus = rds

		default:
			return nil, rmime.ErrUnimplemented
		}

	default:
		bodyR, err := p.Body()
		if err != nil {
			return nil, err
		}
		bodyBytes, err := ioutil.ReadAll(bodyR)
		if err != nil {
			return nil, err
		}
		rp.Body = string(bodyBytes)
	}

	return rp, nil
}

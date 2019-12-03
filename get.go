package pkmail

import (
	"context"

	"github.com/bobg/pk"
	"github.com/bobg/rmime"
	"github.com/pkg/errors"
	"perkeep.org/pkg/blob"
)

// PkGetMsg fetches the blobs from src, rooted at ref, to reconstruct an rmime.Message.
func PkGetMsg(ctx context.Context, src blob.Fetcher, ref blob.Ref) (*rmime.Message, error) {
	var rp rPart
	err := pk.Unmarshal(ctx, src, ref, &rp)
	if err != nil {
		return nil, err
	}

	p, err := fromRPart(&rp, "text/plain", true)
	return (*rmime.Message)(p), err
}

// ErrMalformed is the error produced when a blob does not conform to the pkmail schema.
var ErrMalformed = errors.New("malformed part")

func fromRPart(rp *rPart, defaultContentType string, expectMsg bool) (*rmime.Part, error) {
	if expectMsg {
		if rp.CamliType != "mime-message" {
			return nil, ErrMalformed
		}
	} else {
		switch rp.CamliType {
		case "mime-message", "mime-part": // ok
		default:
			return nil, ErrMalformed
		}
	}

	h := rp.Header
	p := &rmime.Part{
		Header: h,
	}

	switch h.MajorType() {
	case "multipart":
		if h.MinorType() == "digest" {
			defaultContentType = "message/rfc822"
		} else {
			defaultContentType = "text/plain"
		}
		rmulti := rp.Multipart
		multi := &rmime.Multipart{
			Preamble:  rmulti.Preamble,
			Postamble: rmulti.Postamble,
		}
		for _, rsubpart := range rmulti.Parts {
			subpart, err := fromRPart(rsubpart, defaultContentType, false)
			if err != nil {
				return nil, err
			}
			multi.Parts = append(multi.Parts, subpart)
		}
		p.B = multi

	case "message":
		switch h.MinorType() {
		case "rfc822", "news":
			msg, err := fromRPart(rp.SubMessage, "text/plain", false)
			if err != nil {
				return nil, err
			}
			p.B = (*rmime.Message)(msg)

		case "delivery-status":
			p.B = rp.DeliveryStatus

		default:
			return nil, rmime.ErrUnimplemented
		}

	default:
		p.B = rp.Body
	}

	return p, nil
}

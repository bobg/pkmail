package pkmail

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"

	"github.com/bobg/rmime"
	"github.com/pkg/errors"
	"perkeep.org/pkg/blob"
	pkschema "perkeep.org/pkg/schema"
)

// PkGetMsg fetches the blobs from src, rooted at ref, to reconstruct an rmime.Message.
func PkGetMsg(ctx context.Context, src blob.Fetcher, ref blob.Ref) (*rmime.Message, error) {
	part, err := pkGetPart(ctx, src, ref, true)
	if err != nil {
		return nil, errors.Wrap(err, "getting message-root part")
	}
	return (*rmime.Message)(part), nil
}

// ErrMalformed is the error produced when a blob does not conform to the pkmail schema.
var ErrMalformed = errors.New("malformed part")

func pkGetPart(ctx context.Context, src blob.Fetcher, ref blob.Ref, expectMsg bool) (*rmime.Part, error) {
	blobR, _, err := src.Fetch(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "fetching message-part blob")
	}
	defer blobR.Close()

	var s schema
	err = json.NewDecoder(blobR).Decode(&s)
	if err != nil {
		return nil, errors.Wrap(err, "parsing message-root blob")
	}

	if expectMsg && s.CamliType != "mime-message" {
		return nil, ErrMalformed
	}
	if !expectMsg && s.CamliType != "mime-part" {
		return nil, ErrMalformed
	}

	part := &rmime.Part{
		Header: &rmime.Header{
			Fields: s.Header,
		},
	}
	if expectMsg {
		part.Header.DefaultType = "message/rfc822"
	} else {
		part.Header.DefaultType = "text/plain"
	}

	ctParts := strings.Split(s.ContentType, "/")
	if len(ctParts) != 2 {
		return nil, ErrMalformed
	}
	major, minor := ctParts[0], ctParts[1]

	switch major {
	case "multipart":
		multi := new(rmime.Multipart)
		for _, subpartRef := range s.Subparts {
			part, err := pkGetPart(ctx, src, subpartRef, minor == "digest")
			if err != nil {
				return nil, errors.Wrap(err, "getting multipart subpart")
			}
			multi.Parts = append(multi.Parts, part)
		}
		part.B = multi

	case "message":
		switch minor {
		case "rfc822", "news":
			msg, err := pkGetPart(ctx, src, *s.SubMessage, false)
			if err != nil {
				return nil, errors.Wrap(err, "getting nested message")
			}
			part.B = (*rmime.Message)(msg)

		case "delivery-status":
			if s.DeliveryStatusBug != nil {
				part.B = s.DeliveryStatusBug
			} else {
				part.B = s.DeliveryStatus
			}

		default:
			return nil, rmime.ErrUnimplemented
		}

	default:
		var bodyR io.ReadCloser
		if s.PkmailVersion == "" {
			bodyR, err = pkschema.NewFileReader(ctx, src, *s.Body)
		} else {
			bodyR, _, err = src.Fetch(ctx, *s.Body)
		}
		if err != nil {
			return nil, errors.Wrap(err, "fetching body blob(s)")
		}
		defer bodyR.Close()
		bodyBytes, err := ioutil.ReadAll(bodyR)
		if err != nil {
			return nil, errors.Wrap(err, "reading body bytes")
		}
		part.B = bodyBytes
	}

	return part, nil
}

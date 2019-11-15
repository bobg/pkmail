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
	part, err := pkGetPart(ctx, src, ref, "text/plain", true)
	if err != nil {
		return nil, errors.Wrap(err, "getting message-root part")
	}
	return (*rmime.Message)(part), nil
}

// ErrMalformed is the error produced when a blob does not conform to the pkmail schema.
var ErrMalformed = errors.New("malformed part")

func pkGetPart(ctx context.Context, src blob.Fetcher, ref blob.Ref, defaultContentType string, expectMsg bool) (*rmime.Part, error) {
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

	if expectMsg {
		if s.CamliType != "mime-message" {
			return nil, ErrMalformed
		}
	} else {
		switch s.CamliType {
		case "mime-message", "mime-part": // ok
		default:
			return nil, ErrMalformed
		}
	}

	part := &rmime.Part{
		Header: &rmime.Header{
			Fields:      s.Header,
			DefaultType: defaultContentType,
		},
	}

	ctParts := strings.Split(s.ContentType, "/")
	if len(ctParts) != 2 {
		return nil, ErrMalformed
	}
	major, minor := ctParts[0], ctParts[1]

	switch major {
	case "multipart":
		if minor == "digest" {
			defaultContentType = "message/rfc822"
		} else {
			defaultContentType = "text/plain"
		}

		multi := new(rmime.Multipart)
		for _, subpartRef := range s.Subparts {
			part, err := pkGetPart(ctx, src, subpartRef, defaultContentType, false)
			if err != nil {
				return nil, errors.Wrap(err, "getting multipart subpart")
			}
			multi.Parts = append(multi.Parts, part)
		}
		part.B = multi

	case "message":
		switch minor {
		case "rfc822", "news":
			msg, err := pkGetPart(ctx, src, *s.SubMessage, "text/plain", false)
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
		part.B = string(bodyBytes)
	}

	return part, nil
}

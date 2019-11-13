package pkmail

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/bobg/rmime"
	"github.com/pkg/errors"
	"perkeep.org/pkg/blob"
	"perkeep.org/pkg/schema"
)

func PkGetMsg(ctx context.Context, src blob.Fetcher, ref blob.Ref) (*rmime.Message, error) {
	part, err := pkGetPart(ctx, src, ref, "text/plain")
	if err != nil {
		return nil, errors.Wrap(err, "getting message-root part")
	}
	return (*rmime.Message)(part), nil
}

var ErrMalformed = errors.New("malformed part")

func pkGetPart(ctx context.Context, src blob.Fetcher, ref blob.Ref, defaultType string) (*rmime.Part, error) {
	blobR, _, err := src.Fetch(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "fetching message-part blob")
	}
	defer blobR.Close()

	var m partMap
	err = json.NewDecoder(blobR).Decode(&m)
	if err != nil {
		return nil, errors.Wrap(err, "parsing message-root blob")
	}

	part := &rmime.Part{
		Header: &rmime.Header{
			Fields:      m.Header,
			DefaultType: defaultType,
		},
	}

	ctParts := strings.Split(m.ContentType, "/")
	if len(ctParts) != 2 {
		return nil, ErrMalformed
	}
	major, minor := ctParts[0], ctParts[1]

	switch major {
	case "multipart":
		if minor == "digest" {
			defaultType = "message/rfc822"
		} else {
			defaultType = "text/plain"
		}

		multi := new(rmime.Multipart)
		for _, subpartRef := range m.Subparts {
			part, err := pkGetPart(ctx, src, subpartRef, defaultType)
			if err != nil {
				return nil, errors.Wrap(err, "getting multipart subpart")
			}
			multi.Parts = append(multi.Parts, part)
		}
		part.B = multi

	case "message":
		switch minor {
		case "rfc822", "news":
			msg, err := pkGetPart(ctx, src, *m.SubMessage, "text/plain")
			if err != nil {
				return nil, errors.Wrap(err, "getting nested message")
			}
			part.B = (*rmime.Message)(msg)

		case "delivery-status":
			part.B = m.DeliveryStatus

		default:
			return nil, rmime.ErrUnimplemented
		}

	default:
		bodyBlobR, err := schema.NewFileReader(ctx, src, *m.Body)
		if err != nil {
			return nil, errors.Wrap(err, "fetching body blob")
		}
		defer bodyBlobR.Close()
		bodyBytes, err := ioutil.ReadAll(bodyBlobR)
		if err != nil {
			return nil, errors.Wrap(err, "reading body bytes")
		}
		part.B = bodyBytes
	}

	return part, nil
}

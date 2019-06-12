// Command pkmail reads folder names from the command line,
// parses their messages,
// and adds them to a Perkeep server.
//
// Each message added is made a "camliMember" of the "pkmail-messages" permanode.
// It is also made a "camliMember" of a permanode named for the folder it came from.
// Each folder permanode in turn is made a "camliMember" of "pkmail-folders".
package main

import (
	"context"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/bobg/folder/v3"
	"github.com/bobg/rmime"
	"perkeep.org/pkg/blob"
	clientpkg "perkeep.org/pkg/client"
	"perkeep.org/pkg/schema"

	"github.com/bobg/pkmail"
)

func main() {
	clientpkg.AddFlags() // add -server flag
	flag.Parse()

	client, err := clientpkg.New()
	if err != nil {
		log.Fatalf("creating perkeep client: %s", err)
	}

	ctx := context.Background()

	foldersPermanode, err := permanodeRef(ctx, client, "pkmail-folders")
	if err != nil {
		log.Fatalf("getting/creating pkmail-folders permanode: %s", err)
	}
	messagesPermanode, err := permanodeRef(ctx, client, "pkmail-messages")
	if err != nil {
		log.Fatalf("getting/creating pkmail-messages permanode: %s", err)
	}

	for _, arg := range flag.Args() {
		f, err := folder.Open(arg)
		if err != nil {
			log.Printf("processing %s: %s", arg, err)
			continue
		}
		folderPermanode, err := permanodeRef(ctx, client, arg)
		if err != nil {
			log.Printf("getting/creating permanode for folder %s: %s", arg, err)
			continue
		}
		err = addMember(ctx, client, foldersPermanode, folderPermanode)
		if err != nil {
			log.Printf("adding permanode for folder %s to pkmail-folders: %s", arg, err)
			continue
		}
		for i := 1; ; i++ {
			msgR, err := f.Message()
			if err != nil {
				log.Fatalf("opening message %d in %s: %s", i, arg, err)
			}
			if msgR == nil {
				break
			}
			msg, err := rmime.ReadMessage(msgR)
			if err != nil {
				log.Fatalf("reading message %d in %s: %s", i, arg, err)
			}
			err = msgR.Close()
			if err != nil {
				log.Fatalf("closing message %d in %s: %s", i, arg, err)
			}
			ref, err := pkmail.PkPutMsg(ctx, client, msg)
			if err != nil {
				log.Fatalf("adding message %d from %s: %s", i, arg, err)
			}
			log.Printf("message %d in %s added as %s", i, arg, ref)
			err = addMember(ctx, client, folderPermanode, ref)
			if err != nil {
				log.Fatalf("adding message %d from %s to folder permanode: %s", i, arg, err)
			}
			err = addMember(ctx, client, messagesPermanode, ref)
			if err != nil {
				log.Fatalf("adding message %d from %s to pkmail-messages: %s", i, arg, err)
			}
		}
	}
}

func permanodeRef(ctx context.Context, client *clientpkg.Client, key string) (blob.Ref, error) {
	builder := schema.NewPlannedPermanode(key)
	return signAndUpload(ctx, client, builder)
}

func addMember(ctx context.Context, client *clientpkg.Client, dst, src blob.Ref) error {
	builder := schema.NewAddAttributeClaim(dst, "camliMember", src.String())
	_, err := signAndUpload(ctx, client, builder)
	return err
}

func signAndUpload(ctx context.Context, client *clientpkg.Client, builder *schema.Builder) (blob.Ref, error) {
	signer, err := client.Signer()
	if err != nil {
		return blob.Ref{}, err
	}
	jStr, err := builder.SignAt(ctx, signer, time.Now())
	if err != nil {
		return blob.Ref{}, err
	}
	ref := blob.RefFromString(jStr)
	uploadHandle := &clientpkg.UploadHandle{
		BlobRef:  ref,
		Contents: strings.NewReader(jStr),
		Size:     uint32(len(jStr)),
	}
	_, err = client.Upload(ctx, uploadHandle)
	return ref, err
}

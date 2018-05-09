package main

import (
	"context"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/bobg/folder"
	"github.com/bobg/folder/maildir"
	"github.com/bobg/folder/mbox"
	"github.com/bobg/pkmail"
	"github.com/bobg/rmime"
	"github.com/bobg/uncompress"
	"perkeep.org/pkg/blob"
	clientpkg "perkeep.org/pkg/client"
	"perkeep.org/pkg/schema"
)

func main() {
	ctx := context.Background()

	server := flag.String("server", "localhost:3179", "perkeep server address")
	// xxx osutil.AddSecretRingFlag() // xxx it is messed up that this is needed

	flag.Parse()

	client, err := clientpkg.New(clientpkg.OptionServer(*server))
	if err != nil {
		log.Fatal("creating perkeep client: %s", err)
	}

	foldersPermanode, err := permanodeRef(ctx, client, "pkmail-folders")
	if err != nil {
		log.Fatalf("getting/creating pkmail-folders permanode: %s", err)
	}
	messagesPermanode, err := permanodeRef(ctx, client, "pkmail-messages")
	if err != nil {
		log.Fatalf("getting/creating pkmail-messages permanode: %s", err)
	}

	for _, arg := range flag.Args() {
		f, err := getFolder(arg)
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
			msgR, closer, err := f.Message()
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
			err = closer()
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

func getFolder(name string) (folder.Folder, error) {
	f, err := maildir.New(name)
	if err == nil {
		return f, nil
	}
	r, err := uncompress.OpenFile(name)
	if err != nil {
		return nil, err
	}
	return mbox.New(r)
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

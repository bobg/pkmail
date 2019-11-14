package main

import (
	"context"
	"io"
	"log"
	"strings"
	"time"

	"github.com/bobg/folder/v3"
	"github.com/bobg/rmime"
	"github.com/pkg/errors"
	"perkeep.org/pkg/blob"
	clientpkg "perkeep.org/pkg/client"
	"perkeep.org/pkg/schema"

	"github.com/bobg/pkmail"
)

func doImport(args []string) {
	ctx := context.Background()

	foldersPermanode, err := permanodeRef(ctx, client, "pkmail-folders")
	if err != nil {
		log.Fatalf("getting/creating pkmail-folders permanode: %s", err)
	}
	if *verbose {
		log.Printf("pkmail-folders permanode is %s", foldersPermanode)
	}

	messagesPermanode, err := permanodeRef(ctx, client, "pkmail-messages")
	if err != nil {
		log.Fatalf("getting/creating pkmail-messages permanode: %s", err)
	}
	if *verbose {
		log.Printf("pkmail-messages permanode is %s", messagesPermanode)
	}

	for _, arg := range args {
		doImportFolder(ctx, foldersPermanode, messagesPermanode, arg)
	}
}

func doImportFolder(ctx context.Context, foldersPermanode, messagesPermanode blob.Ref, arg string) {
	f, err := folder.Open(arg)
	if err != nil {
		log.Printf("processing %s: %s", arg, err)
		return
	}
	defer f.Close()

	folderPermanode, err := permanodeRef(ctx, client, arg)
	if err != nil {
		log.Fatalf("getting/creating permanode for folder %s: %s", arg, err)
	}

	err = addMember(ctx, client, foldersPermanode, folderPermanode)
	if err != nil {
		log.Fatalf("adding permanode for folder %s to pkmail-folders: %s", arg, err)
	}

	log.Printf("processing %s", arg)

	for i := 1; ; i++ {
		msgR, err := f.Message()
		if err != nil {
			log.Printf("opening message %d in %s: %s", i, arg, err)
			continue
		}
		if msgR == nil {
			break
		}
		err = addMessage(ctx, client, i, msgR, folderPermanode, messagesPermanode)
		if err != nil {
			log.Printf("adding message %d in %s: %s", i, arg, err)
			continue
		}
	}
}

func addMessage(ctx context.Context, client *clientpkg.Client, i int, r io.ReadCloser, folderPermanode, messagesPermanode blob.Ref) error {
	defer r.Close()
	msg, err := rmime.ReadMessage(r)
	if err != nil {
		return errors.Wrap(err, "reading message")
	}
	ref, err := pkmail.PkPutMsg(ctx, client, msg)
	if err != nil {
		return errors.Wrap(err, "storing message")
	}
	log.Printf("message %d added as %s", i, ref)
	err = addMember(ctx, client, folderPermanode, ref)
	if err != nil {
		return errors.Wrap(err, "adding message to folder permanode")
	}
	err = addMember(ctx, client, messagesPermanode, ref)
	return errors.Wrap(err, "adding message to pkmail-messages permanode")
}

func permanodeRef(ctx context.Context, client *clientpkg.Client, key string) (blob.Ref, error) {
	result, err := client.UploadPlannedPermanode(ctx, key, time.Unix(0, 0))
	if err != nil {
		return blob.Ref{}, err
	}
	return result.BlobRef, nil
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

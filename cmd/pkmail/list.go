package main

import (
	"context"
	"fmt"
	"log"

	"perkeep.org/pkg/search"
)

func doListFolders(_ []string) {
	ctx := context.Background()

	foldersPermanode, err := permanodeRef(ctx, client, "pkmail-folders")
	if err != nil {
		log.Fatalf("getting/creating pkmail-folders permanode: %s", err)
	}
	if *verbose {
		log.Printf("pkmail-folders permanode is %s", foldersPermanode)
	}

	req := &search.ClaimsRequest{
		Permanode:  foldersPermanode,
		AttrFilter: "camliMember",
	}
	resp, err := client.GetClaims(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range resp.Claims {
		fmt.Println(item.BlobRef.String())
	}
}

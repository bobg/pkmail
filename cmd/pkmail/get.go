package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew"
	"perkeep.org/pkg/blob"
	clientpkg "perkeep.org/pkg/client"

	"github.com/bobg/pkmail"
)

func doget(client *clientpkg.Client) {
	ctx := context.Background()
	for _, arg := range flag.Args() {
		ref, ok := blob.Parse(arg)
		if !ok {
			log.Fatalf("cannot parse %s as blob ref", arg)
		}
		msg, err := pkmail.PkGetMsg(ctx, client, ref)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s", spew.Sdump(msg))
	}
}

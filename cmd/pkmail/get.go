package main

import (
	"context"
	"log"
	"os"

	"perkeep.org/pkg/blob"

	"github.com/bobg/pkmail"
)

func doGet(args []string) {
	ctx := context.Background()
	for _, arg := range args {
		ref, ok := blob.Parse(arg)
		if !ok {
			log.Fatalf("cannot parse %s as blob ref", arg)
		}
		msg, err := pkmail.PkGetMsg(ctx, client, ref)
		if err != nil {
			log.Fatal(err)
		}
		_, err = msg.WriteTo(os.Stdout)
		if err != nil {
			log.Fatal(err)
		}
	}
}

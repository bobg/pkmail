// Command pkmail reads folder names from the command line,
// parses their messages,
// and adds them to a Perkeep server.
//
// Each message added is made a "camliMember" of the "pkmail-messages" permanode.
// It is also made a "camliMember" of a permanode named for the folder it came from.
// Each folder permanode in turn is made a "camliMember" of "pkmail-folders".
package main

import (
	"flag"
	"log"

	clientpkg "perkeep.org/pkg/client"
)

var commands = map[string]func([]string){
	"import": doImport,
	"get":    doGet,
}

var client *clientpkg.Client

func main() {
	clientpkg.AddFlags() // add -server flag
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("usage: pkmail [global args] COMMAND [command args]")
	}
	cmd := commands[args[0]]
	if cmd == nil {
		log.Fatalf("unknown command %s", args[0])
	}

	var err error
	client, err = clientpkg.New()
	if err != nil {
		log.Fatalf("creating perkeep client: %s", err)
	}

	cmd(args[1:])
}

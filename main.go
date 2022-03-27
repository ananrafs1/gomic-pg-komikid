package main

import (
	// "log"

	"github.com/ananrafs1/gomic-pg-komikid/grpc"
	// "github.com/ananrafs1/gomic-pg-komikid/komikid"
)

func main() {
	grpc.Serve()
	// scr := komikid.Extract("one-piece-id", 1, 10)
	// log.Println(scr.Chapters)
}

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"bootlegger/internal/api"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	web := flag.Bool("web", false, "serve HTML web UI (GoTTH) in addition to JSON API")
	flag.Parse()

	var srv *api.Server
	if *web {
		fmt.Fprintf(os.Stdout, "BOOTLEGGER SERVER [WEB+API] — %s\n", *addr)
		srv = api.NewWebServer(*addr)
	} else {
		fmt.Fprintf(os.Stdout, "BOOTLEGGER SERVER [API] — %s\n", *addr)
		srv = api.NewServer(*addr)
	}
	log.Fatal(srv.ListenAndServe())
}

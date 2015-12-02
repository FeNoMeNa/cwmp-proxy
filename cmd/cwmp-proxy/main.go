package main

import (
	"flag"
	"log"

	"github.com/FeNoMeNa/cwmp-proxy"
)

var (
	port    = flag.Int("port", 0, "CWMP proxy port")
	backend = flag.String("backend", "", "The backend ACS server")
)

func main() {
	flag.Parse()

	if *port == 0 || *backend == "" {
		flag.PrintDefaults()
		return
	}

	p, err := cwmpproxy.New(*port, *backend)

	if err != nil {
		log.Fatalf("The CWMP proxy cannot be created - %v", err)
		return
	}

	err = p.Start()

	if err != nil {
		log.Fatalf("The CWMP proxy cannot be started - %v", err)
		return
	}
}

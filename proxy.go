package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
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

	log.Printf("The CWMP proxy server will be started on port %v\n", *port)

	p := NewProxy(*port, *backend)

	log.Fatalln(p.Start())
}

type Proxy struct {
	port    int
	backend *url.URL
}

func NewProxy(port int, backend string) *Proxy {
	u, _ := url.Parse(backend)

	return &Proxy{port, u}
}

func (p *Proxy) Start() error {
	http.Handle("/", p.handler())
	http.Handle("/client", wakeupHandler())

	return http.ListenAndServe(p.address(), nil)
}

func (p *Proxy) handler() http.Handler {
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			cwmp := newCwmpMessage(req)
			cwmp.replaceConnectionUrl(req.Host)

			req.URL.Scheme = p.backend.Scheme
			req.URL.Host = p.backend.Host
		},
	}
}

func wakeupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.FormValue("origin")

		if origin == "" {
			log.Println("The origin connection URL should be provided!")
			http.Error(w, "The origin connection URL should be provided!", http.StatusBadRequest)
			return
		}

		log.Printf("Waking up CPE with URL: %s", origin)

		_, err := http.Get(origin)

		if err != nil {
			log.Printf("An error occurred with the CPE communication - %v", err)
			http.Error(w, "An error occurred with the CPE communication!", http.StatusBadRequest)
			return
		}
	}
}

func (p *Proxy) address() string {
	return fmt.Sprintf(":%d", p.port)
}

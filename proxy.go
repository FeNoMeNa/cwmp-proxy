package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/FeNoMeNa/goha"
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

	p, err := NewProxy(*port, *backend)

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

type Proxy struct {
	listener net.Listener
	backend  *url.URL
}

func NewProxy(port int, backend string) (*Proxy, error) {
	u, err := url.Parse(backend)

	if err != nil {
		return new(Proxy), err
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		return new(Proxy), err
	}

	return &Proxy{l, u}, nil
}

func (p *Proxy) Start() error {
	http.Handle("/", p.handler())
	http.Handle("/client", basicAuthHandler(wakeupHandler))

	return http.Serve(p.listener, nil)
}

func (p *Proxy) Close() error {
	return p.listener.Close()
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

func wakeupHandler(username, password string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.FormValue("origin")

		if origin == "" {
			http.Error(w, "The origin connection URL should be provided!", http.StatusBadRequest)
			return
		}

		_, err := goha.NewClient(username, password).Get(origin)

		if err != nil {
			http.Error(w, "An error occurred with the CPE communication!", http.StatusBadRequest)
			return
		}
	}
}

func basicAuthHandler(handler func(string, string) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok {
			w.Header().Add("WWW-Authenticate", `Basic realm="cwmp-proxy"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		handler(username, password).ServeHTTP(w, r)
	}
}

package main

import (
	"flag"
	"fmt"
	"github.com/FeNoMeNa/goha"
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
	http.Handle("/client", basicAuthHandler(wakeupHandler))

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

func (p *Proxy) address() string {
	return fmt.Sprintf(":%d", p.port)
}

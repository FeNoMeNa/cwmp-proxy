package main

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestProxyServer(t *testing.T) {
	cpeWokenup := false

	cpe := fakeHttpServer(func(w http.ResponseWriter, r *http.Request) {
		cpeWokenup = true
	})

	in := bytes.NewReader([]byte(
		`
			<ParameterValueStruct>
				<Name>InternetGatewayDevice.ManagementServer.ConnectionRequestURL</Name>
				<Value xsi:type="xsd:string">` + cpe.URL + `</Value>
			</ParameterValueStruct>
		`,
	))

	want := bytes.NewReader([]byte(
		`
			<ParameterValueStruct>
				<Name>InternetGatewayDevice.ManagementServer.ConnectionRequestURL</Name>
				<Value xsi:type="xsd:string">http://localhost:1717/client?origin=` + cpe.URL + `</Value>
			</ParameterValueStruct>
		`,
	))

	backend := fakeHttpServer(func(w http.ResponseWriter, r *http.Request) {
		compareReaders(t, want, r.Body)
		sendWakeupRequest(cpe.URL)
	})

	// Let's hope this port to be free
	startProxy(1717, backend.URL)

	sendInform("http://localhost:1717", in)

	if cpeWokenup == false {
		t.Errorf("The CPE is not woken up!")
	}
}

func startProxy(port int, backend string) {
	proxy := NewProxy(port, backend)
	go proxy.Start() // Run proxy server in background
}

func sendInform(url string, inform io.Reader) {
	http.Post(url, "text/xml", inform)
}

func sendWakeupRequest(origin string) {
	http.Get("http://localhost:1717/client?origin=" + origin)
}

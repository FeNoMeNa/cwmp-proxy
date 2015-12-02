package cwmpproxy

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestProxyServer(t *testing.T) {
	c := struct {
		username, password string
		cpeWokenup         bool
	}{
		"FeNoMeNa", "12345678", false,
	}

	cpe := fakeHttpServer(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok {
			w.Header().Add("WWW-Authenticate", `Basic realm="wakeup-test"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if c.username != username {
			t.Errorf("expected username: %v", c.username)
			t.Errorf("     got username: %v", username)

		}

		if c.password != password {
			t.Errorf("expected password: %v", c.password)
			t.Errorf("     got password: %v", password)
		}

		c.cpeWokenup = true
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

		url := "http://localhost:1717/client?origin=" + cpe.URL

		sendWakeupRequest(url, c.username, c.password)
	})

	// Let's hope this port to be free
	startProxy(1717, backend.URL)

	sendInform("http://localhost:1717", in)

	if c.cpeWokenup == false {
		t.Errorf("The CPE is not woken up!")
	}
}

func startProxy(port int, backend string) {
	proxy, _ := New(port, backend)
	go proxy.Start() // Run proxy server in background
}

func sendInform(url string, inform io.Reader) {
	http.Post(url, "text/xml", inform)
}

func sendWakeupRequest(url, username, password string) {
	request, _ := http.NewRequest("GET", url, nil)
	request.SetBasicAuth(username, password)

	http.DefaultClient.Do(request)
}

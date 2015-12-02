package cwmpproxy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// cwmpMessage represents a cwmp message. It wraps the current http request.
type cwmpMessage struct {
	req *http.Request
}

// newCwmpMessage creates and initializes a new cwmp message.
func newCwmpMessage(req *http.Request) *cwmpMessage {
	return &cwmpMessage{req}
}

// replaceConnectionUrl replaces the connection url assigned in the cwmp message.
// If the current cwmp message does not contain connection url, the message is not
// modified.
func (i *cwmpMessage) replaceConnectionUrl(host string) {
	content, err := ioutil.ReadAll(i.req.Body)

	if err != nil {
		return
	}

	message := string(content)

	url, ok := getConnectionUrl(message)

	if ok {
		newUrl := fmt.Sprintf("http://%s/client?origin=%s", host, url)
		message = strings.Replace(message, url, newUrl, 1)
	}

	length := len(message)
	buffer := bytes.NewReader([]byte(message))

	i.req.Body = ioutil.NopCloser(buffer)
	i.req.ContentLength = int64(length)
}

// getConnectionUrl fetches the connection url of the provided message if exists.
func getConnectionUrl(message string) (string, bool) {
	index := strings.Index(message, "ConnectionRequestURL")

	if index == -1 {
		return "", false
	}

	start := index + strings.Index(message[index:], "http")
	end := start + strings.Index(message[start:], "<")

	return message[start:end], true
}

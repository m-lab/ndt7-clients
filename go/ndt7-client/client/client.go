// Package client implements a minimal ndt7 client.
package client

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/apex/log"
	"github.com/gorilla/websocket"
	"github.com/m-lab/ndt7-clients/go/ndt7-client/protocol"
)

// Client is a ndt7 client.
type Client struct {
	// Hostname is the hostname to use
	Hostname string

	// Port is the port to use
	Port string

	// Insecure controls whether to skip TLS verification
	Insecure bool
}

// dial allows to inject failures when running tests
var dial = func(dialer websocket.Dialer, URL string, header http.Header) (*websocket.Conn, *http.Response, error) {
	return dialer.Dial(URL, header)
}

// dial creates and configures the websocket connection
func (cl Client) dial(urlpath string) (*websocket.Conn, error) {
	var URL url.URL
	URL.Scheme = "wss"
	URL.Path = urlpath
	URL.Host = cl.Hostname + ":" + cl.Port
	var dialer websocket.Dialer
	if cl.Insecure {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	log.Debugf("Connecting to: %s", URL.String())
	headers := http.Header{}
	headers.Add("Sec-WebSocket-Protocol", "net.measurementlab.ndt.v7")
	dialer.HandshakeTimeout = 3 * time.Second
	conn, _, err := dial(dialer, URL.String(), headers)
	if err != nil {
		return nil, err
	}
	// According to the specification we must be prepared to read messages
	// that are smaller than the following value.
	conn.SetReadLimit(1 << 17)
	return conn, nil
}

// closeandwarn will warn if closing a closer causes a failure
func closeandwarn(closer io.Closer, message string) {
	err := closer.Close()
	if err != nil {
		log.WithError(err).Warn(message)
	}
}

// Download runs a ndt7 download test.
func (cl Client) Download() error {
	conn, err := cl.dial("/ndt/v7/download")
	if err != nil {
		return err
	}
	defer closeandwarn(conn, "Ignored error when closing connection")
	return protocol.Counterflow(conn, protocol.Measurer(protocol.Reader(conn)))
}

// Upload runs a ndt7 upload test.
func (cl Client) Upload() error {
	conn, err := cl.dial("/ndt/v7/upload")
	if err != nil {
		return err
	}
	defer closeandwarn(conn, "Ignored error when closing connection")
	go func() {
		for range protocol.Reader(conn) {
			// discard
		}
	}()
	return <-protocol.Writer(conn)
}

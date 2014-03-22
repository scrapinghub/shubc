package scrapinghub

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"
)

// Represent the Http verbs like GET, POST, PUT, etc
type HttpVerb int32

const (
	_            = iota
	GET HttpVerb = 1 * iota
	POST
)

// Retruns the string representation of a HttpVerb
// e.g.: GET -> "GET"
func (hv HttpVerb) String() string {
	switch hv {
	case GET:
		return "GET"
	case POST:
		return "POST"
	}
	return ""
}

// The connection holds information about the http client,
// the user API key, the API url and the parsed form of the
// API url.
type Connection struct {
	client        *http.Client
	apikey        string
	user_agent    string
	BaseUrl       string
	ParsedBaseUrl url.URL
}

// Create a new connection to Scrapinghub API
func (conn *Connection) New(apikey string) (err error) {
	// Create TLS config
	tlsConfig := tls.Config{RootCAs: nil}
	ConnectionTimeout := time.Duration(60 * time.Second)

	tr := &http.Transport{
		TLSClientConfig:    &tlsConfig,
		DisableCompression: true,
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, ConnectionTimeout)
		},
		ResponseHeaderTimeout: ConnectionTimeout,
	}
	conn.apikey = apikey
	conn.BaseUrl = APIURL
    purl, err := url.Parse(conn.BaseUrl)
	if err != nil {
		return fmt.Errorf("Connection.New: cannot parse base url provided, error message: %s\n", err)
	}
    conn.ParsedBaseUrl = *purl
	conn.user_agent = USER_AGENT
	conn.client = &http.Client{Transport: tr}
	return nil
}

// Set a new API url (e.g: for testing purposes)
func (conn *Connection) SetAPIUrl(apiurl string) (err error) {
	conn.BaseUrl = apiurl
    purl, err := url.Parse(conn.BaseUrl)
	if err != nil {
		return fmt.Errorf("Connection.New: cannot parse base url provided, error message: %s\n", err)
	}
    conn.ParsedBaseUrl = *purl
	return nil
}

// Call the API using a GET or POST HTTP request, to the method `method` and  `params` of type url.Values.
// Returns a reponse type `http.Reponse` and `error` (nil if no error ocurred)
func (conn *Connection) APICall(method string, http_method HttpVerb, params *url.Values) (*http.Response, error) {
	var err error
	var buf io.Reader = nil

	query_url := conn.ParsedBaseUrl
	query_url.Path = path.Join(query_url.Path, method)

	if http_method == GET {
		if params != nil {
			query_url.RawQuery, err = url.QueryUnescape(params.Encode())
			if err != nil {
				return nil, err
			}
		}
	} else if http_method == POST {
		if params != nil {
			buf = bytes.NewBufferString(params.Encode())
		}
	} else {
		return nil, fmt.Errorf("Connection.APICall: '%s' http method not supported\n", http_method.String())
	}
	req, err := http.NewRequest(http_method.String(), query_url.String(), buf)
	if err != nil {
		return nil, err
	}
	// Set Scrapinghub api key to request
	req.SetBasicAuth(conn.apikey, "")
	req.Header.Add("User-Agent", conn.user_agent)
	return conn.client.Do(req)
}

// Equal to APICall(method, http_method, params) but reads the body of the response
// and returns a nice []byte type. Also returns an error in case its ocurr.
func (conn *Connection) APICallReadBody(method string, http_method HttpVerb, params *url.Values) ([]byte, error) {
	resp, err := conn.APICall(method, http_method, params)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(resp.Body)
}

// Call the API using a Form POST request, to the method `method` with params
// `params` of type url.Values and with `files` is a map with
// <filename, filepath> to be posted
// Returns a nice []byte type with the response.Body read into it. Also returns
// an error (nil if no error ocurred)
func (conn *Connection) APIPostFilesReadBody(method string, params *url.Values, files map[string]string) ([]byte, error) {
	body := &bytes.Buffer{}

	writer := multipart.NewWriter(body)
	for fname, file_path := range files {
		file, err := os.Open(file_path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		part, err := writer.CreateFormFile(fname, filepath.Base(file_path))
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(part, file)
	}
	for key, vals := range *params {
		for _, val := range vals {
			_ = writer.WriteField(key, val)
		}
	}
	err := writer.Close()
	if err != nil {
		return nil, err
	}

	query_url := conn.ParsedBaseUrl
	query_url.Path = path.Join(query_url.Path, method)

	req, err := http.NewRequest("POST", query_url.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())
	// Set Scrapinghub api key to request
	req.SetBasicAuth(conn.apikey, "")
	req.Header.Add("User-Agent", conn.user_agent)
	resp, err := conn.client.Do(req)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(resp.Body)
}

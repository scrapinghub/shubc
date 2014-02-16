package scrapinghub

import (
    "net/http"
    "crypto/tls"
    "encoding/json"
    "errors"
)


var baseUrl = "https://dash.scrapinghub.com/api"

type Connection struct {
    client *http.Client
    apikey string
}

func (conn *Connection) New (apikey string) {
    // Create TLS config
    tlsConfig := tls.Config{RootCAs: nil}

    // If insecure, skip CA verfication
    //if insecure {
    //    tlsConfig.InsecureSkipVerify = true
    //}

    tr := &http.Transport{
        TLSClientConfig:    &tlsConfig,
        DisableCompression: true,
    }

    conn.apikey = apikey
    conn.client = &http.Client{Transport: tr}
}

func (conn *Connection) do_request(url string) ([]byte, error) {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    // Set Scrapinghub api key to request
    req.SetBasicAuth(conn.apikey, "")
    resp, err := conn.client.Do(req)
    if err != nil {
        return nil, err
    }

    content := make([]byte, resp.ContentLength)
    nread, err := resp.Body.Read(content)
    if err != nil {
        return nil, err
    }
    if int64(nread) != resp.ContentLength {
        return nil, errors.New("Content read is different than the response content length")
    }
    return content, nil
}

type Spiders struct {
    Spiders []map[string]string
    Status string
}

// errors
var spider_list_error = errors.New("Spiders.List: Error while retrieving the spider list")

func (spider *Spiders) List (conn *Connection, project_id string) (*Spiders, error) {
    method := "/spiders/list.json?project=" + project_id
    content, err := conn.do_request(baseUrl + method)
    if err != nil {
        return nil, err
    }
    
    json.Unmarshal(content, spider)

    if spider.Status != "ok" {
        return nil, spider_list_error
    } else {
        return spider, nil
    }
}


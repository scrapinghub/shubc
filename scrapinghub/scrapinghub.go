package scrapinghub

import (
    "fmt"
    "io"
    "bytes"
    "strings"
    "net/url"
    "net/http"
    "crypto/tls"
    "encoding/json"
    "errors"
)

// Scrapinghub Base API URL
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

func (conn *Connection) do_request(rurl string, method string, params map[string]string) ([]byte, error) {
    var req *http.Request
    var err error

    if method == "GET" {
        req, err = http.NewRequest("GET", rurl, nil)
    } else if method == "POST" {
        data := url.Values{}
        for k, v := range(params) {
            data.Add(k, v)
        }
        req, err = http.NewRequest("POST", rurl, bytes.NewBufferString(data.Encode()))
    }

    if err != nil {
        return nil, err
    }
    // Set Scrapinghub api key to request
    req.SetBasicAuth(conn.apikey, "")
    resp, err := conn.client.Do(req)
    if err != nil {
        return nil, err
    }

    // Create buffer
    content := make([]byte, 0)
    buf := make([]byte, 1024)
    for {
        n, err := resp.Body.Read(buf)
        if err != nil && err != io.EOF { panic(err) }
        if n == 0 { break }
        content = append(content, buf[:n]...)
    }
    return content, nil

}

type Spiders struct {
    Spiders []map[string]string
    Status string
}

// errors
var spider_list_error = errors.New("Spiders.List: Error while retrieving the spider list")
var jobs_list_error = errors.New("Jobs.List: Error while retrieving the jobs list")

func (spider *Spiders) List (conn *Connection, project_id string) (*Spiders, error) {
    method := "/spiders/list.json?project=" + project_id
    content, err := conn.do_request(baseUrl + method, "GET", nil)
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

type Jobs struct {
    Status string
    Count int
    Total int
    Jobs []map[string]interface{}
    JobId string
    Message string
}

/* 
  Given a list of ["key=val", "key2=val2", ...] returns its correspdonding map 
*/
func equality_list_to_map(data []string) map[string]string {
    result := make(map[string]string)
    for _, e := range(data) {
        if strings.Index(e, "=") > 0 {
            res := strings.Split(e, "=")
            result[strings.TrimSpace(res[0])] = strings.TrimSpace(res[1])
        }
    }
    return result
}

func (jobs *Jobs) List(conn *Connection, project_id string, count int, filters []string) (*Jobs, error) {
    method := fmt.Sprintf("/jobs/list.json?project=%s&count=%d", project_id, count)
    mfilters := equality_list_to_map(filters)
    for fname, fval := range(mfilters) {
        method = fmt.Sprintf("%s&%s=%s", method, fname, fval)
    }
    content, err := conn.do_request(baseUrl + method, "GET", nil)
    if err != nil {
        return nil, err
    }

    json.Unmarshal(content, jobs)

    if jobs.Status != "ok" {
        return nil, jobs_list_error
    } else {
        return jobs, nil
    }
}

func (jobs *Jobs) JobInfo(conn *Connection, job_id string) (map[string]string, error) {
    res := strings.Split(job_id, "/")
    project_id := res[0]

    method := fmt.Sprintf("/jobs/list.json?project=%s&job=%s", project_id, job_id)
    content, err := conn.do_request(baseUrl + method, "GET", nil)
    if err != nil {
        return nil, err
    }
    json.Unmarshal(content, jobs)

    if jobs.Status != "ok" {
        return nil, errors.New("Jobs.JobInfo: Error while retrieving job info")
    } else {
        if len(jobs.Jobs) > 0 {
            m := make(map[string]string)
            for k, v := range(jobs.Jobs[0]) {
                m[k] = fmt.Sprintf("%v", v)
            }
            return m, nil
        } else {
            return nil, errors.New(fmt.Sprintf("Jobs.JobInfo: Job %s does not exist", job_id))
        }
    }
}

func (jobs *Jobs) Schedule(conn *Connection, project_id string, spider_name string, args []string) (string, error) {
    method := "/schedule.json"
    data := map[string]string {
        "project": project_id,
        "spider": spider_name,
    }
    params := equality_list_to_map(args)
    for k, v := range(params) {
        data[k] = v
    }
    content, err := conn.do_request(baseUrl + method, "POST", data)
    if err != nil {
        return "", err
    }
    json.Unmarshal(content, jobs)

    if jobs.Status != "ok" {
        return "", errors.New(fmt.Sprintf("Jobs.Schedule: Error while scheduling the job. Message: %s", jobs.Message))
    } else {
        return jobs.JobId, nil
    }
}

func (jobs *Jobs) Stop(conn *Connection, job_id string) error {
    res := strings.Split(job_id, "/")
    project_id := res[0]
    method := "/jobs/stop.json"

    data := map[string]string {
        "project": project_id,
        "job": job_id,
    }
    content, err := conn.do_request(baseUrl + method, "POST", data)
    if err != nil {
        return err
    }
    json.Unmarshal(content, jobs)

    if jobs.Status != "ok" {
        return errors.New(fmt.Sprintf("Jobs.Stop: Error while stopping the job. Message: %s", jobs.Message))
    } else {
        return nil
    }
}

func RetrieveItems(conn *Connection, job_id string, count, offset int) ([]map[string]interface{}, error) {
    res := strings.Split(job_id, "/")
    project_id := res[0]
    method := fmt.Sprintf("/items.json?project=%s&job=%s&count=%d&offset=%d", project_id, job_id, count, offset)

    content, err := conn.do_request(baseUrl + method, "GET", nil)
    if err != nil {
        return nil, err
    }
    var f interface{}
    err = json.Unmarshal(content, &f)
    if err != nil {
        return nil, err
    }
    jarray := f.([]interface{})

    items := make([]map[string]interface{}, len(jarray))
    for i, e := range(jarray) {
        items[i] = e.(map[string]interface{})
    }
    return items, nil
}

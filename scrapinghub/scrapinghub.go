// Go bindings for Scrapinghub API (http://doc.scrapinghub.com/api.html)
package scrapinghub

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

// Scrapinghub Base API URL
var re_jobid = regexp.MustCompile(`(?P<project_id>\d+)/\d+/\d+`)
var libversion = "0.1"

type Connection struct {
	client     *http.Client
	apikey     string
	user_agent string
	BaseUrl    string
}

// Create a new connection to Scrapinghub API
func (conn *Connection) New(apikey string) {
	// Create TLS config
	tlsConfig := tls.Config{RootCAs: nil}
	tr := &http.Transport{
		TLSClientConfig:    &tlsConfig,
		DisableCompression: true,
	}
	conn.apikey = apikey
	conn.BaseUrl = "https://dash.scrapinghub.com/api"
	conn.user_agent = fmt.Sprintf("scrapinghub.go/%s (http://github.com/scrapinghub/shubc)", libversion)
	conn.client = &http.Client{Transport: tr}
}

func (conn *Connection) SetAPIUrl(url string) {
	conn.BaseUrl = url
}

// Do a GET HTTP request and returns a reponse type `http.Reponse` and `error`
func (conn *Connection) Get(rurl string) (*http.Response, error) {
	req, err := http.NewRequest("GET", rurl, nil)
	if err != nil {
		return nil, err
	}
	// Set Scrapinghub api key to request
	req.SetBasicAuth(conn.apikey, "")
	req.Header.Add("User-Agent", conn.user_agent)
	return conn.client.Do(req)
}

// Do a POST HTTP request and returns a reponse type `http.Reponse` and `error`
// Argument `params` is a map with the POST parameters
func (conn *Connection) Post(rurl string, params map[string][]string) (*http.Response, error) {
	data := url.Values{}
	for k, vals := range params {
		for _, v := range vals {
			data.Add(k, v)
		}
	}
	req, err := http.NewRequest("POST", rurl, bytes.NewBufferString(data.Encode()))

	if err != nil {
		return nil, err
	}
	// Set Scrapinghub api key to request
	req.SetBasicAuth(conn.apikey, "")
	req.Header.Add("User-Agent", conn.user_agent)
	return conn.client.Do(req)
}

// Do a POST HTTP request and returns a reponse type `http.Reponse` and `error`
// Argument `params` is a map with the POST parameters, and files is a map with
// <filename, filepath> to be posted
func (conn *Connection) PostFiles(rurl string, params map[string][]string, files map[string]string) (*http.Response, error) {
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
	for key, vals := range params {
		for _, val := range vals {
			_ = writer.WriteField(key, val)
		}
	}
	err := writer.Close()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", rurl, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())
	// Set Scrapinghub api key to request
	req.SetBasicAuth(conn.apikey, "")
	req.Header.Add("User-Agent", conn.user_agent)
	return conn.client.Do(req)
}

// Build the full API URL using the connection.BaseUrl, the `method` to query and
// all the parameters `params`.
// Returns the full url processed and an error if exist
func build_api_url(conn *Connection, method string, params *url.Values) (string, error) {
	query_url, err := url.Parse(conn.BaseUrl)
	if err != nil {
		return "", err
	}
	query_url.Path = path.Join(query_url.Path, method)
	query_url.RawQuery, err = url.QueryUnescape(params.Encode())
	if err != nil {
		return "", err
	}
	return query_url.String(), nil
}

type Spiders struct {
	Spiders []map[string]string
	Status  string
}

// errors
var (
	spider_list_error  = errors.New("Spiders.List: Error while retrieving the spider list")
	jobs_list_error    = errors.New("Jobs.List: Error while retrieving the jobs list")
	wrong_job_id_error = errors.New("Job ID is empty or not in the right format (e.g: 123/1/2)")
)

func (spider *Spiders) List(conn *Connection, project_id string) (*Spiders, error) {
	params := url.Values{}
	params.Add("project", project_id)
	query_url, err := build_api_url(conn, "/spiders/list.json", &params)
	if err != nil {
		return nil, err
	}
	resp, err := conn.Get(query_url)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(content, spider)

	if spider.Status != "ok" {
		return nil, spider_list_error
	}
	return spider, nil
}

type Job struct {
	CloseReason       string `json:"close_reason"`
	Elapsed           int
	ErrorsCount       int `json:"errors_count"`
	Id                string
	ItemsScraped      int    `json:"items_scraped"`
	SpiderType        string `json:"spider_type"`
	ResponsesReceived int    `json:"responses_received"`
	Logs              int
	Priority          int
	Spider            string
	SpiderArgs        map[string]string `json:"spider_args"`
	StartedTime       string            `json:"started_time"`
	State             string
	Tags              []string
	UpdatedTime       string `json:"updated_time"`
	Version           string
}

type Jobs struct {
	Status  string
	Count   int
	Total   int
	Jobs    []Job
	JobId   string
	Message string
}

// Returns the list of Jobs for project_id limited by count and those which
// match the filters
func (jobs *Jobs) List(conn *Connection, project_id string, count int, filters map[string]string) (*Jobs, error) {
	params := url.Values{}
	params.Add("project", project_id)
	if count > 0 {
		params.Add("count", strconv.Itoa(count))
	}
	for fname, fval := range filters {
		params.Add(fname, fval)
	}
	query_url, err := build_api_url(conn, "/jobs/list.json", &params)
	if err != nil {
		return nil, err
	}
	resp, err := conn.Get(query_url)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(content, jobs)

	if jobs.Status != "ok" {
		return nil, jobs_list_error
	}
	return jobs, nil
}

// Returns the job information in map object given the job_id
func (jobs *Jobs) JobInfo(conn *Connection, job_id string) (*Job, error) {
	result := re_jobid.FindStringSubmatch(job_id)
	if len(result) == 0 {
		return nil, wrong_job_id_error
	}
	project_id := result[1]

	params := url.Values{}
	params.Add("project", project_id)
	params.Add("job_id", job_id)
	query_url, err := build_api_url(conn, "/jobs/list.json", &params)
	if err != nil {
		return nil, err
	}
	resp, err := conn.Get(query_url)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(content, jobs)

	if jobs.Status != "ok" {
		return nil, errors.New("Jobs.JobInfo: Error while retrieving job info")
	}
	if len(jobs.Jobs) <= 0 {
		return nil, errors.New(fmt.Sprintf("Jobs.JobInfo: Job %s does not exist", job_id))
	}
	return &jobs.Jobs[0], nil
}

func project_data_map(project_id string, spider_name string, job_id string, args map[string]string) map[string][]string {
	data := map[string][]string{
		"project": []string{project_id},
	}
	if spider_name != "" {
		data["spider"] = []string{spider_name}
	}
	if job_id != "" {
		data["job"] = []string{job_id}
	}
	for k, v := range args {
		if k != "project" && k != "spider" {
			data[k] = []string{v}
		}
	}
	return data
}

// Schedule the spider with name `spider_name` and arguments `args` on `project_id`.
func (jobs *Jobs) Schedule(conn *Connection, project_id string, spider_name string, args map[string]string) (string, error) {
	method := "/schedule.json"
	data := project_data_map(project_id, spider_name, "", args)
	resp, err := conn.Post(conn.BaseUrl+method, data)
	if err != nil {
		return "", err
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	json.Unmarshal(content, jobs)

	if jobs.Status != "ok" {
		return "", errors.New(fmt.Sprintf("Jobs.Schedule: Error while scheduling the job. Message: %s", jobs.Message))
	}
	return jobs.JobId, nil
}

// Re-schedule the spider with `job_id` using the same tags and parameters
func (jobs *Jobs) Reschedule(conn *Connection, job_id string) (string, error) {
	result := re_jobid.FindStringSubmatch(job_id)
	if len(result) == 0 {
		return "", wrong_job_id_error
	}
	project_id := result[1]

	job, err := jobs.JobInfo(conn, job_id)
	if err != nil {
		return "", err
	}
	method := "/schedule.json"
	data := project_data_map(project_id, job.Spider, "", job.SpiderArgs)
	data["add_tag"] = job.Tags
	resp, err := conn.Post(conn.BaseUrl+method, data)
	if err != nil {
		return "", err
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	json.Unmarshal(content, jobs)

	if jobs.Status != "ok" {
		return "", errors.New(fmt.Sprintf("Jobs.Reschedule: Error while scheduling the job. Message: %s", jobs.Message))
	}
	return jobs.JobId, nil
}

func (jobs *Jobs) postAction(conn *Connection, job_id string, method string, error_string string, update_data map[string]string) error {
	result := re_jobid.FindStringSubmatch(job_id)
	if len(result) == 0 {
		return wrong_job_id_error
	}
	project_id := result[1]

	data := project_data_map(project_id, "", job_id, update_data)
	resp, err := conn.Post(conn.BaseUrl+method, data)
	if err != nil {
		return err
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	json.Unmarshal(content, jobs)
	if jobs.Status != "ok" {
		return errors.New(fmt.Sprintf("%s. Message: %s", error_string, jobs.Message))
	}
	return nil
}

// Stop the job with `job_id`.
func (jobs *Jobs) Stop(conn *Connection, job_id string) error {
	return jobs.postAction(conn, job_id, "/jobs/stop.json",
		"Jobs.Stop: Error while stopping the job", nil)
}

// Update the job with `job_id` with the `update_data`.
func (jobs *Jobs) Update(conn *Connection, job_id string, update_data map[string]string) error {
	return jobs.postAction(conn, job_id, "/jobs/update.json",
		"Jobs.Update: Error while updating the job", update_data)
}

// Delete the job with `job_id`.
func (jobs *Jobs) Delete(conn *Connection, job_id string) error {
	return jobs.postAction(conn, job_id, "/jobs/delete.json",
		"Jobs.Delete: Error while deleting the job", nil)
}

// Returns up to `count` items for the job `job_id`, starting at `offset`. Each
// item is returned as a map with string key but value of type `interface{}`
func RetrieveItems(conn *Connection, job_id string, count, offset int) ([]map[string]interface{}, error) {
	result := re_jobid.FindStringSubmatch(job_id)
	if len(result) == 0 {
		return nil, wrong_job_id_error
	}
	project_id := result[1]

	params := url.Values{}
	params.Add("project", project_id)
	params.Add("job", job_id)
	params.Add("offset", strconv.Itoa(offset))
	if count > 0 {
		params.Add("count", strconv.Itoa(count))
	}
	query_url, err := build_api_url(conn, "/items.json", &params)
	if err != nil {
		return nil, err
	}
	resp, err := conn.Get(query_url)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(resp.Body)
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
	for i, e := range jarray {
		items[i] = e.(map[string]interface{})
	}
	return items, nil
}

// Download the slybot project for the project `project_id` and the spiders given.
// The method write the zip file to `out` argument.
func RetrieveSlybotProject(conn *Connection, project_id string, spiders []string, out *os.File) error {
	method := fmt.Sprintf("/as/project-slybot.zip?project=%s", project_id)
	for _, spider := range spiders {
		method = method + fmt.Sprintf("&spider=%s", spider)
	}
	resp, err := conn.Get(conn.BaseUrl + method)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		if _, err := out.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}

// Retrieve an stream of lines from the connection `conn` and to API method `method`. `count`
// and `offset` parameters are availble to jump to any place of the stream (counting lines)
// on the position `offset`.
// It behaves reliable when the connection drops or when the API is not available.
func retrieveLinesStream(conn *Connection, method string, params *url.Values, count, offset int) (<-chan string, <-chan error) {
	const (
		BATCH_SIZE     = 1000
		MAX_RETRIES    = 3
		RETRY_INTERVAL = time.Second * 30
	)

	var in_count int = BATCH_SIZE
	out := make(chan string)
	errch := make(chan error)

	go func() {
		defer close(out)
		defer close(errch)

		for {
			if count < BATCH_SIZE {
				in_count = count
			}
			params.Set("offset", strconv.Itoa(offset))
			if in_count > 0 {
				params.Set("count", strconv.Itoa(in_count))
			}
			query_url, err := build_api_url(conn, method, params)
			if err != nil {
				close(out)
				errch <- err
			}
			resp, err := conn.Get(query_url)
			if err != nil {
				i := 0
				for (err != nil || resp.StatusCode >= 400) && i < MAX_RETRIES {
					time.Sleep(RETRY_INTERVAL)
					resp, err = conn.Get(query_url)
					i++
				}
				if i == MAX_RETRIES && err != nil {
					close(out)
					errch <- err
				}
			}
			scanner := bufio.NewScanner(resp.Body)
			retrieved := 0
			for scanner.Scan() {
				retrieved++
				out <- scanner.Text()
			}
			if scanner.Err() != nil {
				if retrieved == 0 && count > 0 {
					close(out)
					errch <- scanner.Err()
				}
				offset += retrieved
				count -= retrieved
			} else {
				offset += in_count
				count -= in_count
			}
			if count <= 0 {
				break
			}
		}
	}()
	return out, errch
}

// Helper method that check for job_id format, build the method using
// `project_id` extracted from `job_id` plus `job_id`.
// Is to avoid repeate the same code every time.
func asLineStream(conn *Connection, method string, params *url.Values, job_id string, count, offset int) (<-chan string, <-chan error) {
	result := re_jobid.FindStringSubmatch(job_id)
	if len(result) == 0 {
		errch := make(chan error)
		go func() {
			defer close(errch)
			errch <- wrong_job_id_error
		}()
		return nil, errch
	}
	project_id := result[1]

	params.Set("project", project_id)
	params.Set("job", job_id)

	return retrieveLinesStream(conn, method, params, count, offset)

}

//  Given a job_id, returns a channel of strings where each element is a line of
//  the JsonLines returned by the API items.jl endpoint.
//  Returns a channel with errors
func ItemsAsJsonLines(conn *Connection, job_id string, count, offset int) (<-chan string, <-chan error) {
	return asLineStream(conn, "items.jl", &url.Values{}, job_id, count, offset)
}

//  Given a job_id, returns a channel of strings where each element is a line of
//  the CSV returned by the API items.csv endpoint.
//  Returns a channel with errors
func ItemsAsCSV(conn *Connection, job_id string, count, offset int, include_headers bool, fields string) (<-chan string, <-chan error) {
	iih := 0
	if include_headers {
		iih = 1
	}
	params := url.Values{}
	params.Add("include_headers", strconv.Itoa(iih))
	params.Add("fields", fields)
	return asLineStream(conn, "items.csv", &params, job_id, count, offset)
}

// Returns a channel of strings which each element is a line of the log for job with `job_id`
// Count and offset parameters are accepted to paginate results.
//  Returns a channel with errors
func LogLines(conn *Connection, job_id string, count, offset int) (<-chan string, <-chan error) {
	return asLineStream(conn, "log.txt", &url.Values{}, job_id, count, offset)
}

// Returns a channel of strings which each element is a JSON serialized job for
// the project `project_id`. `count` and filters (a list of string of the type
// key=value to apply to the result (see http://doc.scrapinghub.com/api.html#jobs-list-json)
//  Returns a channel with errors
func JobsAsJsonLines(conn *Connection, project_id string, count, offset int, filters map[string]string) (<-chan string, <-chan error) {
	params := url.Values{}
	params.Add("project", project_id)
	for fname, fval := range filters {
		params.Add(fname, fval)
	}
	return retrieveLinesStream(conn, "/jobs/list.jl", &params, count, offset)
}

/*
  Eggs API methods
*/

type Egg struct {
	Name    string
	Version string
}
type Eggs struct {
	Status  string
	Message string
	EggData Egg   `json:"egg"`
	EggList []Egg `json:"eggs"`
}

// Add a python egg to the project `project_id` with `name` and `version` given.
func (eggs *Eggs) Add(conn *Connection, project_id, name, version, egg_path string) (*Egg, error) {
	params := map[string][]string{
		"project": []string{project_id},
		"name":    []string{name},
		"version": []string{version},
	}
	method := "/eggs/add.json"
	resp, err := conn.PostFiles(conn.BaseUrl+method, params, map[string]string{"egg": egg_path})
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(content, eggs)

	if eggs.Status != "ok" {
		return nil, errors.New(fmt.Sprintf("Eggs.Add: Error ocurred while uploading egg: %s", eggs.Message))
	}
	return &eggs.EggData, nil
}

// Delete the egg `egg_name` from project `project_id`
func (eggs *Eggs) Delete(conn *Connection, project_id, egg_name string) error {
	method := "/eggs/delete.json"
	params := map[string][]string{
		"project": []string{project_id},
		"name":    []string{egg_name},
	}
	resp, err := conn.Post(conn.BaseUrl+method, params)
	if err != nil {
		return err
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	json.Unmarshal(content, eggs)
	if eggs.Status != "ok" {
		return errors.New(fmt.Sprintf("Eggs.Delete: Error ocurred while deleting the egg: ", eggs.Message))
	}
	return nil
}

// List all the eggs in the project `project_id`
func (eggs *Eggs) List(conn *Connection, project_id string) ([]Egg, error) {
	method := fmt.Sprintf("/eggs/list.json?project=%s", project_id)
	resp, err := conn.Get(conn.BaseUrl + method)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(content, eggs)
	if eggs.Status != "ok" {
		return nil, errors.New(fmt.Sprintf("Eggs.List: Error ocurred while listing the project <%s> eggs: %s", project_id, eggs.Message))
	}
	return eggs.EggList, nil
}

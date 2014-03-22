package scrapinghub

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Type to make easier handle the operations of retrieve
// streams of lines of API call methods
type LinesStream struct {
	Conn   *Connection
	Count  int
	Offset int
}

// Retrieve an stream of lines from the connection `conn` and to API method `method`. `count`
// and `offset` parameters are availble to jump to any place of the stream (counting lines)
// on the position `offset`.
// It behaves reliable when the connection drops or when the API is not available.
func (ls *LinesStream) asLinesStream(method string, params *url.Values) (<-chan string, <-chan error) {
	const (
		BATCH_SIZE     = 1000
		MAX_RETRIES    = 3
		RETRY_INTERVAL = time.Second * 30
	)

	out := make(chan string)
	errch := make(chan error)

	go func() {
		defer close(out)
		defer close(errch)

		var resp *http.Response
		var err error
		count := ls.Count
		offset := ls.Offset
		in_count := BATCH_SIZE
		scan_retries := 1

		for {
			if count < BATCH_SIZE {
				in_count = count
			}
			params.Set("offset", strconv.Itoa(offset))
			if in_count > 0 {
				params.Set("count", strconv.Itoa(in_count))
			}

			i := 1
			for {
				resp, err = ls.Conn.APICall(method, GET, params)
				if err == nil && resp != nil && resp.StatusCode < 400 {
					break
				}
				time.Sleep(RETRY_INTERVAL)
				i++
				if i == MAX_RETRIES {
					close(out)
					errch <- fmt.Errorf("Max retries reached: %d, internal error message : %v\n", MAX_RETRIES, err)
					return
				}
			}

			scanner := bufio.NewScanner(resp.Body)
			retrieved := 0
			for scanner.Scan() {
				retrieved++
				out <- scanner.Text()
			}
			if scanner.Err() != nil {
				if retrieved == 0 {
					scan_retries++
					if scan_retries == MAX_RETRIES {
						close(out)
						errch <- scanner.Err()
						return
					}
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

// Make an API call to `method` and paramaeters `params` but using an
// Scrapinghub job_id.
func (ls *LinesStream) withJobID(method string, params *url.Values, job_id string) (<-chan string, <-chan error) {
	if err := ValidateJobID(job_id); err != nil {
		return emptyStringChan(), fromErrToErrChan(err)
	} else {
		params.Set("job", job_id)
		params.Set("project", ProjectID(job_id))
		return ls.asLinesStream(method, params)
	}
}

// Make an API call to `method` and paramaeters `params` but using an
// Scrapinghub project_id.
func (ls *LinesStream) withProjectID(method string, params *url.Values, project_id string) (<-chan string, <-chan error) {
	if err := ValidateProjectID(project_id); err != nil {
		return emptyStringChan(), fromErrToErrChan(err)
	} else {
		params.Set("project", project_id)
		return ls.asLinesStream(method, params)
	}
}

//  Given a job_id, returns a channel of strings where each element is a line of
//  the JsonLines returned by the API items.jl endpoint.
//  Returns a channel with errors
func (ls *LinesStream) ItemsAsJsonLines(job_id string) (<-chan string, <-chan error) {
	return ls.withJobID("items.jl", &url.Values{}, job_id)
}

//  Given a job_id, returns a channel of strings where each element is a line of
//  the CSV returned by the API items.csv endpoint.
//  Returns a channel with errors
func (ls *LinesStream) ItemsAsCSV(job_id string, include_headers bool, fields string) (<-chan string, <-chan error) {
	iih := 0
	if include_headers {
		iih = 1
	}
	params := url.Values{}
	params.Add("include_headers", strconv.Itoa(iih))
	params.Add("fields", fields)
	return ls.withJobID("items.csv", &params, job_id)
}

// Returns a channel of strings which each element is a line of the log for job with `job_id`
// Count and offset parameters are accepted to paginate results.
//  Returns a channel with errors
func (ls *LinesStream) LogLines(job_id string) (<-chan string, <-chan error) {
	return ls.withJobID("log.txt", &url.Values{}, job_id)
}

// Returns a channel of strings which each element is a JSON serialized job for
// the project `project_id`. `count` and filters (a list of string of the type
// key=value to apply to the result (see http://doc.scrapinghub.com/api.html#jobs-list-json)
//  Returns a channel with errors
func (ls *LinesStream) JobsAsJsonLines(project_id string, filters map[string]string) (<-chan string, <-chan error) {
	params := url.Values{}
	for fname, fval := range filters {
		params.Add(fname, fval)
	}
	return ls.withProjectID("/jobs/list.jl", &params, project_id)
}

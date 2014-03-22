package scrapinghub

import (
	"io"
	"net/url"
	"os"
)

// Download the slybot project for the project `project_id` and the spiders given.
// The method write the zip file to `out` argument.
func RetrieveSlybotProject(conn *Connection, project_id string, spiders []string, out *os.File) error {
	params := url.Values{}
	params.Add("project", project_id)
	for _, spider := range spiders {
		params.Set("spider", spider)
	}

	resp, err := conn.APICall("/as/project-slybot.zip", GET, &params)
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

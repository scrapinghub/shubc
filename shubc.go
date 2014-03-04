package main

import (
	"flag"
	"fmt"
	"github.com/andrix/shubc/scrapinghub"
	"os"
	"os/user"
	"path"
	"strings"
)

func dashes(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "-"
	}
	return s
}

func find_apikey() string {
	u, _ := user.Current()
	scrapy_cfg := path.Join(u.HomeDir, "/.scrapy.cfg")
	if st, err := os.Stat(scrapy_cfg); err == nil {
		f, err := os.Open(scrapy_cfg)
		if err != nil {
			panic(err)
		}
		buf := make([]byte, st.Size())
		n, err := f.Read(buf)
		if err != nil {
			panic(err)
		}
		s := string(buf[:n])
		lines := strings.Split(s, "\n")
		for _, l := range lines {
			if strings.Index(l, "username") < 0 {
				continue
			}
			result := strings.Split(l, "=")
			return strings.TrimSpace(result[1])
		}
	}
	return ""
}

// Returns a map given a list of ["key=value", ...] strings
func equality_list_to_map(data []string) map[string]string {
	result := make(map[string]string)
	for _, e := range data {
		if strings.Index(e, "=") > 0 {
			res := strings.Split(e, "=")
			result[strings.TrimSpace(res[0])] = strings.TrimSpace(res[1])
		}
	}
	return result
}

func main() {
	var apikey = flag.String("apikey", find_apikey(), "Scrapinghub api key")
	var count = flag.Int("count", 100, "Count for those commands that need a count limit")
	var offset = flag.Int("offset", 0, "Number of results to skip from the beginning")
	var output = flag.String("o", "", "Write output to a file instead of Stdout")
	var jsonlines = flag.Bool("jl", false, "If given, for command items and jobs will retrieve all the data writing to os.Stdout as JsonLines format")

	flag.Parse()

	if len(flag.Args()) <= 0 {
		fmt.Printf("Usage: shubc [options] url\n")
	} else {
		// Create new connection
		var conn scrapinghub.Connection
		conn.New(*apikey)

		cmd := flag.Arg(0)

		if cmd == "help" {
			fmt.Println("shubc [options] <command> arg1 .. argN")
			fmt.Println()
			fmt.Println(" Options: ")
			flag.PrintDefaults()
			fmt.Println()
			fmt.Println(" Commands: ")
			fmt.Println("   delete <job_id>                            - delete the job with <job_id>")
			fmt.Println("   items <job_id>                             - print to stdout the items for <job_id> (count & offset available)")
			fmt.Println("   jobs <project_id> [filters]                - list the last 100 jobs on project_id")
			fmt.Println("   jobinfo <job_id>                           - print information about the job with <job_id>")
			fmt.Println("   log <job_id>                               - print to stdout the log for the job `job_id` (count & offset available)")
			fmt.Println("   project-slybot <project_id> [spiders]      - download the zip and write it to Stdout or o.zip if -o option is given")
			fmt.Println("   schedule <project_id> <spider_name> [args] - schedule the spider <spider_name> with [args] in project <project_id>")
			fmt.Println("   spiders <project_id>                       - list the spiders on project_id")
			fmt.Println("   stop <job_id>                              - stop the job with <job_id>")
			fmt.Println("   update <job_id> [args]                     - update the job with <job_id> using the `args` given")

		} else {
			if *apikey == "" {
				fmt.Println("No API Key given, neither through the option or in ~/.scrapy.cfg")
			}
			if cmd == "spiders" {
				var spiders scrapinghub.Spiders
				spider_list, err := spiders.List(&conn, flag.Arg(1))

				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				} else {
					fmt.Printf("| %30s | %10s | %20s |\n", "name", "type", "version")
					fmt.Println(dashes(70))
					for _, spider := range spider_list.Spiders {
						fmt.Printf("| %30s | %10s | %20s |\n", spider["id"], spider["type"], spider["version"])
					}
				}
			} else if cmd == "jobs" {
				project_id := flag.Arg(1)
				if len(flag.Args()) < 2 {
					fmt.Println("Missing argument <project_id>")
					os.Exit(1)
				}
				filters := equality_list_to_map(flag.Args()[2:])
				if *jsonlines {
					ch_jobs, err := scrapinghub.JobsAsJsonLines(&conn, project_id, *count, filters)
					if err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
					for line := range ch_jobs {
						fmt.Println(line)
					}
				} else {
					var jobs scrapinghub.Jobs
					jobs_list, err := jobs.List(&conn, project_id, *count, filters)

					if err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
					outfmt := "| %10s | %25s | %12s | %10s | %10s | %20s |\n"
					fmt.Printf(outfmt, "id", "spider", "state", "items", "errors", "started_time")
					fmt.Println(dashes(106))
					for _, j := range jobs_list.Jobs {
						fmt.Printf("| %10s | %25s | %12s | %10d | %10d | %20s |\n", j.Id, j.Spider, j.State,
							j.ItemsScraped, j.ErrorsCount, j.StartedTime)
					}
				}
			} else if cmd == "jobinfo" {
				var jobs scrapinghub.Jobs
				jobinfo, err := jobs.JobInfo(&conn, flag.Arg(1))

				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				} else {
					outfmt := "| %-30s | %60s |\n"
					fmt.Printf(outfmt, "key", "value")
					fmt.Println(dashes(97))
					fmt.Printf(outfmt, "id", jobinfo.Id)
					fmt.Printf(outfmt, "spider", jobinfo.Spider)
					fmt.Printf(outfmt, "spider_args", "")
					for k, v := range jobinfo.SpiderArgs {
						fmt.Printf(outfmt, " ", fmt.Sprintf("%s = %s", k, v))
					}
					fmt.Printf(outfmt, "spider_type", jobinfo.SpiderType)
					fmt.Printf(outfmt, "state", jobinfo.State)
					fmt.Printf(outfmt, "close_reason", jobinfo.CloseReason)
					fmt.Println(dashes(97))
					fmt.Printf(outfmt, "responses_received", fmt.Sprintf("%d", jobinfo.ResponsesReceived))
					fmt.Printf(outfmt, "items_scraped", fmt.Sprintf("%d", jobinfo.ItemsScraped))
					fmt.Printf(outfmt, "errors_count", fmt.Sprintf("%d", jobinfo.ErrorsCount))
					fmt.Printf(outfmt, "logs", fmt.Sprintf("%d", jobinfo.Logs))
					fmt.Println(dashes(97))
					fmt.Printf(outfmt, "started_time", jobinfo.StartedTime)
					fmt.Printf(outfmt, "updated_time", jobinfo.UpdatedTime)
					fmt.Printf(outfmt, "elapsed", fmt.Sprintf("%d", jobinfo.Elapsed))
					fmt.Printf(outfmt, "tags", " ")
					for _, e := range jobinfo.Tags {
						fmt.Printf(outfmt, " ", e)
					}
					fmt.Printf(outfmt, "priority", fmt.Sprintf("%d", jobinfo.Priority))
					fmt.Printf(outfmt, "version", jobinfo.Version)
					fmt.Println(dashes(97))
				}
			} else if cmd == "schedule" {
				var jobs scrapinghub.Jobs
				project_id := flag.Arg(1)
				spider_name := flag.Arg(2)
				args := equality_list_to_map(flag.Args()[3:])
				job_id, err := jobs.Schedule(&conn, project_id, spider_name, args)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				} else {
					fmt.Printf("Scheduled job: %s\n", job_id)
				}
			} else if cmd == "stop" {
				var jobs scrapinghub.Jobs
				job_id := flag.Arg(1)
				err := jobs.Stop(&conn, job_id)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				} else {
					fmt.Printf("Stopped job: %s\n", job_id)
				}
			} else if cmd == "update" {
				var jobs scrapinghub.Jobs
				job_id := flag.Arg(1)
				update_data := equality_list_to_map(flag.Args()[2:])
				err := jobs.Update(&conn, job_id, update_data)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				} else {
					fmt.Printf("Updated job: %s\n", job_id)
				}
			} else if cmd == "delete" {
				var jobs scrapinghub.Jobs
				job_id := flag.Arg(1)
				err := jobs.Delete(&conn, job_id)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				} else {
					fmt.Printf("Deleted job: %s\n", job_id)
				}
			} else if cmd == "items" {
				job_id := flag.Arg(1)
				if *jsonlines {
					ch_lines, err := scrapinghub.ItemsAsJsonLines(&conn, job_id)
					if err != nil {
						fmt.Printf("Error: %s\n", err)
						os.Exit(1)
					}
					for line := range ch_lines {
						fmt.Println(line)
					}
				} else {
					items, err := scrapinghub.RetrieveItems(&conn, job_id, *count, *offset)
					if err != nil {
						fmt.Printf("Error: %s\n", err)
						os.Exit(1)
					}
					for i, e := range items {
						fmt.Printf("Item %5d %s\n", i, dashes(129))
						for k, v := range e {
							fmt.Printf("| %-33s | %100s |\n", k, fmt.Sprintf("%v", v))
						}
						fmt.Println(dashes(140))
					}
				}
			} else if cmd == "project-slybot" {
				project_id := flag.Arg(1)
				spiders := flag.Args()[2:]

				var out *os.File = os.Stdout
				var err error
				if *output != "" {
					out, err = os.Create(*output)
					if err != nil {
						fmt.Printf("Error writing to file: %s\n", err)
						os.Exit(1)
					}
				}
				defer func() {
					if err := out.Close(); err != nil {
						panic(err)
					}
				}()

				err = scrapinghub.RetrieveSlybotProject(&conn, project_id, spiders, out)

				if err != nil {
					fmt.Printf("Error: %s\n", err)
					os.Exit(1)
				}
			} else if cmd == "log" {
				job_id := flag.Arg(1)
				ch_lines, err := scrapinghub.LogLines(&conn, job_id, *count, *offset)
				if err != nil {
					fmt.Printf("Error: %s\n", err)
					os.Exit(1)
				}
				for line := range ch_lines {
					fmt.Println(line)
				}
			} else if cmd == "reschedule" {
				var jobs scrapinghub.Jobs
				job_id := flag.Arg(1)
				new_job_id, err := jobs.Reschedule(&conn, job_id)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				} else {
					fmt.Printf("Re-scheduled job new id: %s\n", new_job_id)
				}
			} else {
				fmt.Printf("'%s' command not found\n", cmd)
				os.Exit(1)
			}

		}
	}
}

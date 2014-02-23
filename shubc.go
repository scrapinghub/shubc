package main

import (
    "os"
    "os/user"
    "fmt"
    "path"
    "flag"
    "strings"
    "github.com/andrix/shubc/scrapinghub"
)

func dashes(n int) string {
    s := ""
    for i:=0; i < n; i++ {
        s += "-"
    }
    return s
}

func find_apikey() string {
    u, _ := user.Current()
    scrapy_cfg := path.Join(u.HomeDir, "/.scrapy.cfg")
    if st, err := os.Stat(scrapy_cfg); err == nil {
        f, err := os.Open(scrapy_cfg)
        if err != nil { panic(err) }
        buf := make([]byte, st.Size())
        n, err := f.Read(buf)
        if err != nil { panic(err) }
        s := string(buf[:n])
        lines := strings.Split(s, "\n")
        for _, l := range(lines) {
            if strings.Index(l, "username") < 0 {
                continue
            }
            result := strings.Split(l, "=")
            return strings.TrimSpace(result[1])
        }
    }
    return ""
}

func main() {
    var apikey = flag.String("apikey", find_apikey(), "Scrapinghub api key")
    var count = flag.Int("count", 100, "Count for those commands that need a count limit")
    var offset = flag.Int("offset", 0, "Number of results to skip from the beginning")
    var output = flag.String("o", "", "Write output to a file instead of Stdout")
    var raw = flag.Bool("raw", false, "If given, for command items and jobs will retrieve all the data writing to os.Stdout as raw format")

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
            fmt.Println(" Commands: ")
            fmt.Println("   spiders <project_id>                       - list the spiders on project_id")
            fmt.Println("   jobs <project_id> [filters]                - list the last 100 jobs on project_id")
            fmt.Println("   jobinfo <job_id>                           - print information about the job with <job_id>")
            fmt.Println("   schedule <project_id> <spider_name> [args] - schedule the spider <spider_name> with [args] in project <project_id>")
            fmt.Println("   stop <job_id>                              - stop the job with <job_id>")
            fmt.Println("   items <job_id>                             - print to stdout the items for <job_id> (count & offset available)")
            fmt.Println("   project-slybot <project_id> [spiders] - download the zip and write it to Stdout or o.zip if -o option is given")

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
                    for _, spider := range(spider_list.Spiders) {
                        fmt.Printf("| %30s | %10s | %20s |\n", spider["id"], spider["type"], spider["version"])
                    }
                }
            } else if cmd == "jobs" {
                project_id := flag.Arg(1)
                if len(flag.Args()) < 2 {
                    fmt.Println("Missing argument <project_id>")
                    os.Exit(1)
                }
                filters := flag.Args()[2:]
                if *raw {
                    ch_jobs, err := scrapinghub.RetrieveJobsJsonLines(&conn, project_id, *count, filters)
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
                    started_time := ""
                    for _, j := range(jobs_list.Jobs) {
                        jid := j["id"].(string)
                        if j["started_time"] != nil {
                           started_time = j["started_time"].(string)
                        }
                        fmt.Printf("| %10s | %25s | %12s | %10d | %10d | %20s |\n", jid, j["spider"].(string), j["state"].(string), 
                            int(j["items_scraped"].(float64)), int(j["errors_count"].(float64)), started_time)
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
                    for k, v := range(jobinfo) {
                        fmt.Printf(outfmt, k, v)
                    }
                }
            } else if cmd == "schedule" {
                var jobs scrapinghub.Jobs
                project_id := flag.Arg(1)
                spider_name := flag.Arg(2)
                args := flag.Args()[3:]
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
            } else if cmd == "items" {
                job_id := flag.Arg(1)
                if *raw {
                    ch_lines, err := scrapinghub.RetrieveItemsJsonLines(&conn, job_id)
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
                    for i, e := range(items) {
                        fmt.Printf("Item %5d %s\n", i, dashes(129))
                        for k, v := range(e) {
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
            } else {
                fmt.Printf("'%s' command not found\n", cmd)
                os.Exit(1)
            }

        }
    }
}




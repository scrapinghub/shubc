package main

import (
    "os"
    "os/user"
    "fmt"
    "path"
    "flag"
    "strings"
    "shubc/scrapinghub"
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
            fmt.Println("   spiders <project_id>  - list the spiders on project_id")
            fmt.Println("   jobs <project_id>     - list the last 100 jobs on project_id")
            fmt.Println("   jobinfo <job_id>      - print information about the job with <job_id>")

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
                var jobs scrapinghub.Jobs
                if len(flag.Args()) < 2 {
                    fmt.Println("Missing argument <project_id>")
                    os.Exit(1)
                }
                filters := flag.Args()[2:]
                jobs_list, err := jobs.List(&conn, flag.Arg(1), *count, filters)

                if err != nil {
                    fmt.Println(err)
                    os.Exit(1)
                } else {
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
            } else {
                fmt.Printf("'%s' command not found\n", cmd)
                os.Exit(1)
            }

        }
    }
}




package main

import (
    "os"
    "fmt"
    "flag"
    "shubc/scrapinghub"
)

func dashes(n int) string {
    s := ""
    for i:=0; i < n; i++ {
        s += "-"
    }
    return s
}

func main() {
    var apikey = flag.String("apikey", "", "Scrapinghub api key")
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
            fmt.Println("   spiders <project_id> - list the spiders on project_id")
            fmt.Println("   jobs <project_id> - list the last 100 jobs on project_id")

        } else {
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
                filters := flag.Args()[2:]
                jobs_list, err := jobs.List(&conn, flag.Arg(1), *count, filters)

                if err != nil {
                    fmt.Println(err)
                    os.Exit(1)
                } else {
                    outfmt := "| %10s | %25s | %12s | %10s | %10s | %20s |\n"
                    fmt.Printf(outfmt, "id", "spider", "state", "items", "errors", "started_time")
                    fmt.Println(dashes(106))
                    for _, j := range(jobs_list.Jobs) {
                        fmt.Printf("| %10s | %25s | %12s | %10d | %10d | %20s |\n", j["id"].(string), j["spider"].(string), j["state"].(string), 
                        int(j["items_scraped"].(float64)), int(j["errors_count"].(float64)), j["started_time"].(string))
                    }
                }
            }
        }
    }
}




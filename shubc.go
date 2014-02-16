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
            }
        }
    }
}




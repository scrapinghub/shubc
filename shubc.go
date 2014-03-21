package main

import (
	"flag"
	"fmt"
	"github.com/scrapinghub/shubc/scrapinghub"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var re_egg_pattern = regexp.MustCompile(`(.+?)-(\d(?:\.\d)*)-?.*`)

type CmdFun func(conn *scrapinghub.Connection, args []string, flags *PFlags)

func dashes(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "-"
	}
	return s
}

func find_apikey() string {
	if os.Getenv("SH_APIKEY") != "" {
		return os.Getenv("SH_APIKEY")
	}

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

type PFlagsCSV struct {
	IncludeHeaders bool
	Fields         string
}

type PFlags struct {
	Count       int
	Offset      int
	Output      string
	AsJsonLines bool
	AsCSV       bool
	CSVFlags    PFlagsCSV
}

/** Commands **/

func cmd_help() {
	fmt.Println("shubc [options] <command> arg1 .. argN")
	fmt.Println()
	fmt.Println(" Options: ")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println(" Commands: ")

	fmt.Println("   Spiders API: ")
	fmt.Println("     spiders <project_id>                       - list the spiders on project_id")

	fmt.Println("   Jobs API: ")
	fmt.Println("     schedule <project_id> <spider_name> [args] - schedule the spider <spider_name> with [args] in project <project_id>")
	fmt.Println("     reschedule <job_id>                        - re-schedule the job `job_id` with the same arguments and tags")
	fmt.Println("     jobs <project_id> [filters]                - list the last 100 jobs on project_id")
	fmt.Println("     jobinfo <job_id>                           - print information about the job with <job_id>")
	fmt.Println("     update <job_id> [args]                     - update the job with <job_id> using the `args` given")
	fmt.Println("     stop <job_id>                              - stop the job with <job_id>")
	fmt.Println("     delete <job_id>                            - delete the job with <job_id>")

	fmt.Println("   Items API: ")
	fmt.Println("     items <job_id>                             - print to stdout the items for <job_id> (count & offset available)")

	fmt.Println("   Logs API: ")
	fmt.Println("     log <job_id>                               - print to stdout the log for the job `job_id` (count & offset available)")

	fmt.Println("   Eggs API: ")
	fmt.Println("     eggs-add <project_id> <path> [name=n version=v] - add the egg in `path` to the project `project_id`. By default it guess the name and version from `path`, but can be given using name=eggname and version=XXX.")
	fmt.Println("     eggs-list <project_id>                          - list the eggs in `project_id`")
	fmt.Println("     eggs-delete <project_id> <egg_name>             - delete the egg `egg_name` in the project `project_id`")

	fmt.Println("   Autoscraping API: ")
	fmt.Println("     project-slybot <project_id> [spiders]      - download the zip and write it to Stdout or o.zip if -o option is given")
}

func cmd_spiders(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		fmt.Printf("Missing argument: <project_id>\n")
		os.Exit(1)
	}
	project_id := args[0]
	var spiders scrapinghub.Spiders
	spider_list, err := spiders.List(conn, project_id)

	if err != nil {
		fmt.Printf("spiders error: %s\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("| %30s | %10s | %20s |\n", "name", "type", "version")
		fmt.Println(dashes(70))
		for _, spider := range spider_list.Spiders {
			fmt.Printf("| %30s | %10s | %20s |\n", spider["id"], spider["type"], spider["version"])
		}
	}
}

func cmd_jobs(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		fmt.Printf("Missing argument: <project_id>\n")
		os.Exit(1)
	}
	project_id := args[0]
	filters := equality_list_to_map(args[1:])

	count := flags.Count
	offset := flags.Offset

	if flags.AsJsonLines {
		ch_jobs, errch := scrapinghub.JobsAsJsonLines(conn, project_id, count, offset, filters)
		for line := range ch_jobs {
			fmt.Println(line)
		}
		for err := range errch {
			fmt.Printf("jobs error: %s\n", err)
			os.Exit(1)
		}
	} else {
		var jobs scrapinghub.Jobs
		jobs_list, err := jobs.List(conn, project_id, count, filters)
		if err != nil {
			fmt.Printf("jobs error: %s", err)
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
}

func cmd_jobinfo(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		fmt.Printf("Missing argument: <job_id>\n")
		os.Exit(1)
	}
	job_id := args[0]

	var jobs scrapinghub.Jobs
	jobinfo, err := jobs.JobInfo(conn, job_id)

	if err != nil {
		fmt.Printf("jobinfo error: %s\n", err)
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
}

func cmd_schedule(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 2 {
		fmt.Printf("Missing arguments: <project_id> and <spider_name>\n")
		os.Exit(1)
	}
	var jobs scrapinghub.Jobs
	project_id := args[0]
	spider_name := args[1]
	spider_args := equality_list_to_map(args[2:])
	job_id, err := jobs.Schedule(conn, project_id, spider_name, spider_args)

	if err != nil {
		fmt.Printf("schedule error: %s\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("Scheduled job: %s\n", job_id)
	}
}

func cmd_jobs_stop(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		fmt.Printf("Missing argument: <job_id>\n")
		os.Exit(1)
	}
	var jobs scrapinghub.Jobs
	job_id := args[0]
	err := jobs.Stop(conn, job_id)
	if err != nil {
		fmt.Println("stop error: %s\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("Stopped job: %s\n", job_id)
	}
}

func cmd_jobs_update(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		fmt.Printf("Missing argument: <job_id>\n")
		os.Exit(1)
	}

	var jobs scrapinghub.Jobs

	job_id := args[0]
	update_data := equality_list_to_map(args[1:])

	err := jobs.Update(conn, job_id, update_data)
	if err != nil {
		fmt.Printf("update error: %s\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("Updated job: %s\n", job_id)
	}
}

func cmd_jobs_delete(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		fmt.Printf("Missing argument: <job_id>\n")
		os.Exit(1)
	}

	var jobs scrapinghub.Jobs
	job_id := args[0]
	err := jobs.Delete(conn, job_id)

	if err != nil {
		fmt.Printf("delete error: %s\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("Deleted job: %s\n", job_id)
	}

}

func cmd_items(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		fmt.Printf("Missing argument: <job_id>\n")
		os.Exit(1)
	}

	job_id := args[0]
	count := flags.Count
	offset := flags.Offset

	if flags.AsJsonLines {
		ch_lines, errch := scrapinghub.ItemsAsJsonLines(conn, job_id, count, offset)
		for line := range ch_lines {
			fmt.Println(line)
		}
		for err := range errch {
			fmt.Printf("items error: %s\n", err)
			os.Exit(1)
		}
	} else if flags.AsCSV {
		ch_lines, errch := scrapinghub.ItemsAsCSV(conn, job_id, count, offset,
			flags.CSVFlags.IncludeHeaders, flags.CSVFlags.Fields)
		for line := range ch_lines {
			fmt.Println(line)
		}
		for err := range errch {
			fmt.Printf("items error: %s\n", err)
			os.Exit(1)
		}
	} else {
		items, err := scrapinghub.RetrieveItems(conn, job_id, count, offset)
		if err != nil {
			fmt.Printf("items error: %s\n", err)
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
}

func cmd_as_project_slybot(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		fmt.Printf("Missing argument: <project_id>\n")
		os.Exit(1)
	}

	project_id := args[0]
	spiders := args[1:]
	output := flags.Output

	var out *os.File = os.Stdout
	var err error
	if output != "" {
		out, err = os.Create(output)
		if err != nil {
			fmt.Printf("project slybot error: fail to write to file: %s\n", err)
			os.Exit(1)
		}
	}
	defer func() {
		if err := out.Close(); err != nil {
			panic(err)
		}
	}()
	err = scrapinghub.RetrieveSlybotProject(conn, project_id, spiders, out)
	if err != nil {
		fmt.Printf("project-slybot error: %s\n", err)
		os.Exit(1)
	}
}

func cmd_log(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		fmt.Printf("Missing argument: <job_id>\n")
		os.Exit(1)
	}
	job_id := args[0]

	count := flags.Count
	offset := flags.Offset

	ch_lines, ch_err := scrapinghub.LogLines(conn, job_id, count, offset)
	for line := range ch_lines {
		fmt.Println(line)
	}
	for err := range ch_err {
		fmt.Printf("log error: %s\n", err)
		os.Exit(1)
	}
}

func cmd_reschedule(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		fmt.Printf("Missing argument: <job_id>\n")
		os.Exit(1)
	}
	job_id := args[0]

	var jobs scrapinghub.Jobs
	new_job_id, err := jobs.Reschedule(conn, job_id)
	if err != nil {
		fmt.Printf("reschedule error: %s\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("Re-scheduled job new id: %s\n", new_job_id)
	}
}

func cmd_eggs_add(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 2 {
		fmt.Printf("Missing arguments: <project_id> and <egg_path>\n")
		os.Exit(1)
	}
	project_id := args[0]
	egg_path := args[1]
	name_version := args[2:]

	var egg_name string
	var egg_ver string

	if len(name_version) > 0 {
		name_ver_map := equality_list_to_map(name_version)
		egg_name = name_ver_map["name"]
		egg_ver = name_ver_map["version"]
	} else {
		result := re_egg_pattern.FindStringSubmatch(filepath.Base(egg_path))
		if len(result) <= 0 {
			fmt.Println("eggs-add error: Can't guess the name and version from egg path filename, provide it using name=<name> and version=<version> as parameters.")
			os.Exit(1)
		}
		egg_name = result[1]
		egg_ver = result[2]
	}
	if egg_name == "" || egg_ver == "" {
		fmt.Println("Error: name and version are required")
		os.Exit(1)
	}
	var eggs scrapinghub.Eggs
	eggdata, err := eggs.Add(conn, project_id, egg_name, egg_ver, egg_path)
	if err != nil {
		fmt.Printf("eggs-add error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Egg uploaded successfully! Project: %s, Egg name: %s, version: %s\n", project_id, eggdata.Name, eggdata.Version)
}

func cmd_eggs_list(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		fmt.Printf("Missing argument: <project_id>\n")
		os.Exit(1)
	}
	project_id := args[0]

	var eggs scrapinghub.Eggs

	egglist, err := eggs.List(conn, project_id)
	if err != nil {
		fmt.Printf("eggs-list error: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(dashes(97))
	outfmt := "| %-30s | %60s |\n"
	fmt.Printf(outfmt, "Name", "Version")
	fmt.Println(dashes(97))
	for _, egg := range egglist {
		fmt.Printf(outfmt, egg.Name, egg.Version)
	}
	fmt.Println(dashes(97))
}

func cmd_eggs_delete(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 2 {
		fmt.Printf("Missing arguments: <project_id> and <egg_name>\n")
		os.Exit(1)
	}
	project_id := args[0]
	egg_name := args[1]

	var eggs scrapinghub.Eggs
	err := eggs.Delete(conn, project_id, egg_name)
	if err != nil {
		fmt.Printf("eggs-delete error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Egg %s successfully deleted from project: %s\n", egg_name, project_id)
}

func main() {
	var apikey = flag.String("apikey", find_apikey(), "Scrapinghub api key")
	var gflags PFlags

	count := flag.Int("count", 0, "Count for those commands that need a count limit")
	offset := flag.Int("offset", 0, "Number of results to skip from the beginning")
	output := flag.String("o", "", "Write output to a file instead of Stdout")
	fjl := flag.Bool("jl", false, "If given, for command items and jobs will retrieve all the data writing to os.Stdout as JsonLines format")

	fcsv := flag.Bool("csv", false, "If given, for command items, they will retrieve as CSV writing to os.Stdout")
	fincheads := flag.Bool("include_headers", false, "When -csv given, include the headers of the CSV in the output")
	fcsv_fields := flag.String("fields", "", "When -csv given, list of comma separated fields to include in the CSV")

	flag.Usage = cmd_help

	flag.Parse()

	// Set flags
	gflags.Count = *count
	gflags.Offset = *offset
	gflags.Output = *output

	gflags.AsJsonLines = *fjl
	gflags.AsCSV = *fcsv
	gflags.CSVFlags.IncludeHeaders = *fincheads
	gflags.CSVFlags.Fields = *fcsv_fields

	commands := map[string]CmdFun{
		"spiders":        cmd_spiders,
		"jobs":           cmd_jobs,
		"jobinfo":        cmd_jobinfo,
		"schedule":       cmd_schedule,
		"stop":           cmd_jobs_stop,
		"update":         cmd_jobs_update,
		"delete":         cmd_jobs_delete,
		"items":          cmd_items,
		"project-slybot": cmd_as_project_slybot,
		"log":            cmd_log,
		"reschedule":     cmd_reschedule,
		"eggs-add":       cmd_eggs_add,
		"eggs-list":      cmd_eggs_list,
		"eggs-delete":    cmd_eggs_delete,
	}

	if len(flag.Args()) <= 0 {
		fmt.Printf("Usage: shubc [options] url\n")
	} else {
		// Create new connection
		var conn scrapinghub.Connection
		conn.New(*apikey)

		cmd := flag.Arg(0)
		args := flag.Args()[1:]
		if cmd == "help" {
			cmd_help()
		} else {
			if cmd_func, ok := commands[cmd]; ok {
				if *apikey == "" {
					fmt.Println("No API Key given, neither through the option or in ~/.scrapy.cfg")
					os.Exit(1)
				}
				cmd_func(&conn, args, &gflags)
			} else {
				fmt.Printf("'%s' command not found\n", cmd)
				os.Exit(1)
			}
		}
	}
}

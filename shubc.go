package main

import (
	"flag"
	"fmt"
    // "github.com/scrapinghub/shubc/scrapinghub"
    //FIXME: return to the above one ^^^
    "shubc/scrapinghub"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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

func print_out(flags *PFlags, format string, args ...interface{}) {
	output := flags.Output
	line := fmt.Sprintf(format, args...)
	var out *os.File = os.Stdout
	var err error
	if output == "" {
		fmt.Println(line)
		return
	}
	out, err = os.OpenFile(output, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Error writing output to file: %s\n", err)
	} else {
		fmt.Fprintln(out, line)
	}
	defer func() {
		if err := out.Close(); err != nil {
			panic(err)
		}
	}()
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
	Tailing     bool
	Debug       bool
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

	fmt.Println("   Deploy API: ")
	fmt.Println("     deploy <target> [egg] [project_id] [version]  - deploy `target` to Scrapinghub")
	fmt.Println("     deploy-list-targets                           - list available targets to deploy")
	fmt.Println("     build-egg                                     - build egg but not deploy")
}

func cmd_spiders(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		log.Fatalf("Missing argument: <project_id>\n")
	}
	project_id := args[0]
	var spiders scrapinghub.Spiders
	spider_list, err := spiders.List(conn, project_id)

	if err != nil {
		log.Fatalf("spiders error: %s\n", err)
	} else {
		print_out(flags, "| %30s | %10s | %20s |\n", "name", "type", "version")
		print_out(flags, dashes(70))
		for _, spider := range spider_list.Spiders {
			print_out(flags, "| %30s | %10s | %20s |\n", spider["id"], spider["type"], spider["version"])
		}
	}
}

func cmd_jobs(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		log.Fatalf("Missing argument: <project_id>\n")
	}
	project_id := args[0]
	filters := equality_list_to_map(args[1:])

	count := flags.Count
	offset := flags.Offset

	if flags.AsJsonLines {
		ls := scrapinghub.LinesStream{Conn: conn, Count: count, Offset: offset}
		ch_jobs, errch := ls.JobsAsJsonLines(project_id, filters)
		for line := range ch_jobs {
			fmt.Println(line)
		}
		for err := range errch {
			log.Fatalf("jobs error: %s\n", err)
		}
	} else {
		var jobs scrapinghub.Jobs
		jobs_list, err := jobs.List(conn, project_id, count, filters)
		if err != nil {
			log.Fatalf("jobs error: %s", err)
		}
		outfmt := "| %10s | %25s | %12s | %10s | %10s | %10s | %20s |\n"
		print_out(flags, outfmt, "id", "spider", "state", "items", "errors", "log lines", "started_time")
		print_out(flags, dashes(106))
		for _, j := range jobs_list.Jobs {
			print_out(flags, "| %10s | %25s | %12s | %10d | %10d | %10d | %20s |\n", j.Id, j.Spider, j.State,
				j.ItemsScraped, j.ErrorsCount, j.Logs, j.StartedTime)
		}
	}
}

func cmd_jobinfo(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		log.Fatalf("Missing argument: <job_id>\n")
	}
	job_id := args[0]

	var jobs scrapinghub.Jobs
	jobinfo, err := jobs.JobInfo(conn, job_id)

	if err != nil {
		log.Fatalf("jobinfo error: %s\n", err)
	} else {
		outfmt := "| %-30s | %60s |\n"
		print_out(flags, outfmt, "key", "value")
		print_out(flags, dashes(97))
		print_out(flags, outfmt, "id", jobinfo.Id)
		print_out(flags, outfmt, "spider", jobinfo.Spider)
		print_out(flags, outfmt, "spider_args", "")
		for k, v := range jobinfo.SpiderArgs {
			print_out(flags, outfmt, " ", fmt.Sprintf("%s = %s", k, v))
		}
		print_out(flags, outfmt, "spider_type", jobinfo.SpiderType)
		print_out(flags, outfmt, "state", jobinfo.State)
		print_out(flags, outfmt, "close_reason", jobinfo.CloseReason)
		print_out(flags, dashes(97))
		print_out(flags, outfmt, "responses_received", fmt.Sprintf("%d", jobinfo.ResponsesReceived))
		print_out(flags, outfmt, "items_scraped", fmt.Sprintf("%d", jobinfo.ItemsScraped))
		print_out(flags, outfmt, "errors_count", fmt.Sprintf("%d", jobinfo.ErrorsCount))
		print_out(flags, outfmt, "logs", fmt.Sprintf("%d", jobinfo.Logs))
		print_out(flags, dashes(97))
		print_out(flags, outfmt, "started_time", jobinfo.StartedTime)
		print_out(flags, outfmt, "updated_time", jobinfo.UpdatedTime)
		print_out(flags, outfmt, "elapsed", fmt.Sprintf("%d", jobinfo.Elapsed))
		print_out(flags, outfmt, "tags", " ")
		for _, e := range jobinfo.Tags {
			print_out(flags, outfmt, " ", e)
		}
		print_out(flags, outfmt, "priority", fmt.Sprintf("%d", jobinfo.Priority))
		print_out(flags, outfmt, "version", jobinfo.Version)
		print_out(flags, dashes(97))
	}
}

func cmd_schedule(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 2 {
		log.Fatalf("Missing arguments: <project_id> and <spider_name>\n")
	}
	var jobs scrapinghub.Jobs
	project_id := args[0]
	spider_name := args[1]
	spider_args := equality_list_to_map(args[2:])
	job_id, err := jobs.Schedule(conn, project_id, spider_name, spider_args)

	if err != nil {
		log.Fatalf("schedule error: %s\n", err)
	} else {
		fmt.Printf("Scheduled job: %s\n", job_id)
	}
}

func cmd_jobs_stop(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		log.Fatalf("Missing argument: <job_id>\n")
	}
	var jobs scrapinghub.Jobs
	job_id := args[0]
	err := jobs.Stop(conn, job_id)
	if err != nil {
		log.Fatalf("stop error: %s\n", err)
	} else {
		fmt.Printf("Stopped job: %s\n", job_id)
	}
}

func cmd_jobs_update(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		log.Fatalf("Missing argument: <job_id>\n")
	}

	var jobs scrapinghub.Jobs
	job_id := args[0]
	update_data := equality_list_to_map(args[1:])

	err := jobs.Update(conn, job_id, update_data)
	if err != nil {
		log.Fatalf("update error: %s\n", err)
	} else {
		fmt.Printf("Updated job: %s\n", job_id)
	}
}

func cmd_jobs_delete(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		log.Fatalf("Missing argument: <job_id>\n")
	}

	var jobs scrapinghub.Jobs
	job_id := args[0]
	err := jobs.Delete(conn, job_id)

	if err != nil {
		log.Fatalf("delete error: %s\n", err)
	} else {
		fmt.Printf("Deleted job: %s\n", job_id)
	}
}

func cmd_items(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		log.Fatalf("Missing argument: <job_id>\n")
	}

	job_id := args[0]
	count := flags.Count
	offset := flags.Offset
	ls := scrapinghub.LinesStream{Conn: conn, Count: count, Offset: offset}

	if flags.AsJsonLines {
		ch_lines, errch := ls.ItemsAsJsonLines(job_id)

		for line := range ch_lines {
			print_out(flags, line)
		}
		for err := range errch {
			log.Fatalf("items error: %s\n", err)
		}
	} else if flags.AsCSV {
		ch_lines, errch := ls.ItemsAsCSV(job_id, flags.CSVFlags.IncludeHeaders, flags.CSVFlags.Fields)
		for line := range ch_lines {
			print_out(flags, line)
		}
		for err := range errch {
			log.Fatalf("items error: %s\n", err)
		}
	} else {
		items, err := scrapinghub.RetrieveItems(conn, job_id, count, offset)
		if err != nil {
			log.Fatalf("items error: %s\n", err)
		}
		for i, e := range items {
			print_out(flags, "Item %5d %s\n", i, dashes(129))
			for k, v := range e {
				//fmt.Printf("| %-33s | %100s |\n", k, fmt.Sprintf("%v", v))
				print_out(flags, "| %-33s | %100s |\n", k, fmt.Sprintf("%v", v))
			}
			print_out(flags, dashes(140))
		}
	}
}

func cmd_as_project_slybot(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		log.Fatalf("Missing argument: <project_id>\n")
	}

	project_id := args[0]
	spiders := args[1:]
	output := flags.Output

	var out *os.File = os.Stdout
	var err error
	if output != "" {
		out, err = os.Create(output)
		if err != nil {
			log.Fatalf("project slybot error: fail to write to file: %s\n", err)
		}
	}
	defer func() {
		if err := out.Close(); err != nil {
			panic(err)
		}
	}()
	err = scrapinghub.RetrieveSlybotProject(conn, project_id, spiders, out)
	if err != nil {
		log.Fatalf("project-slybot error: %s\n", err)
	}
}

func cmd_log(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		log.Fatalf("Missing argument: <job_id>\n")
	}

	job_id := args[0]
	count := flags.Count
	offset := flags.Offset

	if flags.Tailing {
		log_tailing(conn, job_id)
	} else {
		ls := scrapinghub.LinesStream{Conn: conn, Count: count, Offset: offset}
		ch_lines, ch_err := ls.LogLines(job_id)

		for line := range ch_lines {
			print_out(flags, line)
		}
		for err := range ch_err {
			log.Fatalf("log error: %s\n", err)
		}
	}
}

func log_tailing(conn *scrapinghub.Connection, job_id string) {
	var jobs scrapinghub.Jobs
	jobinfo, err := jobs.JobInfo(conn, job_id)
	if err != nil {
		log.Fatalf("%s\n", err)
	}
	// Number of log lines in the job
	offset := jobinfo.Logs
	if offset > 0 {
		offset -= 1 // start one line before
	}
	count := 10 // ask for this lines in every call
	ls := scrapinghub.LinesStream{Conn: conn, Count: count, Offset: offset}
	for {
		retrieved := 0
		ch_lines, ch_err := ls.LogLines(job_id)
		for line := range ch_lines {
			retrieved++
			fmt.Fprintf(os.Stdout, "%s\n", line)
		}
		for err := range ch_err {
			log.Fatalf("%s\n", err)
		}
		ls.Offset += retrieved
		time.Sleep(time.Second)
	}
}

func cmd_reschedule(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		log.Fatalf("Missing argument: <job_id>\n")
	}
	job_id := args[0]

	var jobs scrapinghub.Jobs
	new_job_id, err := jobs.Reschedule(conn, job_id)
	if err != nil {
		log.Fatalf("reschedule error: %s\n", err)
	} else {
		fmt.Printf("Re-scheduled job new id: %s\n", new_job_id)
	}
}

func cmd_eggs_add(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 2 {
		log.Fatalf("Missing arguments: <project_id> and <egg_path>\n")
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
			log.Fatalf("eggs-add error: Can't guess the name and version from egg path filename, provide it using name=<name> and version=<version> as parameters.\n")
		}
		egg_name = result[1]
		egg_ver = result[2]
	}
	if egg_name == "" || egg_ver == "" {
		log.Fatalf("Error: name and version are required\n")
	}
	var eggs scrapinghub.Eggs
	eggdata, err := eggs.Add(conn, project_id, egg_name, egg_ver, egg_path)
	if err != nil {
		log.Fatalf("eggs-add error: %s\n", err)
	}
	fmt.Printf("Egg uploaded successfully! Project: %s, Egg name: %s, version: %s\n", project_id, eggdata.Name, eggdata.Version)
}

func cmd_eggs_list(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if len(args) < 1 {
		log.Fatalf("Missing argument: <project_id>\n")
	}

	project_id := args[0]
	var eggs scrapinghub.Eggs
	egglist, err := eggs.List(conn, project_id)
	if err != nil {
		log.Fatalf("eggs-list error: %s\n", err)
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
		log.Fatalf("Missing arguments: <project_id> and <egg_name>\n")
	}
	project_id := args[0]
	egg_name := args[1]

	var eggs scrapinghub.Eggs
	err := eggs.Delete(conn, project_id, egg_name)
	if err != nil {
		log.Fatalf("eggs-delete error: %s\n", err)
	}
	fmt.Printf("Egg %s successfully deleted from project: %s\n", egg_name, project_id)
}

//TODO: implement
func cmd_deploy(conn *scrapinghub.Connection, args []string, flags *PFlags) {
}

//TODO: implement
func cmd_deploy_list_targets(conn *scrapinghub.Connection, args []string, flags *PFlags) {
	if !scrapinghub.Inside_scrapy_project() {
		log.Fatal("Error: no Scrapy project found in this location")
	}

	for name, _ := range scrapinghub.Scrapy_cfg_targets() {
		fmt.Println(name)
	}
}

//TODO: implement
func cmd_deploy_build_egg(conn *scrapinghub.Connection, args []string, flags *PFlags) {
}

func main() {
	// Set loggin prefix & flags
	log.SetPrefix("shubc: ")
	log.SetFlags(0)

	var gflags PFlags
	var defaultApiUrl = "https://dash.scrapinghub.com/api"

	apikey := flag.String("apikey", find_apikey(), "Scrapinghub api key")
	apiurl := flag.String("apiurl", defaultApiUrl, "Scrapinghub API URL (can be changed to another uri for testing).")
	count := flag.Int("count", 0, "Count for those commands that need a count limit")
	offset := flag.Int("offset", 0, "Number of results to skip from the beginning")
	output := flag.String("o", "", "Write output to a file instead of Stdout")
	fjl := flag.Bool("jl", false, "If given, for command items and jobs will retrieve all the data writing to os.Stdout as JsonLines format")

	fcsv := flag.Bool("csv", false, "If given, for command items, they will retrieve as CSV writing to os.Stdout")
	fincheads := flag.Bool("include_headers", false, "When -csv given, include the headers of the CSV in the output")
	fcsv_fields := flag.String("fields", "", "When -csv given, list of comma separated fields to include in the CSV")
	tail := flag.Bool("tail", false, "The same that `tail -f` for command `log`")
	debug := flag.Bool("debug", false, "debug mode for some commands (deploy: not remove debug dir)")

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
	gflags.Tailing = *tail
	gflags.Debug = *debug

	commands := map[string]CmdFun{
		"spiders":             cmd_spiders,
		"jobs":                cmd_jobs,
		"jobinfo":             cmd_jobinfo,
		"schedule":            cmd_schedule,
		"stop":                cmd_jobs_stop,
		"update":              cmd_jobs_update,
		"delete":              cmd_jobs_delete,
		"items":               cmd_items,
		"project-slybot":      cmd_as_project_slybot,
		"log":                 cmd_log,
		"reschedule":          cmd_reschedule,
		"eggs-add":            cmd_eggs_add,
		"eggs-list":           cmd_eggs_list,
		"eggs-delete":         cmd_eggs_delete,
		"deploy":              cmd_deploy,
		"deploy-list-targets": cmd_deploy_list_targets,
		"build-egg":           cmd_deploy_build_egg,
	}

	if len(flag.Args()) <= 0 {
		fmt.Fprintf(os.Stderr, "Usage: shubc [options] url\n")
	} else {
		// Create new connection
		var conn scrapinghub.Connection
		err := conn.New(*apikey)
		if err != nil {
			log.Fatalf("error creating scrapinghub.Connection: %s", err)
		}
		err = conn.SetAPIUrl(*apiurl)
		if err != nil {
			log.Fatalf("error setting api url: %s", err)
		}

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
				log.Fatalf("'%s' command not found\n", cmd)
			}
		}
	}
}

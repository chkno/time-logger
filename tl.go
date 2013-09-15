package main

import (
	"bufio"
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var initial_days = flag.Int(
	"initial_days", 14,
	"How many days to display initially")

var listen_address = flag.String(
	"listen_address", "",
	"Local address to listen on.  Typically \"\" (permissive) or \"localhost\" (restrictive)")

var log_filename = flag.String(
	"log_file", "tl.log",
	"Where to keep the log")

var port = flag.Int(
	"port", 29804,
	"Port to listen on")

var template_path = flag.String(
	"template_path", ".",
	"Where to find the HTML template file")

type Event struct {
	Name             string
	Time             time.Time
	OriginalDuration time.Duration // Duration before day splitting
	Duration         time.Duration // Intra-day (display) duration
	TotalDuration    time.Duration // Sum of all similarly-named Events
}

type Day struct {
	Events []Event
}

type Report struct {
	Days []Day
}

func read_data_file(in io.Reader) (events []Event, err error) {
	lines := bufio.NewScanner(in)
	line_number := 0
	for lines.Scan() {
		line_number++
		fields := strings.SplitN(lines.Text(), " ", 7)
		var numerically [6]int
		for i := 0; i < 6; i++ {
			var err error
			numerically[i], err = strconv.Atoi(fields[i])
			if err != nil {
				return nil, errors.New(fmt.Sprint("Field ", i, " on line ", line_number, " is not numeric: ", err))
			}
		}
		events = append(events, Event{
			Name: fields[6],
			Time: time.Date(
				numerically[0],
				time.Month(numerically[1]),
				numerically[2],
				numerically[3],
				numerically[4],
				numerically[5],
				0, // Nanoseconds
				time.Local),
		})
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}
	return
}

func calculate_durations(events []Event) {
	// The duration of an event is the difference between that event's
	// timestamp and the following event's timestamp.  I.e., Event.Time
	// is the beginning of the event.
	for i := range events[:len(events)-1] {
		d := events[i+1].Time.Sub(events[i].Time)
		events[i].OriginalDuration = d
		events[i].Duration = d
	}
	d := time.Now().Sub(events[len(events)-1].Time)
	events[len(events)-1].OriginalDuration = d
	events[len(events)-1].Duration = d
}

func calculate_total_durations(events []Event) {
	totals := make(map[string]time.Duration)
	for _, e := range events {
		totals[e.Name] += e.Duration
	}
	for i, e := range events {
		events[i].TotalDuration = totals[e.Name]
	}
}

func start_of_day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func start_of_next_day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, time.Local)
}

func split_by_day(events []Event) (by_day []Day) {
	var current_day time.Time
	var this_day Day
	for _, e := range events {
		for {
			first_day_of_e := e
			day := start_of_day(e.Time)
			if start_of_day(e.Time.Add(e.Duration)) != day {
				split_at := start_of_next_day(e.Time)
				first_day_of_e.Duration = split_at.Sub(e.Time)
				e.Time = split_at
				e.Duration -= first_day_of_e.Duration

			}
			if current_day != day {
				if !current_day.IsZero() {
					by_day = append(by_day, this_day)
					this_day = Day{}
				}
				current_day = day
			}
			this_day.Events = append(this_day.Events, first_day_of_e)
			if start_of_day(first_day_of_e.Time) == start_of_day(e.Time) {
				break
			}
		}
	}
	by_day = append(by_day, this_day)
	return
}

func (e *Event) TimeOfDay() int {
	return e.Time.Hour()*3600 + e.Time.Minute()*60 + e.Time.Second()
}

func (e *Event) Color() template.CSS {
	if e.Name == "" {
		return template.CSS("white")
	}
	hash := sha1.New()
	io.WriteString(hash, e.Name)
	hue := 360.0 * int(hash.Sum(nil)[0]) / 256.0
	return template.CSS("hsl(" + strconv.Itoa(hue) + ",90%,45%)")
}

func DescribeDuration(t time.Duration) string {
	if t.Hours() > 24 {
		return fmt.Sprintf("%.1f days", t.Hours()/24)
	}
	if t.Hours() > 1 {
		return fmt.Sprintf("%.1f hours", t.Hours())
	}
	if t.Minutes() > 1 {
		return fmt.Sprintf("%.1f min", t.Minutes())
	}
	return fmt.Sprintf("%.0f sec", t.Seconds())
}

func (e *Event) DurationDescription() string {
	if e.OriginalDuration == e.TotalDuration {
		return DescribeDuration(e.OriginalDuration)
	}
	return (DescribeDuration(e.OriginalDuration) +
		" of " + DescribeDuration(e.TotalDuration))

}

func (e *Event) Height() float32 {
	return 100 * float32(e.Duration.Seconds()) / 86400
}

func (r Report) BodyWidth() float32 {
	days_on_screen := *initial_days
	if len(r.Days) < days_on_screen {
		days_on_screen = len(r.Days)
	}
	return 100 * float32(len(r.Days)) / float32(days_on_screen)
}

func (r Report) DayWidth() float32 {
	return 100.0 / float32(len(r.Days))
}

func generate_report(days []Day) (td Report) {
	td.Days = days
	return
}

func backfill_first_day(d *Day) {
	// Stuff an empty event at the beginning of the first day
	first_event_time := d.Events[0].Time
	start_of_first_day := start_of_day(first_event_time)
	time_until_first_event := first_event_time.Sub(start_of_first_day)
	first_day_events := append([]Event{Event{Duration: time_until_first_event}}, d.Events...)
	d.Events = first_day_events
}

func execute_template(template_name string, data interface{}, out io.Writer) error {
	t := template.New("tl")
	t, err := t.ParseFiles(filepath.Join(*template_path, template_name))
	if err != nil {
		return err
	}
	err = t.ExecuteTemplate(out, template_name, data)
	if err != nil {
		return err
	}
	return nil
}

func view_handler(w http.ResponseWriter, r *http.Request) {
	log_file, err := os.Open(*log_filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer log_file.Close()
	all_events, err := read_data_file(log_file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	calculate_durations(all_events)
	calculate_total_durations(all_events)
	by_day := split_by_day(all_events)
	backfill_first_day(&by_day[0])
	report := generate_report(by_day)
	err = execute_template("view.template", report, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func log_handler(w http.ResponseWriter, r *http.Request) {
	err := execute_template("log.template", nil, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func write_to_log(line []byte) error {
	log_file, err := os.OpenFile(*log_filename, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return errors.New(fmt.Sprint("Couldn't open log file: ", err))
	}
	defer log_file.Close() // Closed with error checking below
	written, err := log_file.Write(line)
	if err != nil {
		if written == 0 {
			return errors.New(fmt.Sprint("Couldn't write to log file: ", err))
		} else {
			return errors.New(fmt.Sprint("Only wrote ", written, " bytes to log file: ", err))
		}
	}
	err = log_file.Close()
	if err != nil {
		return errors.New(fmt.Sprint("Couldn't close log file: ", err))
	}
	return nil
}

func log_submit_handler(w http.ResponseWriter, r *http.Request) {
	w.Header()["Allow"] = []string{"POST"}
	if r.Method != "POST" {
		http.Error(w, "Please use POST", http.StatusMethodNotAllowed)
		return
	}
	t := time.Now().Format("2006 01 02 15 04 05 ")
	thing := strings.Replace(r.FormValue("thing"), "\n", "", -1)
	err := write_to_log([]byte(t + thing + "\n"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = execute_template("log_submit.template", nil, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	flag.Parse()
	http.HandleFunc("/view", view_handler)
	http.HandleFunc("/log", log_handler)
	http.HandleFunc("/log_submit", log_submit_handler)
	err := http.ListenAndServe(*listen_address+":"+strconv.Itoa(*port), nil)
	if err != nil {
		log.Fatal("http.ListenAndServe: ", err)
	}
}

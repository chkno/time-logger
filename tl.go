package main

import "bufio"
import "crypto/sha1"
import "fmt"
import "html/template"
import "io"
import "os"
import "strconv"
import "strings"
import "time"

const daylength = 86400 // TODO: daylight savings time

type Day string

type Event struct {
	Name             string
	Time             time.Time
	OriginalDuration time.Duration
	Duration         time.Duration
}

type TemplateDay struct {
	Events []Event
}

type TemplateData struct {
	DayWidth float32
	Days     []TemplateDay
}

func read_data_file(in io.Reader) (events []Event) {
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
				panic(fmt.Sprint("Field ", i, " on line ", line_number, " is not numeric"))
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
		panic(err)
	}
	return
}

func calculate_durations(events []Event) {
	for i := range events[:len(events)-1] {
		d := events[i+1].Time.Sub(events[i].Time)
		events[i].OriginalDuration = d
		events[i].Duration = d
	}
	d := time.Now().Sub(events[len(events)-1].Time)
	events[len(events)-1].OriginalDuration = d
	events[len(events)-1].Duration = d
}

func split_by_day(events []Event) (days []Day, by_day map[Day]([]Event)) {
	by_day = make(map[Day]([]Event))
	for _, e := range events {
		for {
			first_day_of_e := e
			if TimeDay(e.Time) != TimeDay(e.Time.Add(e.Duration)) {
				split_at := time.Date(
					e.Time.Year(),
					e.Time.Month(),
					e.Time.Day()+1,
					0, 0, 0, 0, time.Local)
				first_day_of_e.Duration = split_at.Sub(e.Time)
				e.Time = split_at
				e.Duration -= first_day_of_e.Duration

			}
			day := TimeDay(first_day_of_e.Time)
			if len(days) == 0 || days[len(days)-1] != day {
				days = append(days, day)
			}
			by_day[day] = append(by_day[day], first_day_of_e)
			if TimeDay(first_day_of_e.Time) == TimeDay(e.Time) {
				break
			}
		}
	}
	return
}

func TimeDay(t time.Time) Day {
	return Day(fmt.Sprint(t.Year(), t.Month(), t.Day()))
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

func (e *Event) DurationDescription() string {
	if e.OriginalDuration.Hours() > 24 {
		return fmt.Sprintf("%.1f days", e.OriginalDuration.Hours()/24)
	}
	if e.OriginalDuration.Hours() > 1 {
		return fmt.Sprintf("%.1f hours", e.OriginalDuration.Hours())
	}
	if e.OriginalDuration.Minutes() > 1 {
		return fmt.Sprintf("%.1f min", e.OriginalDuration.Minutes())
	}
	return fmt.Sprintf("%.0f sec", e.OriginalDuration.Seconds())
}

func (e *Event) Height() float32 {
	return 100 * float32(e.Duration.Seconds()) / daylength
}

func generate_report(days []Day, events map[Day]([]Event)) (td TemplateData) {
	td.DayWidth = 100.0 / float32(len(days))
	for i, day := range days {
		var tday TemplateDay
		if i == 0 {
			// Stuff an empty event at the beginning
			first_event_time := events[day][0].Time
			start_of_first_day := time.Date(
				first_event_time.Year(),
				first_event_time.Month(),
				first_event_time.Day(),
				0, 0, 0, 0, time.Local)
			time_until_first_event := first_event_time.Sub(start_of_first_day)
			tday.Events = append(tday.Events, Event{Duration: time_until_first_event})
		}
		for _, event := range events[day] {
			tday.Events = append(tday.Events, event)
		}
		td.Days = append(td.Days, tday)
	}
	return
}

func main() {
	all_events := read_data_file(os.Stdin)
	calculate_durations(all_events)
	days, events := split_by_day(all_events)
	t := template.New("tl")
	t, err := t.ParseFiles("tl.template")
	if err != nil {
		panic(err)
	}
	err = t.ExecuteTemplate(os.Stdout, "tl.template", generate_report(days, events))
	if err != nil {
		panic(err)
	}
}

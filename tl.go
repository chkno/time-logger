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
	Duration         time.Duration
	Day              Day
	TimeOfDay        int // Seconds after midnight
	IntraDayDuration int // Size in seconds of this chunk of the original event after day splitting
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
		day := Day(fields[0] + fields[1] + fields[2])
		events = append(events, Event{
			Day:       day,
			Name:      fields[6],
			TimeOfDay: numerically[3]*3600 + numerically[4]*60 + numerically[5],
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
		events[i].Duration = events[i+1].Time.Sub(events[i].Time)
	}
	events[len(events)-1].Duration = time.Now().Sub(events[len(events)-1].Time)
}

func split_by_day(events []Event) (days []Day, by_day map[Day]([]Event)) {
	by_day = make(map[Day]([]Event))
	for _, e := range events {
		if len(days) == 0 || days[len(days)-1] != e.Day {
			days = append(days, e.Day)
		}
		by_day[e.Day] = append(by_day[e.Day], e)
	}
	return
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
	// TODO: Use actual Duration, not IntraDayDuration
	if e.IntraDayDuration > 3600 {
		return fmt.Sprintf("%.1f hours", float32(e.IntraDayDuration)/3600)
	}
	if e.IntraDayDuration > 60 {
		return fmt.Sprintf("%.1f min", float32(e.IntraDayDuration)/60)
	}
	return fmt.Sprintf("%d sec", e.IntraDayDuration)
}

func (e *Event) Height() float32 {
	return 100 * float32(e.IntraDayDuration) / daylength
}

func print_event(duration int, name string) (e Event) {
	e.IntraDayDuration = duration
	e.Name = name
	return
}

func generate_report(days []Day, events map[Day]([]Event)) (td TemplateData) {
	now_time := time.Now()
	now := now_time.Hour()*3600 + now_time.Minute()*60 + now_time.Second()
	today := Day(fmt.Sprintf("%04d%02d%02d", now_time.Year(), now_time.Month(), now_time.Day()))
	td.DayWidth = 100.0 / float32(len(days))
	prevname := ""
	for _, day := range days {
		var tday TemplateDay
		prevtime := 0
		for _, event := range events[day] {
			tday.Events = append(tday.Events, print_event(event.TimeOfDay-prevtime, prevname))
			prevtime = event.TimeOfDay
			prevname = event.Name
		}
		final_event_time := daylength
		if day == today {
			final_event_time = now
		}
		tday.Events = append(tday.Events, print_event(final_event_time-prevtime, prevname))
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

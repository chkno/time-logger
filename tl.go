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
	Name           string
	Time, Duration int
	Height         float32
}

type TemplateDay struct {
	Events []Event
}

type TemplateData struct {
	DayWidth float32
	Days     []TemplateDay
}

func read_data_file(in io.Reader) (days []Day, events map[Day]([]Event)) {
	events = make(map[Day]([]Event))
	lines := bufio.NewScanner(in)
	for lines.Scan() {
		fields := strings.SplitN(lines.Text(), " ", 7)
		day := Day(fields[0] + fields[1] + fields[2])
		hour, hour_err := strconv.Atoi(fields[3])
		minute, minute_err := strconv.Atoi(fields[4])
		second, second_err := strconv.Atoi(fields[5])
		if hour_err != nil || minute_err != nil || second_err != nil {
			panic("Malformed line")
		}
		time := hour*3600 + minute*60 + second

		if len(days) == 0 || days[len(days)-1] != day {
			days = append(days, day)
		}
		events[day] = append(events[day], Event{Name: fields[6], Time: time})
	}
	if err := lines.Err(); err != nil {
		panic(err)
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
	if e.Duration > 3600 {
		return fmt.Sprintf("%.1f hours", float32(e.Duration)/3600)
	}
	if e.Duration > 60 {
		return fmt.Sprintf("%.1f min", float32(e.Duration)/60)
	}
	return fmt.Sprintf("%d sec", e.Duration)
}

func print_event(duration int, name string) (e Event) {
	e.Duration = duration
	e.Name = name
	e.Height = 100 * float32(duration) / daylength
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
			tday.Events = append(tday.Events, print_event(event.Time-prevtime, prevname))
			prevtime = event.Time
			prevname = event.Name
		}
		if day == today {
			tday.Events = append(tday.Events, print_event(now-prevtime, prevname))
		} else {
			tday.Events = append(tday.Events, print_event(daylength-prevtime, prevname))
		}
		td.Days = append(td.Days, tday)
	}
	return
}

func main() {
	days, events := read_data_file(os.Stdin)
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

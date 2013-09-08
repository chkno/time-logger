package main

import "bufio"
import "crypto/sha1"
import "fmt"
import "io"
import "os"
import "strconv"
import "strings"
import "time"

const daylength = 86400 // TODO: daylight savings time

type Day string

type Event struct {
	Name string
	Time int
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
		events[day] = append(events[day], Event{fields[6], time})
	}
	if err := lines.Err(); err != nil {
		panic(err)
	}
	return
}

func hash_color(title string) string {
	if title == "" {
		return "white"
	}
	hash := sha1.New()
	io.WriteString(hash, title)
	hue := 360.0 * int(hash.Sum(nil)[0]) / 256.0
	return "hsl(" + strconv.Itoa(hue) + ",90%,45%)"
}

func summarize_time(duration int) string {
	if duration > 3600 {
		return fmt.Sprintf("%.1f hours", float32(duration)/3600)
	}
	if duration > 60 {
		return fmt.Sprintf("%.1f min", float32(duration)/60)
	}
	return fmt.Sprintf("%d sec", duration)
}

func print_event(duration int, title string) string {
	height := 100 * float32(duration) / daylength
	color := hash_color(title)
	return fmt.Sprint("<div class='event_outer' style='background-color: ", color,
		"; height: ", height, "%'><div class='event_inner'><span title='", title,
		" (", summarize_time(duration), ")'>", title, "</span></div></div>")
}

func generate_report(days []Day, events map[Day]([]Event)) string {
	now_time := time.Now()
	now := now_time.Hour()*3600 + now_time.Minute()*60 + now_time.Second()
	today := Day(fmt.Sprintf("%04d%02d%02d", now_time.Year(), now_time.Month(), now_time.Day()))
	output := fmt.Sprint(`<html><head><style type='text/css'>
      body { margin: 0; padding: 0; height: 100% }
      html, day { margin: 0; padding: 0; height: 100% }
      .day { width: `, 100.0/float32(len(days)), `%; float: left }
      .event_outer { overflow: hidden; margin: 0 }
      .event_inner { height: 100%; padding: 2px; border: thin solid black;
                     font-family: Verdana, Geneva, sans-serif; 
                     color: white; font-weight: bold }
    </style>
    <script type="text/javascript">
      document.onkeypress = function(e) {
        e = e || window.event;
        var dir; var scale;
        switch(e.keyCode) {
          case 122: document.body.style.height = document.body.style.width = "100%"; return;
          case 107: case 119:  dir = "height"; factor = 9/10; break;
          case 106: case 115:  dir = "height"; factor = 10/9; break;
          case 104: case  97:  dir = "width"; factor = 9/10; break;
          case 108: case 100:  dir = "width"; factor = 10/9; break;
          default: return;
        }
        document.body.style[dir] = (parseInt(document.body.style[dir] || "100%") * factor) + "%";
      }
    </script></head><body>`)

	prevname := ""
	for _, day := range days {
		output += "<div class='day'>"
		prevtime := 0
		for _, event := range events[day] {
			output += print_event(event.Time-prevtime, prevname)
			prevtime = event.Time
			prevname = event.Name
		}
		if day == today {
			output += print_event(now-prevtime, prevname)
		} else {
			output += print_event(daylength-prevtime, prevname)
		}
		output += "</div>"
	}
	output += "</body></html>"
	return output
}

func main() {
	days, events := read_data_file(os.Stdin)
	fmt.Print(generate_report(days, events))
}

package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/akamensky/argparse"
	ics "github.com/arran4/golang-ical"
)

type CliArgs struct {
	icsPath            string
	outPath            string
	eventSummaryFilter string
	name               string
	comma              string
	isStdout           bool
}

func parseArgs() CliArgs {
	parser := argparse.NewParser("ics2csv", "Converts an ics file to a csv file")
	icsPath := parser.String("i", "ics", &argparse.Options{Required: true, Help: "Path to the ics file"})
	outPath := parser.String("c", "csv", &argparse.Options{Required: false, Help: "Path to the output csv file", Default: "out.csv"})
	eventSummaryFilter := parser.String("f", "filter", &argparse.Options{Required: false, Help: "Filter events by summary"})
	name := parser.String("n", "name", &argparse.Options{Required: false, Help: "Your name", Default: "yourName"})
	comma := parser.String("d", "delimiter", &argparse.Options{Required: false, Help: "Delimiter for csv", Default: "\t"})
	isStdout := parser.Flag("s", "stdout", &argparse.Options{Required: false, Help: "Write to stdout instead of a file"})
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}
	return CliArgs{
		*icsPath,
		*outPath,
		*eventSummaryFilter,
		*name,
		*comma,
		*isStdout,
	}
}

func parseTime(timeString string) (time.Time, error) {
	t, err := time.Parse("20060102T150405", timeString)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) - hours*60
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}

type Column struct {
	name      string
	date      string
	startTime string
	endTime   string
	duration  string
}

func NewColumn(event *ics.VEvent, name string) (Column, error) {
	start, err := parseTime(event.GetProperty("DTSTART").Value)
	if err != nil {
		return Column{}, err
	}

	end, err := parseTime(event.GetProperty("DTEND").Value)
	if err != nil {
		return Column{}, err
	}

	return Column{
		name:      name,
		date:      start.Format("2006/01/02"),
		startTime: start.Format("15:04"),
		endTime:   end.Format("15:04"),
		duration:  formatDuration(end.Sub(start)),
	}, nil
}

func writeCsv(w csv.Writer, cols []Column) error {
	for _, col := range cols {
		err := w.Write([]string{
			col.name,
			col.date,
			col.startTime,
			col.endTime,
			col.duration,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	args := parseArgs()

	// read ics
	icsFile, err := os.Open(args.icsPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = icsFile.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	cal, err := ics.ParseCalendar(icsFile)
	if err != nil {
		log.Fatal(err)
	}

	// filter events
	events := make([]*ics.VEvent, 0, len(cal.Events()))

	for _, event := range cal.Events() {
		eventSummary := event.GetProperty("SUMMARY").Value
		if args.eventSummaryFilter != "" && eventSummary != args.eventSummaryFilter {
			continue
		}
		events = append(events, event)
	}

	// create columns
	columns := make([]Column, len(events))
	for i, event := range events {
		column, err := NewColumn(event, args.name)
		if err != nil {
			log.Fatal(err)
		}
		columns[i] = column
	}
	// sort columns by date and start time
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].date < columns[j].date || (columns[i].date == columns[j].date && columns[i].startTime < columns[j].startTime)
	})

	// write csv
	var writer io.Writer
	if args.isStdout {
		writer = os.Stdout
	} else {
		f, err := os.Create(args.outPath)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			if err = f.Close(); err != nil {
				log.Fatal(err)
			}
		}()
		writer = f
	}

	w := csv.NewWriter(writer)
	c, _ := utf8.DecodeRuneInString(args.comma)
	w.Comma = c
	defer w.Flush()

	err = writeCsv(*w, columns)
	if err != nil {
		log.Fatal(err)
	}
}

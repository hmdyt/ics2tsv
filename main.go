package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/akamensky/argparse"
	ics "github.com/arran4/golang-ical"
)

type CliArgs struct {
	icsPath            string
	outPath            string
	eventSummaryFilter string
	name               string
}

func parseArgs() CliArgs {
	parser := argparse.NewParser("ics2csv", "Converts an ics file to a csv file")
	icsPath := parser.String("i", "ics", &argparse.Options{Required: true, Help: "Path to the ics file"})
	outPath := parser.String("c", "csv", &argparse.Options{Required: true, Help: "Path to the output csv file"})
	eventSummaryFilter := parser.String("f", "filter", &argparse.Options{Required: false, Help: "Filter events by summary"})
	name := parser.String("n", "name", &argparse.Options{Required: false, Help: "Your name", Default: "yourName"})
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}
	return CliArgs{*icsPath, *outPath, *eventSummaryFilter, *name}
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

func writeCsv(w csv.Writer, events []*ics.VEvent, name string) error {
	for _, event := range events {
		start, err := parseTime(event.GetProperty("DTSTART").Value)
		if err != nil {
			return err
		}

		end, err := parseTime(event.GetProperty("DTEND").Value)
		if err != nil {
			return err
		}

		err = w.Write([]string{
			name,
			start.Format("2006/01/02"),
			start.Format("15:04"),
			end.Format("15:04"),
			formatDuration(end.Sub(start)),
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

	// write csv
	outFile, err := os.Create(args.outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = outFile.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	w := csv.NewWriter(outFile)
	defer w.Flush()

	err = writeCsv(*w, events, args.name)
	if err != nil {
		log.Fatal(err)
	}
}

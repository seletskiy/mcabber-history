package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/docopt/docopt-go"
	"github.com/reconquest/ser-go"
)

var version = "1.0"

var usage = `mcabber-history - search by mcabber history.

Tool for searching mcabber history using history file parsing and filtering.

Usage:
  mcabber-history -h | --help
  mcabber-history [options] -S <channel> <filter>...

Options:
  -h --help      Show this help.
  -S             Search specified channel by specified filter.
  --path <path>  Path to history files directory.
                  [default: $HOME/.mcabber/history]
`

type Header struct {
	Type    string
	Time    time.Time
	Length  int
	Message string
}

func main() {
	args, err := docopt.Parse(
		os.ExpandEnv(usage),
		nil,
		true,
		"mcabber-history "+version,
		false,
	)
	if err != nil {
		panic(err)
	}

	switch {
	case args["-S"].(bool):
		err = search(args)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func search(args map[string]interface{}) error {
	files, err := filepath.Glob(
		args["--path"].(string) + "/" +
			args["<channel>"].(string) + "*",
	)
	if err != nil {
		return ser.Errorf(
			err,
			"can't obtain files list for %q",
			args["<channel>"].(string),
		)
	}

	if len(files) == 0 {
		return ser.Errorf(
			err,
			"no history files found in %q (%q)",
			args["--path"].(string),
			args["<channel>"].(string),
		)
	}

	expression := `(?si)` + strings.Join(args["<filter>"].([]string), `.*`)
	filter, err := regexp.Compile(expression)
	if err != nil {
		return ser.Errorf(
			err,
			"can't compile regexp %q",
			expression,
		)
	}

	separator := false

	for _, file := range files {
		handle, err := os.Open(file)
		if err != nil {
			return ser.Errorf(
				err,
				"can't open history file %q",
				file,
			)
		}

		scanner := bufio.NewScanner(handle)
		for scanner.Scan() {
			header, err := parseHeader(scanner.Text())
			if err != nil {
				return ser.Errorf(
					err,
					"line malformed: %q (file %q)",
					scanner.Text(),
					file,
				)
			}

			var (
				lines = []string{
					fmt.Sprintf("%s %s",
						color.BlueString(header.Time.Format(time.ANSIC)),
						header.Message,
					),
				}
			)

			for i := 0; i < header.Length; i++ {
				if !scanner.Scan() {
					return ser.Errorf(
						err,
						"not enough lines in message (%d)",
						header.Length,
					)
				}

				lines = append(lines, scanner.Text())
			}

			message := strings.Join(lines, "\n")

			if filter.MatchString(message) {
				if separator {
					fmt.Println()
				}

				fmt.Println(message)

				separator = true
			}
		}
	}

	return nil
}

func parseHeader(line string) (*Header, error) {
	fields := strings.SplitN(line, ` `, 4)
	if len(fields) < 4 {
		return nil, fmt.Errorf("at least 4 fields should present")
	}

	length, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("can't parse length %q", fields[2])
	}

	timedate, err := time.Parse("20060102T15:04:05Z", fields[1])
	if err != nil {
		return nil, fmt.Errorf("can't parse datetime %q", fields[1])
	}

	return &Header{
		Time:    timedate,
		Length:  int(length),
		Message: fields[3],
	}, nil
}

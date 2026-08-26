// SPDX-License-Identifier: MIT

// Package cli implements goname's command-line interface.
package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/blater/goname"
)

const help = `Generate human-readable random names

Usage: goname [-w|--words INT] [-l|--letters INT]
                [-s|--separator STR] [-d|--dir STR]
                [-c|--complexity INT] [-u|--ubuntu]

  -w, --words INT       number of words; default: 2
  -l, --letters INT     maximum letters in each word; default: unlimited
  -s, --separator STR   separator between words; default: -
  -d, --dir DIR         custom word-list directory
  -c, --complexity INT  0=small, 1=medium, 2=large
  -u, --ubuntu          generate an alliterative name
      --adverb          generate one adverb
      --adjective       generate one adjective
      --name            generate one animal name
  -h, --help            show this help
`

// Run runs the CLI and returns its process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	for _, argument := range args {
		if argument == "-h" || argument == "--help" {
			_, _ = io.WriteString(stdout, help)
			return 0
		}
	}

	options, err := parse(args)
	if err == nil {
		var generated string
		generated, err = goname.NewGenerator().Generate(options)
		if err == nil {
			_, _ = fmt.Fprintln(stdout, generated)
			return 0
		}
	}
	_, _ = fmt.Fprintln(stderr, "ERROR:", err)
	return 1
}

func parse(args []string) (goname.Options, error) {
	options := goname.DefaultOptions()
	var name, adjective, adverb bool

	for index := 0; index < len(args); index++ {
		option := args[index]
		switch option {
		case "-w", "--words":
			value, err := optionValue(args, &index, option)
			if err != nil {
				return options, err
			}
			options.Words, err = parsePositive(value, "words")
			if err != nil {
				return options, err
			}
		case "-l", "--letters":
			value, err := optionValue(args, &index, option)
			if err != nil {
				return options, err
			}
			options.MaxLetters, err = parseNonNegative(value, "letters")
			if err != nil {
				return options, err
			}
			options.MaxLetters = max(3, options.MaxLetters)
		case "-s", "--separator":
			value, err := optionValue(args, &index, option)
			if err != nil {
				return options, err
			}
			options.Separator = value
		case "-d", "--dir":
			value, err := optionValue(args, &index, option)
			if err != nil {
				return options, err
			}
			options.WordDirectory = value
		case "-c", "--complexity":
			value, err := optionValue(args, &index, option)
			if err != nil {
				return options, err
			}
			options.Complexity, err = parseComplexity(value)
			if err != nil {
				return options, err
			}
		case "-u", "--ubuntu":
			options.Alliterate = true
		case "--adverb":
			adverb = true
		case "--adjective":
			adjective = true
		case "--name":
			name = true
		default:
			return options, fmt.Errorf("Unknown options [%s]", option)
		}
	}

	switch {
	case name:
		options.Type = goname.TypeName
	case adjective:
		options.Type = goname.TypeAdjective
	case adverb:
		options.Type = goname.TypeAdverb
	}
	return options, nil
}

func optionValue(args []string, index *int, option string) (string, error) {
	*index++
	if *index >= len(args) {
		return "", fmt.Errorf("missing value for %s", option)
	}
	return args[*index], nil
}

func parsePositive(value, label string) (int, error) {
	parsed, err := parseNonNegative(value, label)
	if err != nil {
		return 0, err
	}
	if parsed == 0 {
		return 0, fmt.Errorf("%s must be a positive integer, got: %s", label, value)
	}
	return parsed, nil
}

func parseNonNegative(value, label string) (int, error) {
	if value == "" || strings.Trim(value, "0123456789") != "" {
		return 0, fmt.Errorf("%s must be a positive integer, got: %s", label, value)
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s is too large, got: %s", label, value)
	}
	return int(parsed), nil
}

func parseComplexity(value string) (goname.Complexity, error) {
	switch value {
	case "0":
		return goname.ComplexitySmall, nil
	case "1":
		return goname.ComplexityMedium, nil
	case "2":
		return goname.ComplexityLarge, nil
	default:
		return goname.ComplexityDefault, fmt.Errorf(
			"complexity must be 0 (small), 1 (medium), or 2 (large), got: %s", value)
	}
}

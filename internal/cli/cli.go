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
                [-s|--separator STR] [-p|--prefix STR] [-d|--dir STR]
                [-c|--complexity INT] [-t|--strategy STR] [-u|--ubuntu]
                [-m|--mixedcase]

  -w, --words INT       number of words; default: 2 (1 for token strategies)
  -l, --letters INT     word limit; token width for hex/base32, ignored by ulid
  -s, --separator STR   separator between words; default: -
  -p, --prefix STR      prefix before the generated name
  -d, --dir DIR         custom word-list directory
  -c, --complexity INT  0=small, 1=medium, 2=large
  -t, --strategy STR    generation strategy: tolkien, hex, base32, ulid
  -u, --ubuntu          generate an alliterative name
  -m, --mixedcase       preserve dictionary case; uppercase generated tokens
      --adverb          generate one adverb
      --adjective       generate one adjective
      --name            generate one name word
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
		case "-w", "--words", "-l", "--letters", "-s", "--separator",
			"-p", "--prefix", "-d", "--dir", "-c", "--complexity", "-t", "--strategy":
			value, err := optionValue(args, &index, option)
			if err != nil {
				return options, err
			}
			if err := setValue(&options, option, value); err != nil {
				return options, err
			}
		case "-u", "--ubuntu":
			options.Alliterate = true
		case "-m", "--mixedcase":
			options.MixedCase = true
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

// setValue converts a value-bearing option after the parser has consumed it.
func setValue(options *goname.Options, option, value string) (err error) {
	switch option {
	case "-w", "--words":
		options.Words, err = parsePositive(value, "words")
	case "-l", "--letters":
		options.MaxLetters, err = parseNonNegative(value, "letters")
		if err == nil {
			options.MaxLetters = max(3, options.MaxLetters)
		}
	case "-s", "--separator":
		options.Separator = value
	case "-p", "--prefix":
		options.Prefix = value
	case "-d", "--dir":
		options.WordDirectory = value
	case "-c", "--complexity":
		options.Complexity, err = parseComplexity(value)
	case "-t", "--strategy":
		options.Strategy, err = parseStrategy(value)
	}
	return err
}

func parseStrategy(value string) (goname.Strategy, error) {
	switch value {
	case "tolkien":
		return goname.StrategyTolkien, nil
	case "hex":
		return goname.StrategyHex, nil
	case "base32":
		return goname.StrategyBase32, nil
	case "ulid":
		return goname.StrategyULID, nil
	}
	return goname.StrategyDefault, fmt.Errorf("strategy must be tolkien, hex, base32, or ulid, got: %s", value)
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

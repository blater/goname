// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blater/goname"
)

func TestSupportsLongOptions(t *testing.T) {
	directory := writeWords(t, "swiftly", "calm", "otter")
	result := run("--words", "3", "--separator", ":", "--dir", directory)
	if result.exitCode != 0 || result.stdout != "swiftly:calm:otter\n" || result.stderr != "" {
		t.Fatalf("run() = %+v", result)
	}
}

func TestSupportsShortOptionsAndEmptySeparator(t *testing.T) {
	directory := writeWords(t, "swiftly", "calm", "otter")
	result := run("-w", "2", "-s", "", "-d", directory)
	if result.exitCode != 0 || result.stdout != "calmotter\n" {
		t.Fatalf("run() = %+v", result)
	}
}

func TestSupportsPrefixWithDictionaryAndTokenStrategies(t *testing.T) {
	directory := writeWords(t, "swiftly", "frosty", "aragorn")
	result := run("-p", "ticket", "-s", "_", "-d", directory)
	if result.exitCode != 0 || result.stdout != "ticket_frosty_aragorn\n" || result.stderr != "" {
		t.Fatalf("dictionary prefix run() = %+v", result)
	}

	result = run("--prefix", "ticket", "--separator", "_", "--strategy", "tolkien")
	parts := strings.Split(strings.TrimSpace(result.stdout), "_")
	if result.exitCode != 0 || result.stderr != "" || len(parts) != 3 || parts[0] != "ticket" {
		t.Fatalf("Tolkien prefix run() = %+v, want prefix plus two words", result)
	}

	result = run("-p", "ticket", "--strategy", "hex", "--words", "2")
	parts = strings.Split(strings.TrimSpace(result.stdout), "-")
	if result.exitCode != 0 || result.stderr != "" || len(parts) != 3 || parts[0] != "ticket" {
		t.Fatalf("token prefix run() = %+v, want prefix plus two tokens", result)
	}
}

func TestSupportsSingleWordOptionsWithUpstreamPrecedence(t *testing.T) {
	directory := writeWords(t, "swiftly", "calm", "otter")
	result := run("--adverb", "--adjective", "--name", "-d", directory)
	if result.exitCode != 0 || result.stdout != "otter\n" {
		t.Fatalf("run() = %+v", result)
	}
}

func TestSupportsLengthComplexityAndUbuntuOptions(t *testing.T) {
	directory := t.TempDir()
	writeWordsAt(t, filepath.Join(directory, "small"),
		"ably\nboldly", "agile\nbrisk", "alpaca\nbadger")
	result := run("-d", directory, "-c", "0", "-l", "6", "-u")
	if result.exitCode != 0 {
		t.Fatalf("run() = %+v", result)
	}
	parts := strings.Split(strings.TrimSpace(result.stdout), "-")
	if len(parts) != 2 || []rune(parts[0])[0] != []rune(parts[1])[0] {
		t.Fatalf("run() did not alliterate: %q", result.stdout)
	}
	for _, part := range parts {
		if len([]rune(part)) > 6 {
			t.Fatalf("run() exceeded maximum length: %q", result.stdout)
		}
	}
}

func TestSupportsTolkienStrategy(t *testing.T) {
	options, err := parse([]string{"--strategy", "tolkien"})
	if err != nil {
		t.Fatal(err)
	}
	if options.Strategy != goname.StrategyTolkien {
		t.Fatalf("strategy = %v, want Tolkien", options.Strategy)
	}
	result := run("-t", "tolkien", "--name")
	if result.exitCode != 0 || result.stdout == "" || result.stderr != "" {
		t.Fatalf("run() = %+v", result)
	}
}

func TestSupportsTokenStrategiesAndTheirWordCounts(t *testing.T) {
	for _, test := range []struct {
		name     string
		strategy goname.Strategy
		length   int
		alphabet string
	}{
		{"hex", goname.StrategyHex, 4, "0123456789abcdef"},
		{"base32", goname.StrategyBase32, 4, "0123456789abcdefghjkmnpqrstvwxyz"},
		{"ulid", goname.StrategyULID, 26, "0123456789abcdefghjkmnpqrstvwxyz"},
	} {
		t.Run(test.name, func(t *testing.T) {
			options, err := parse([]string{"--strategy", test.name})
			if err != nil || options.Strategy != test.strategy {
				t.Fatalf("parse() = (%+v, %v), want strategy %d", options, err, test.strategy)
			}

			result := run("--strategy", test.name)
			word := strings.TrimSpace(result.stdout)
			if result.exitCode != 0 || result.stderr != "" || len(word) != test.length || !onlyTokenChars(word, test.alphabet) {
				t.Fatalf("default run() = %+v, want one %d-character token", result, test.length)
			}

			result = run("--strategy", test.name, "--words", "2", "--separator", ":", "--mixedcase")
			parts := strings.Split(strings.TrimSpace(result.stdout), ":")
			if result.exitCode != 0 || result.stderr != "" || len(parts) != 2 {
				t.Fatalf("two-token run() = %+v, want two tokens", result)
			}
			for _, part := range parts {
				if len(part) != test.length || !onlyTokenChars(part, strings.ToUpper(test.alphabet)) {
					t.Errorf("mixed-case token %q has invalid length or symbols", part)
				}
			}

			result = run("--strategy", test.name, "--letters", "6")
			word = strings.TrimSpace(result.stdout)
			wantLength := 6
			if test.strategy == goname.StrategyULID {
				wantLength = 26
			}
			if result.exitCode != 0 || result.stderr != "" || len(word) != wantLength {
				t.Errorf("run() with --letters 6 = %+v, want a %d-character token", result, wantLength)
			}
		})
	}
}

func TestMixedCaseOptionPreservesDictionaryCase(t *testing.T) {
	directory := writeWords(t, "swiftly", "frosty", "Aragorn")

	for _, test := range []struct {
		option string
		want   string
	}{
		{option: "", want: "frosty-aragorn\n"},
		{option: "-m", want: "frosty-Aragorn\n"},
		{option: "--mixedcase", want: "frosty-Aragorn\n"},
	} {
		args := []string{"--dir", directory}
		if test.option != "" {
			args = append(args, test.option)
		}
		result := run(args...)
		if result.exitCode != 0 || result.stdout != test.want || result.stderr != "" {
			t.Errorf("run(%q) = %+v, want stdout %q", args, result, test.want)
		}
	}
}

func TestReportsInvalidArguments(t *testing.T) {
	tests := []struct {
		args    []string
		message string
	}{
		{[]string{"--complexity", "7"}, "complexity must be 0"},
		{[]string{"--strategy", "fantasy"}, "strategy must be tolkien"},
		{[]string{"--complexity", "1", "--strategy", "tolkien"}, "cannot be combined with a complexity tier"},
		{[]string{"--words", "0"}, "words must be a positive integer"},
		{[]string{"--words", "2147483648"}, "words is too large"},
		{[]string{"--letters", "-1"}, "letters must be a positive integer"},
		{[]string{"--separator"}, "missing value for --separator"},
		{[]string{"--prefix"}, "missing value for --prefix"},
		{[]string{"--unknown"}, "Unknown options [--unknown]"},
	}
	for _, test := range tests {
		result := run(test.args...)
		if result.exitCode != 1 || !strings.Contains(result.stderr, test.message) {
			t.Errorf("run(%q) = %+v, want error containing %q", test.args, result, test.message)
		}
	}
}

func TestHelpIsSelfContainedAndTakesPrecedence(t *testing.T) {
	result := run("--bad", "--help")
	if result.exitCode != 0 || result.stderr != "" {
		t.Fatalf("run() = %+v", result)
	}
	for _, expected := range []string{"Usage: goname", "--ubuntu", "--adverb", "--strategy", "--prefix", "-m|--mixedcase"} {
		if !strings.Contains(result.stdout, expected) {
			t.Errorf("help does not contain %q", expected)
		}
	}
}

type result struct {
	exitCode int
	stdout   string
	stderr   string
}

func run(args ...string) result {
	var stdout, stderr bytes.Buffer
	exitCode := Run(args, &stdout, &stderr)
	return result{exitCode: exitCode, stdout: stdout.String(), stderr: stderr.String()}
}

func onlyTokenChars(token, alphabet string) bool {
	if token == "" {
		return false
	}
	for _, character := range token {
		if !strings.ContainsRune(alphabet, character) {
			return false
		}
	}
	return true
}

func writeWords(t *testing.T, adverbs, adjectives, names string) string {
	t.Helper()
	directory := t.TempDir()
	writeWordsAt(t, directory, adverbs, adjectives, names)
	return directory
}

func writeWordsAt(t *testing.T, directory, adverbs, adjectives, names string) {
	t.Helper()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, contents := range map[string]string{
		"adverbs.txt": adverbs, "adjectives.txt": adjectives, "names.txt": names,
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

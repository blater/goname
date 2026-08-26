// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestReportsInvalidArguments(t *testing.T) {
	tests := []struct {
		args    []string
		message string
	}{
		{[]string{"--complexity", "7"}, "complexity must be 0"},
		{[]string{"--words", "0"}, "words must be a positive integer"},
		{[]string{"--words", "2147483648"}, "words is too large"},
		{[]string{"--letters", "-1"}, "letters must be a positive integer"},
		{[]string{"--separator"}, "missing value for --separator"},
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
	for _, expected := range []string{"Usage: goname", "--ubuntu", "--adverb"} {
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

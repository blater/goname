// SPDX-License-Identifier: MIT

package goname

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

func TestFollowsUpstreamWordGrammar(t *testing.T) {
	directory := writeWords(t, "swiftly", "calm", "otter")
	generator := NewSeededGenerator(1)

	tests := []struct {
		words     int
		separator string
		want      string
	}{
		{1, "-", "otter"},
		{2, "-", "calm-otter"},
		{3, "_", "swiftly_calm_otter"},
		{4, ":", "swiftly:swiftly:calm:otter"},
	}
	for _, test := range tests {
		options := DefaultOptions()
		options.WordDirectory = directory
		options.Words = test.words
		options.Separator = test.separator
		got, err := generator.Generate(options)
		if err != nil {
			t.Fatalf("Generate(%d): %v", test.words, err)
		}
		if got != test.want {
			t.Errorf("Generate(%d) = %q, want %q", test.words, got, test.want)
		}
	}
}

func TestFiltersByUnicodeCodePointLength(t *testing.T) {
	directory := writeWords(t, "aptly\npatiently", "calm\ntranquil", "🦉owl\nalpaca")
	options := DefaultOptions()
	options.WordDirectory = directory
	options.Words = 3
	options.MaxLetters = 5

	got, err := NewSeededGenerator(1).Generate(options)
	if err != nil {
		t.Fatal(err)
	}
	if got != "aptly-calm-🦉owl" {
		t.Fatalf("Generate() = %q, want %q", got, "aptly-calm-🦉owl")
	}
}

func TestGeneratesTrueAlliteration(t *testing.T) {
	directory := writeWords(t, "ably\nboldly", "agile\nbrisk", "alpaca\nbadger")
	options := DefaultOptions()
	options.WordDirectory = directory
	options.Words = 4
	options.Alliterate = true

	got, err := NewSeededGenerator(7).Generate(options)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(got, "-")
	if len(parts) != 4 {
		t.Fatalf("Generate() returned %d words: %q", len(parts), got)
	}
	initial, _ := utf8.DecodeRuneInString(parts[0])
	for _, part := range parts[1:] {
		partInitial, _ := utf8.DecodeRuneInString(part)
		if partInitial != initial {
			t.Fatalf("Generate() did not alliterate: %q", got)
		}
	}
}

func TestSupportsIndividualWordTypes(t *testing.T) {
	directory := writeWords(t, "swiftly", "calm", "otter")
	for nameType, want := range map[Type]string{
		TypeAdverb:    "swiftly",
		TypeAdjective: "calm",
		TypeName:      "otter",
	} {
		options := DefaultOptions()
		options.WordDirectory = directory
		options.Type = nameType
		got, err := NewSeededGenerator(1).Generate(options)
		if err != nil {
			t.Fatalf("Generate(%v): %v", nameType, err)
		}
		if got != want {
			t.Errorf("Generate(%v) = %q, want %q", nameType, got, want)
		}
	}
}

func TestResolvesComplexityBelowCustomDirectory(t *testing.T) {
	directory := t.TempDir()
	writeWordsAt(t, filepath.Join(directory, "small"), "aptly", "calm", "ibis")
	options := DefaultOptions()
	options.WordDirectory = directory
	options.Complexity = ComplexitySmall

	got, err := NewSeededGenerator(1).Generate(options)
	if err != nil {
		t.Fatal(err)
	}
	if got != "calm-ibis" {
		t.Fatalf("Generate() = %q, want calm-ibis", got)
	}
}

func TestRejectsImpossibleConfigurations(t *testing.T) {
	directory := writeWords(t, "swiftly", "calm", "otter")
	tests := []struct {
		name    string
		mutate  func(*Options)
		message string
	}{
		{"zero words", func(o *Options) { o.Words = 0 }, "words must be a positive integer"},
		{"negative length", func(o *Options) { o.MaxLetters = -1 }, "maxLetters must be zero"},
		{"too short", func(o *Options) { o.MaxLetters = 2 }, "no words satisfy"},
		{"long separator", func(o *Options) { o.Separator = strings.Repeat("🦉", 101) }, "separator must be"},
		{"invalid type", func(o *Options) { o.Type = Type(99) }, "invalid name type"},
		{"invalid complexity", func(o *Options) { o.Complexity = Complexity(99) }, "invalid complexity"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := DefaultOptions()
			options.WordDirectory = directory
			test.mutate(&options)
			_, err := NewSeededGenerator(1).Generate(options)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("Generate() error = %v, want containing %q", err, test.message)
			}
		})
	}
}

func TestRejectsAlliterationWithoutCommonInitial(t *testing.T) {
	directory := writeWords(t, "boldly", "agile", "alpaca")
	options := DefaultOptions()
	options.WordDirectory = directory
	options.Words = 3
	options.Alliterate = true
	_, err := NewSeededGenerator(1).Generate(options)
	if err == nil || !strings.Contains(err.Error(), "no alliterative goname") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestSeededGeneratorIsRepeatable(t *testing.T) {
	options := DefaultOptions()
	options.Words = 4
	first := NewSeededGenerator(42)
	second := NewSeededGenerator(42)
	for range 20 {
		left, leftErr := first.Generate(options)
		right, rightErr := second.Generate(options)
		if leftErr != nil || rightErr != nil {
			t.Fatalf("Generate() errors = %v, %v", leftErr, rightErr)
		}
		if left != right {
			t.Fatalf("seeded generators differ: %q != %q", left, right)
		}
	}
}

func TestGeneratorIsSafeForConcurrentUse(t *testing.T) {
	options := DefaultOptions()
	generator := NewSeededGenerator(1)
	var wait sync.WaitGroup
	for range 25 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := generator.Generate(options); err != nil {
				t.Errorf("Generate(): %v", err)
			}
		}()
	}
	wait.Wait()
}

func TestBuiltInWordListsMatchPinnedCopies(t *testing.T) {
	hashes := map[string]string{
		"large/adjectives.txt":  "8a36e132c66fc2a879770dfc1f73f2a11b454194ca75f3f0fde3e6922965608c",
		"large/adverbs.txt":     "95b8d19579711acc199a3a6a6e102d5abd3d2df57bc052166d6009f651241591",
		"large/names.txt":       "6e0db1b0619462388115fb2ae26277db4391052f6161842c15f8d8189078a14c",
		"medium/adjectives.txt": "b1cd54d8f27d4514aaa9bbf4a8248b696867b63b75ffcbe49b277227e2875a13",
		"medium/adverbs.txt":    "c19e6fc3d06acf6d5a3812772f60012e443cc96f41e63c5e88e0007e7111646c",
		"medium/names.txt":      "0f37daddd7e68ba01240032188b542d32a838f2352197d04c1bb0f46a22f647a",
		"small/adjectives.txt":  "5f934ce94a217b85aec434d6803c6fc5b74628d10d83c4d0ddfdec9256e062b0",
		"small/adverbs.txt":     "f22baacc9c281e6e13daea8dcc8d204c5ad08b26d0c4cd5a1ea0e6e1e782b445",
		"small/names.txt":       "a15d352808bda5b983dad1d68d71f97dd5819c28fe5779169e8ce9c82f0aff6b",
	}
	for path, want := range hashes {
		data, err := embeddedWords.ReadFile("words/" + path)
		if err != nil {
			t.Errorf("ReadFile(%q): %v", path, err)
			continue
		}
		got := fmt.Sprintf("%x", sha256.Sum256(data))
		if got != want {
			t.Errorf("hash(%q) = %s, want %s", path, got, want)
		}
	}
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

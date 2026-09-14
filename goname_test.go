// SPDX-License-Identifier: MIT

package goname

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"
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

func TestTokenStrategiesUseTheirDefaultsAndWordControls(t *testing.T) {
	tests := []struct {
		name     string
		strategy Strategy
		length   int
		valid    func(string) bool
	}{
		{"hex", StrategyHex, 4, func(word string) bool { return tokenContainsOnly(word, "0123456789abcdef") }},
		{"base32", StrategyBase32, 4, func(word string) bool { return tokenContainsOnly(word, "0123456789abcdefghjkmnpqrstvwxyz") }},
		{"ulid", StrategyULID, 26, func(word string) bool { return isValidULID(word) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := DefaultOptions()
			options.Strategy = test.strategy
			got, err := NewSeededGenerator(12).Generate(options)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(got, options.Separator) || len(got) != test.length || !test.valid(got) {
				t.Fatalf("default token = %q, want one valid token", got)
			}

			options.Words = 3
			options.Separator = ":"
			got, err = NewSeededGenerator(12).Generate(options)
			if err != nil {
				t.Fatal(err)
			}
			parts := strings.Split(got, ":")
			if len(parts) != 3 {
				t.Fatalf("three-word token = %q, want 3 parts", got)
			}
			for _, word := range parts {
				if len(word) != test.length || !test.valid(word) {
					t.Errorf("invalid %s token %q", test.name, word)
				}
			}
		})
	}
}

func TestHexAndBase32MixedCaseAreUppercase(t *testing.T) {
	for strategy, alphabet := range map[Strategy]string{
		StrategyHex:    "0123456789ABCDEF",
		StrategyBase32: "0123456789ABCDEFGHJKMNPQRSTVWXYZ",
	} {
		options := DefaultOptions()
		options.Strategy = strategy
		options.MixedCase = true
		got, err := NewSeededGenerator(9).Generate(options)
		if err != nil {
			t.Fatal(err)
		}
		if !tokenContainsOnly(got, alphabet) {
			t.Errorf("strategy %d mixed-case token %q contains non-uppercase symbols", strategy, got)
		}
	}
}

func TestTokenStrategiesCanExplicitlyRequestTwoWords(t *testing.T) {
	for _, strategy := range []Strategy{StrategyHex, StrategyBase32, StrategyULID} {
		options := DefaultOptions()
		options.Strategy = strategy
		options.Words = 2
		got, err := NewSeededGenerator(3).Generate(options)
		if err != nil {
			t.Fatal(err)
		}
		if parts := strings.Split(got, options.Separator); len(parts) != 2 {
			t.Errorf("strategy %d with explicit Words=2 = %q, want two tokens", strategy, got)
		}
	}
}

func TestLettersSetHexAndBase32WidthButDoNotChangeULIDWidth(t *testing.T) {
	for _, strategy := range []Strategy{StrategyHex, StrategyBase32} {
		options := DefaultOptions()
		options.Strategy = strategy
		options.MaxLetters = 7
		got, err := NewSeededGenerator(11).Generate(options)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 7 {
			t.Errorf("strategy %d with MaxLetters=7 generated %q (%d chars)", strategy, got, len(got))
		}
	}

	options := DefaultOptions()
	options.Strategy = StrategyULID
	options.MaxLetters = 3
	got, err := NewSeededGenerator(11).Generate(options)
	if err != nil {
		t.Fatal(err)
	}
	if !isValidULID(got) {
		t.Fatalf("ULID with MaxLetters=3 = %q, want fixed-width ULID", got)
	}
}

func TestULIDContainsCurrentTimestampAndLowercaseCrockfordEncoding(t *testing.T) {
	before := time.Now().UnixMilli()
	options := DefaultOptions()
	options.Strategy = StrategyULID
	got, err := NewSeededGenerator(4).Generate(options)
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now().UnixMilli()
	if !isValidULID(got) {
		t.Fatalf("Generate() = %q, want lowercase 26-character ULID", got)
	}
	encoded := ulidInteger(got)
	encoded.Rsh(encoded, 80)
	timestamp := encoded.Int64()
	if timestamp < before || timestamp > after {
		t.Fatalf("ULID timestamp = %d, want between %d and %d", timestamp, before, after)
	}
}

func TestULIDsAreMonotonicWithinOneMillisecond(t *testing.T) {
	generator := NewSeededGenerator(23)
	now := time.UnixMilli(1_700_000_000_000)
	generator.now = func() time.Time { return now }
	options := DefaultOptions()
	options.Strategy = StrategyULID

	previous := new(big.Int)
	for index := 0; index < 10; index++ {
		got, err := generator.Generate(options)
		if err != nil {
			t.Fatal(err)
		}
		current := ulidInteger(got)
		if index > 0 && current.Cmp(previous) <= 0 {
			t.Fatalf("ULID %q is not greater than its predecessor", got)
		}
		previous = current
	}
}

func TestULIDMonotonicitySurvivesClockRollback(t *testing.T) {
	times := []time.Time{
		time.UnixMilli(1_700_000_000_001),
		time.UnixMilli(1_700_000_000_000),
	}
	call := 0
	generator := NewSeededGenerator(24)
	generator.now = func() time.Time {
		now := times[call]
		call++
		return now
	}
	options := DefaultOptions()
	options.Strategy = StrategyULID

	first, err := generator.Generate(options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := generator.Generate(options)
	if err != nil {
		t.Fatal(err)
	}
	if ulidInteger(second).Cmp(ulidInteger(first)) <= 0 {
		t.Fatalf("ULID after clock rollback %q is not greater than %q", second, first)
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

func TestTolkienStrategyAddsThemedWordsAndNames(t *testing.T) {
	options := DefaultOptions()
	options.Strategy = StrategyTolkien
	words, err := loadWordLists(options)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"quickly", "anciently"} {
		if !containsWord(words.adverbs, want) {
			t.Errorf("Tolkien adverbs do not contain %q", want)
		}
	}
	for _, want := range []string{"calm", "grandeur", "wielding"} {
		if !containsWord(words.adjectives, want) {
			t.Errorf("Tolkien adjectives do not contain %q", want)
		}
	}
	for _, want := range []string{"Aragorn", "Feanor", "Gandalf", "Luthien", "Smaug"} {
		if !containsWord(words.names, want) {
			t.Errorf("Tolkien names do not contain %q", want)
		}
	}
	for _, category := range []struct {
		name string
		base []string
		got  []string
	}{
		{"adverbs", wordsFromSmall(t, "adverbs"), words.adverbs},
		{"adjectives", wordsFromSmall(t, "adjectives"), words.adjectives},
	} {
		for _, word := range category.base {
			if !containsWord(category.got, word) {
				t.Errorf("Tolkien %s do not include small-list word %q", category.name, word)
			}
		}
	}
	seenNames := make(map[string]struct{}, len(words.names))
	for _, name := range words.names {
		if _, exists := seenNames[name]; exists {
			t.Errorf("Tolkien names contain duplicate %q", name)
		}
		seenNames[name] = struct{}{}
	}

	options.Type = TypeName
	options.MixedCase = true
	got, err := NewSeededGenerator(7).Generate(options)
	if err != nil {
		t.Fatal(err)
	}
	if !containsWord(words.names, got) {
		t.Fatalf("generated Tolkien name %q is not in the name list", got)
	}
}

func TestTolkienStrategyResolvesCustomDirectory(t *testing.T) {
	directory := t.TempDir()
	writeWordsAt(t, filepath.Join(directory, "tolkien"), "nobly", "ancient", "Aragorn")
	options := DefaultOptions()
	options.Strategy = StrategyTolkien
	options.WordDirectory = directory
	got, err := NewSeededGenerator(1).Generate(options)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ancient-aragorn" {
		t.Fatalf("Generate() = %q, want ancient-aragorn", got)
	}
}

func TestRejectsImpossibleConfigurations(t *testing.T) {
	directory := writeWords(t, "swiftly", "calm", "otter")
	tests := []struct {
		name    string
		mutate  func(*Options)
		message string
	}{
		{"negative words", func(o *Options) { o.Words = -1 }, "words must be zero or a positive integer"},
		{"negative length", func(o *Options) { o.MaxLetters = -1 }, "maxLetters must be zero"},
		{"too short", func(o *Options) { o.MaxLetters = 2 }, "no words satisfy"},
		{"long separator", func(o *Options) { o.Separator = strings.Repeat("🦉", 101) }, "separator must be"},
		{"invalid type", func(o *Options) { o.Type = Type(99) }, "invalid name type"},
		{"invalid complexity", func(o *Options) { o.Complexity = Complexity(99) }, "invalid complexity"},
		{"invalid strategy", func(o *Options) { o.Strategy = Strategy(99) }, "invalid strategy"},
		{"strategy and complexity", func(o *Options) {
			o.Strategy = StrategyTolkien
			o.Complexity = ComplexitySmall
		}, "cannot be combined with a complexity tier"},
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

func TestAlliterationMatchesCapitalizedWordsCaseInsensitively(t *testing.T) {
	directory := writeWords(t, "Able", "Agile", "Aragorn")
	options := DefaultOptions()
	options.WordDirectory = directory
	options.Words = 3
	options.Alliterate = true
	got, err := NewSeededGenerator(1).Generate(options)
	if err != nil {
		t.Fatal(err)
	}
	if got != "able-agile-aragorn" {
		t.Fatalf("Generate() = %q, want able-agile-aragorn", got)
	}
}

func TestTolkienStrategyAlliteratesAcrossCapitalizedNames(t *testing.T) {
	options := DefaultOptions()
	options.Strategy = StrategyTolkien
	options.Words = 3
	options.Alliterate = true
	options.MixedCase = true
	got, err := NewSeededGenerator(1).Generate(options)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(got, options.Separator)
	initial := normalizedInitial(parts[0])
	for _, word := range parts[1:] {
		if normalizedInitial(word) != initial {
			t.Fatalf("Tolkien name does not alliterate: %q", got)
		}
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
		"medium/adjectives.txt": "3b91020cc035b00b912f7cfb0b77a66a8465ee1e3dd1880b246699b57e703f2b",
		"medium/adverbs.txt":    "c19e6fc3d06acf6d5a3812772f60012e443cc96f41e63c5e88e0007e7111646c",
		"medium/names.txt":      "2d929c68b1e1e431ab7eff914514e8ea7f70d05e54d2fb3d9a9d6c90deb7a0c9",
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

func TestMediumWordListsIncludeSmallWordLists(t *testing.T) {
	for _, category := range []string{"adverbs", "adjectives", "names"} {
		smallData, err := embeddedWords.ReadFile("words/small/" + category + ".txt")
		if err != nil {
			t.Fatalf("ReadFile(small/%s): %v", category, err)
		}
		mediumData, err := embeddedWords.ReadFile("words/medium/" + category + ".txt")
		if err != nil {
			t.Fatalf("ReadFile(medium/%s): %v", category, err)
		}
		mediumWords := make(map[string]struct{})
		for _, word := range strings.Fields(string(mediumData)) {
			mediumWords[word] = struct{}{}
		}
		for _, word := range strings.Fields(string(smallData)) {
			if _, ok := mediumWords[word]; !ok {
				t.Errorf("medium/%s.txt is missing small-list word %q", category, word)
			}
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

func containsWord(words []string, want string) bool {
	for _, word := range words {
		if word == want {
			return true
		}
	}
	return false
}

func tokenContainsOnly(token, alphabet string) bool {
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

func isValidULID(token string) bool {
	return len(token) == 26 && token[0] <= '7' &&
		tokenContainsOnly(token, "0123456789abcdefghjkmnpqrstvwxyz")
}

func ulidInteger(token string) *big.Int {
	encoded := new(big.Int)
	for _, character := range token {
		index := strings.IndexRune(crockfordBase32, unicode.ToUpper(character))
		encoded.Mul(encoded, big.NewInt(32))
		encoded.Add(encoded, big.NewInt(int64(index)))
	}
	return encoded
}

func wordsFromSmall(t *testing.T, category string) []string {
	t.Helper()
	data, err := embeddedWords.ReadFile("words/small/" + category + ".txt")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(data))
}

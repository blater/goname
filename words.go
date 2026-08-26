// SPDX-License-Identifier: MIT

package goname

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed words/*/*.txt
var embeddedWords embed.FS

type wordLists struct {
	adverbs    []string
	adjectives []string
	names      []string
}

var builtInCache struct {
	sync.Mutex
	words map[Complexity]wordLists
}

func loadWordLists(options Options) (wordLists, error) {
	if options.WordDirectory == "" {
		complexity := options.Complexity
		if complexity == ComplexityDefault {
			complexity = ComplexityMedium
		}
		return loadBuiltIn(complexity)
	}

	directory, err := filepath.Abs(options.WordDirectory)
	if err != nil {
		return wordLists{}, fmt.Errorf("resolve word directory: %w", err)
	}
	directory = filepath.Clean(directory)
	if options.Complexity != ComplexityDefault {
		directory = filepath.Join(directory, complexityDirectory(options.Complexity))
	}
	return loadDirectory(directory)
}

func loadBuiltIn(complexity Complexity) (wordLists, error) {
	builtInCache.Lock()
	defer builtInCache.Unlock()
	if builtInCache.words == nil {
		builtInCache.words = make(map[Complexity]wordLists)
	}
	if words, ok := builtInCache.words[complexity]; ok {
		return words, nil
	}

	directory := complexityDirectory(complexity)
	words, err := loadFiles(func(name string) ([]byte, error) {
		path := "words/" + directory + "/" + name
		data, readErr := embeddedWords.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("missing built-in word list: %s", path)
		}
		return data, nil
	})
	if err != nil {
		return wordLists{}, err
	}
	builtInCache.words[complexity] = words
	return words, nil
}

func loadDirectory(directory string) (wordLists, error) {
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return wordLists{}, fmt.Errorf("word directory does not exist or is not readable: %s", directory)
	}
	return loadFiles(func(name string) ([]byte, error) {
		path := filepath.Join(directory, name)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("cannot read word list: %s: %w", path, readErr)
		}
		return data, nil
	})
}

func loadFiles(read func(string) ([]byte, error)) (wordLists, error) {
	adverbs, err := readWordList(read, "adverbs.txt")
	if err != nil {
		return wordLists{}, err
	}
	adjectives, err := readWordList(read, "adjectives.txt")
	if err != nil {
		return wordLists{}, err
	}
	names, err := readWordList(read, "names.txt")
	if err != nil {
		return wordLists{}, err
	}
	return wordLists{adverbs: adverbs, adjectives: adjectives, names: names}, nil
}

func readWordList(read func(string) ([]byte, error), name string) ([]string, error) {
	data, err := read(name)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	words := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			words = append(words, line)
		}
	}
	if len(words) == 0 {
		return nil, fmt.Errorf("word list is empty: %s", name)
	}
	return words, nil
}

func complexityDirectory(complexity Complexity) string {
	switch complexity {
	case ComplexitySmall:
		return "small"
	case ComplexityMedium:
		return "medium"
	case ComplexityLarge:
		return "large"
	default:
		return ""
	}
}

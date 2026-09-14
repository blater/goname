# goname

Goname is a small, dependency-free Go implementation of
[Dustin Kirkland's petname](https://github.com/dustinkirkland/petname). It
generates random names from adverbs, adjectives, and animal, character, or
place names, as well as hexadecimal, Base32, and ULID tokens. Use it as either
a Go package or a command-line program.

## Examples

```console
$ goname
plausible-dace
$ goname -s _
foxy_squirrel
$ goname --ubuntu
vehement-vulture
$ goname --adjective
rapid
$ goname --strategy tolkien
ancient-aragorn
$ goname --strategy tolkien --mixedcase
ancient-Aragorn
$ goname --strategy hex
8d31
$ goname --strategy base32 --words 2
9f2c-j7wx
$ goname --strategy ulid
01arz3ndektsv4rrffq69g5fav
$ goname -p ticket
ticket-benign-puffin
$ goname -p ticket -s _ --strategy tolkien
ticket_relevant_rose
```

## Command-line usage

```text
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
```

Custom dictionary directories contain `adverbs.txt`, `adjectives.txt`, and
`names.txt`, with one word per line. When a complexity is selected, those
files are loaded from the corresponding `small`, `medium`, or `large`
subdirectory. `--strategy tolkien` selects Tolkien character and place names
with themed modifiers. It reuses the small adverb and adjective lists, then
adds fantasy-flavoured modifiers. A custom word directory supplies complete
lists from its `tolkien` subdirectory when this strategy is selected.

## Generation strategies

The default strategy builds names from the selected word lists. `tolkien`
selects Tolkien-themed modifiers and names. A non-default strategy cannot be
combined with `--complexity`.

`hex` generates four-character hexadecimal tokens. `base32` generates
four-character tokens from Crockford's alphabet,
`0123456789ABCDEFGHJKMNPQRSTVWXYZ`, which excludes I, L, O, and U. Both are
lowercase by default. A positive `--letters` value sets their token width.
`ulid` generates a standard 26-character ULID using the current time and random
entropy; its fixed width is unaffected by `--letters`. All three token
strategies generate one token by default. Use `--words` and `--separator` to
generate multiple tokens.

Dictionary words are lowercase by default. Pass `-m` or `--mixedcase` to
preserve their spelling from the selected word lists. Token output is lowercase
by default and uppercase with `-m` or `--mixedcase`.

Use `-p` or `--prefix` to put a prefix before the generated name. The
configured separator goes between the prefix and name, and the prefix does not
count toward `--words`. For example, `goname -p ticket -s _ --strategy
tolkien` produces a result such as `ticket_relevant_rose`.

## Go API

```go
package main

import (
	"fmt"
	"log"

	"github.com/blater/goname"
)

func main() {
	name, err := goname.Generate()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(name)

	options := goname.DefaultOptions()
	options.Words = 3
	options.Separator = "_"
	options.MaxLetters = 8
	options.Complexity = goname.ComplexityMedium
	options.Alliterate = true

	name, err = goname.GenerateWithOptions(options)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(name)

	options = goname.DefaultOptions()
	options.Strategy = goname.StrategyBase32
	options.Words = 2
	options.Prefix = "ticket"

	name, err = goname.GenerateWithOptions(options)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(name) // e.g. ticket-9f2c-j7wx
}
```

For Tolkien names, set `options.Strategy = goname.StrategyTolkien` instead of
setting a complexity tier. The CLI equivalent is `goname --strategy tolkien`.
Set `options.Strategy` to `goname.StrategyHex`, `goname.StrategyBase32`, or
`goname.StrategyULID` to generate tokens. These strategies default to one
token when `Words` is zero; set any positive `Words` value to choose the token
count, including two. Set `options.Prefix` to prepend a prefix using the
configured separator; the prefix does not count toward `Words`. The CLI uses
`-p` or `--prefix` for the same behavior.

`NewGenerator` uses cryptographically secure randomness.
`NewSeededGenerator` provides deterministic dictionary and token randomness
(ULID timestamps still reflect the current time), and
`NewGeneratorWithRandom` accepts an application-controlled `math/rand`-style
source. Generators are safe for concurrent use.

## Build

Go 1.24 or newer and Make are required.

```sh
make          # build bin/goname
make test     # run all tests
make verify   # format, vet, test, build, and smoke-test
make install  # install with go install
```

The word lists are embedded in the executable. The built program does not
read or otherwise depend on the sibling `jname` project.

## Origin, attribution, and license

This project is based on the design and shell implementation of
[dustinkirkland/petname](https://github.com/dustinkirkland/petname). Its
bundled dictionaries are copied from upstream petname 2.11 commit
`70ed924cb96c290ac051b8ae797417c4adbbb5c9`.

Goname is licensed under the [MIT License](LICENSE). The copied dictionaries
remain licensed under the Apache License, Version 2.0. Their provenance is
recorded in [NOTICE](NOTICE) and [words/UPSTREAM.md](words/UPSTREAM.md), and the
Apache license text is included in [LICENSES/Apache-2.0.txt](LICENSES/Apache-2.0.txt).

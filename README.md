# goname

Goname is a small, dependency-free Go implementation of
[Dustin Kirkland's petname](https://github.com/dustinkirkland/petname). It
generates human-readable random names from adverbs, adjectives, and animal
names and can be used as either a Go package or a command-line program.

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
```

## Command-line usage

```text
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
```

Custom dictionary directories contain `adverbs.txt`, `adjectives.txt`, and
`names.txt`, with one word per line. When a complexity is selected, those
files are loaded from the corresponding `small`, `medium`, or `large`
subdirectory.

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
}
```

`NewGenerator` uses cryptographically secure randomness.
`NewSeededGenerator` provides deterministic generation, and
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

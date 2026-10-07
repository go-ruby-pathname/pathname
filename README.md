<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-pathname/brand/main/social/go-ruby-pathname-pathname.png" alt="go-ruby-pathname/pathname" width="720"></p>

# pathname — go-ruby-pathname

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-pathname.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.1%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the path-manipulation surface of Ruby's
[`pathname`](https://docs.ruby-lang.org/en/master/Pathname.html) standard
library** — the deterministic, interpreter-independent core of MRI 4.0.5's
`Pathname`. It wraps a path string and exposes the lexical operations Ruby's
`Pathname` offers (`basename`, `dirname`, `extname`, `cleanpath`,
`relative_path_from`, `join`/`+`, `split`, `each_filename`, `ascend`/`descend`,
`sub_ext`, comparison) — **without any Ruby runtime** and **without touching the
filesystem**.

It is the path-algebra backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module — a sibling of
[go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) (the Onigmo engine),
[go-ruby-erb](https://github.com/go-ruby-erb/erb) (the ERB compiler) and
[go-ruby-yaml](https://github.com/go-ruby-yaml/yaml) (the Psych port).

> **What it is — and isn't.** Ruby's `Pathname` is two libraries in one: a pure
> path algebra (lexical string surgery, no I/O) and a thin set of delegations to
> `File`/`Dir`/`IO` (`read`, `write`, `exist?`, `children`, …). This package is
> the first half — fully deterministic, needs no interpreter, lives here as pure
> Go. The filesystem-touching half stays host-side in go-embedded-ruby, where it
> forwards to the host's `File` class; it is deliberately **out of scope** here.

## Lexical, `/`-based, platform-independent

Ruby's `Pathname` hardcodes `"/"` as the component separator for **every** lexical
operation on **every** platform — it never consults the OS path separator for
`join`/`split`/`cleanpath`/`basename`. This package does the same: it uses `"/"`
unconditionally, so the behaviour (and the test suite) is identical on Linux,
macOS and Windows. The module therefore builds and passes `GOOS=windows` with no
OS-specific path code. (Ruby's own `File.basename` etc. are `\`-aware on Windows;
`Pathname`'s lexical methods are not, and neither is this port.)

## Features

Faithful port of `Pathname`'s pure path methods, validated against the `ruby`
binary on every supported platform:

- **`New` / `ToS` / `Inspect`** — wrap a path, render it, `#<Pathname:…>`.
- **`Basename` / `BasenameSuffix`** — last component, with optional suffix strip
  (`".*"` strips any extension).
- **`Dirname` / `Parent`** — all but the last component (`.`, `/` edge cases).
- **`Extname` / `SubExt`** — MRI's `File.extname` rule **byte-for-byte**,
  including the fiddly trailing-dot behaviour (`"foo." → "."`, `"a..b" → ".b"`)
  and dotfile rule (`".foo" → ""`).
- **`Cleanpath`** — collapse `.`, `..` and redundant separators; a `..` past the
  root of an absolute path is dropped, a `..` escaping a relative path is kept
  (`"a/./b/../c" → "a/c"`, `"/a/../../b" → "/b"`, `"a/../../b" → "../b"`).
- **`RelativePathFrom`** — MRI's lexical `relative_path_from` algorithm, with its
  two `ArgumentError` cases (mixing absolute/relative; a `..` in the base),
  message text matching MRI.
- **`Plus` / `Join` (`+`, `join`)** — append components; an absolute component
  resets to the root.
- **`Split` / `EachFilename` / `Filenames`** — `[dirname, basename]` and the
  non-empty component list.
- **`Ascend` / `Descend`** — the path and each parent up to the root (or the
  first relative component), and the reverse.
- **`Absolute` / `Relative` / `Root`** — the predicate trio.
- **`Cmp` / `Eql` / `Hash`** — `<=>`, `==`/`eql?`, a stable FNV hash.

CGO-free, dependency-free, **100% test coverage**, `gofmt` + `go vet` clean, and
green across the six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le,
s390x) and the three host OSes (Linux, macOS, Windows).

## Install

```sh
go get github.com/go-ruby-pathname/pathname
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-ruby-pathname/pathname"
)

func main() {
	p := pathname.New("a/./b/../c")
	fmt.Println(p.Cleanpath().ToS()) // a/c

	fmt.Println(pathname.New("/usr/bin/ruby.rb").BasenameSuffix(".*").ToS()) // ruby
	fmt.Println(pathname.New("foo.tar.gz").Extname())                       // .gz

	rel, _ := pathname.New("/a/b/c").RelativePathFrom(pathname.New("/a/x/y"))
	fmt.Println(rel.ToS()) // ../../b/c

	fmt.Println(pathname.New("a").JoinStrings("b", "/c", "d").ToS()) // /c/d
}
```

## API

`*Pathname` is an immutable wrapper over a path string; every transforming method
returns a fresh `*Pathname`, `string`, `[]*Pathname` or `bool`, mirroring the
corresponding Ruby method. `RelativePathFrom` returns `(*Pathname, error)` where
the error is an `*ArgumentError` carrying MRI's exact message. `Ascend`,
`Descend` and `EachFilename` take a `func` callback (the Go analogue of Ruby's
block); `Filenames` returns the slice form.

## Differential testing against MRI

The `*_oracle_test.go` suite shells out to the `ruby` binary (gated on
`RUBY_VERSION >= "4.0"`) and checks each method against MRI's `Pathname` over a
corpus of edge cases — the trailing-slash / `.` / `..` / root-escape cleanpath
cases, the trailing-dot extname cases, and the `relative_path_from`
`ArgumentError` messages. The oracle skips itself where `ruby` is absent (the
qemu cross-arch lanes and the Windows lane); the deterministic, ruby-free tests
alone hold coverage at 100% there.

## Tests & coverage

```sh
go test -race -cover ./...
```

100% statement coverage is enforced in CI on every supported OS.

## License

BSD-3-Clause. Copyright (c) 2026, the go-ruby-pathname/pathname authors.

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```

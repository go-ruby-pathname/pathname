// Copyright (c) the go-ruby-pathname/pathname authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package pathname is a pure-Go (no cgo) reimplementation of the pure
// path-manipulation surface of Ruby's pathname standard library, as embedded in
// go-embedded-ruby's prelude and matching MRI 4.0.5.
//
// A [Pathname] wraps a path string and exposes the lexical operations Ruby's
// Pathname offers — basename, dirname, extname, cleanpath, relative_path_from,
// join/+, split, each_filename, ascend/descend, sub_ext, sub, comparison — none
// of which touch the filesystem. The filesystem-touching delegations of Ruby's
// Pathname (read, write, exist?, …) are deliberately out of scope: in
// go-embedded-ruby they remain host-side, forwarding to the File class, while
// this library is the standalone, interpreter-independent path algebra.
//
// Ruby's Pathname is "/"-based for these lexical operations on every platform
// (it never consults the OS path separator for join/split/cleanpath), so this
// package uses "/" unconditionally. The behaviour is therefore identical on
// Windows, Linux and macOS; the package builds and runs the same on GOOS=windows.
package pathname

import (
	"fmt"
	"strings"
)

// Separator is the path component separator. Ruby's Pathname hardcodes "/" for
// the lexical operations on every platform, and so does this package.
const Separator = "/"

// Pathname wraps a path string and exposes Ruby's pure path-manipulation
// methods. The zero value is a Pathname for the empty string. It is immutable:
// every transforming method returns a fresh *Pathname.
type Pathname struct {
	path string
}

// New returns a Pathname for the given path string, like Ruby's Pathname.new.
func New(path string) *Pathname {
	return &Pathname{path: path}
}

// ToS returns the wrapped path string (Ruby's #to_s / #to_path / #to_str).
func (p *Pathname) ToS() string {
	return p.path
}

// String implements fmt.Stringer, returning the same value as ToS so a
// *Pathname formats as its path. (Ruby's #to_s.)
func (p *Pathname) String() string {
	return p.path
}

// Inspect mirrors Ruby's Pathname#inspect: "#<Pathname:PATH>".
func (p *Pathname) Inspect() string {
	return "#<Pathname:" + p.path + ">"
}

// Absolute reports whether the path is absolute (begins with "/"), like Ruby's
// Pathname#absolute?.
func (p *Pathname) Absolute() bool {
	return strings.HasPrefix(p.path, Separator)
}

// Relative reports whether the path is relative, like Ruby's Pathname#relative?.
func (p *Pathname) Relative() bool {
	return !p.Absolute()
}

// Root reports whether the path is a root, i.e. one or more "/" and nothing
// else, like Ruby's Pathname#root?.
func (p *Pathname) Root() bool {
	if p.path == "" {
		return false
	}
	for i := 0; i < len(p.path); i++ {
		if p.path[i] != '/' {
			return false
		}
	}
	return true
}

// plus implements the +/join append rule (Ruby's Pathname.__plus): an absolute
// component resets to the root; otherwise components join with a single "/".
func plus(base, rel string) string {
	if strings.HasPrefix(rel, Separator) { // absolute resets to root
		return rel
	}
	if base == "" {
		return rel
	}
	if rel == "" || rel == "." {
		return base
	}
	if strings.HasSuffix(base, Separator) {
		return base + rel
	}
	return base + Separator + rel
}

// Plus appends a path component, matching Ruby's Pathname#+ (and #/): an
// absolute argument resets to the root, otherwise a single "/" separates them.
func (p *Pathname) Plus(other *Pathname) *Pathname {
	return New(plus(p.path, other.path))
}

// PlusString is Plus accepting a raw string component, matching Ruby's Pathname#+
// when handed a String (it wraps it in a Pathname first).
func (p *Pathname) PlusString(other string) *Pathname {
	return New(plus(p.path, other))
}

// Join appends one or more components left to right, matching Ruby's
// Pathname#join: each component follows the +/join append rule, so an absolute
// component resets the accumulated path to the root.
func (p *Pathname) Join(args ...*Pathname) *Pathname {
	result := p
	for _, a := range args {
		result = result.Plus(a)
	}
	return result
}

// JoinStrings is Join accepting raw string components.
func (p *Pathname) JoinStrings(args ...string) *Pathname {
	result := p
	for _, a := range args {
		result = result.PlusString(a)
	}
	return result
}

// nonEmptyParts splits the path on "/" and drops empty components (leading,
// trailing and doubled separators), matching Ruby's
// split(SEPARATOR).reject(&:empty?).
func nonEmptyParts(s string) []string {
	out := make([]string, 0, strings.Count(s, Separator)+1)
	for _, part := range strings.Split(s, Separator) {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// Basename returns the last path component (Ruby's Pathname#basename with an
// empty suffix). For an all-separator or empty path it returns "/" as MRI does
// for a path whose only non-empty component is the root.
func (p *Pathname) Basename() *Pathname {
	return p.BasenameSuffix("")
}

// BasenameSuffix returns the last path component with an optional trailing
// suffix stripped, matching Ruby's Pathname#basename(suffix). A suffix of ".*"
// strips any file extension; any other non-empty suffix is removed only when the
// component ends with it.
func (p *Pathname) BasenameSuffix(suffix string) *Pathname {
	parts := nonEmptyParts(p.path)
	var base string
	if len(parts) == 0 {
		base = Separator
	} else {
		base = parts[len(parts)-1]
	}
	if suffix == ".*" {
		e := fileExtname(base)
		if e != "" {
			base = base[:len(base)-len(e)]
		}
	} else if suffix != "" && strings.HasSuffix(base, suffix) {
		base = base[:len(base)-len(suffix)]
	}
	return New(base)
}

// Dirname returns all but the last path component, matching Ruby's
// Pathname#dirname (and its alias #parent): "." when there is no separator and
// "/" when the only separator is the leading one.
func (p *Pathname) Dirname() *Pathname {
	idx := strings.LastIndex(p.path, Separator)
	if idx < 0 {
		return New(".")
	}
	if idx == 0 {
		return New(Separator)
	}
	return New(p.path[:idx])
}

// Parent is an alias for Dirname, matching Ruby's Pathname#parent.
func (p *Pathname) Parent() *Pathname {
	return p.Dirname()
}

// fileExtname extracts a file extension the way MRI's File.extname (and thus
// Pathname#extname) does, matching CRuby's ruby_enc_find_extname: a leading run
// of dots never starts an extension (so dotfiles like ".foo" and "..a" have no
// extension), and beyond that run the extension begins at the last "." — a
// trailing run of dots therefore collapses to a single "." ("foo." and "foo.."
// both yield "."), while "a..b" yields ".b".
//
// (Note: this is byte-for-byte MRI, which differs from the simplified rule in
// go-embedded-ruby's prelude that returns "" for a trailing dot; the prelude is
// the binding surface, MRI is the oracle, and extname is one of the fiddly
// methods the spec requires matching against MRI.)
func fileExtname(name string) string {
	// Skip the leading run of dots; those never begin an extension.
	start := 0
	for start < len(name) && name[start] == '.' {
		start++
	}
	if start >= len(name) {
		return ""
	}
	// The extension begins at the last "." at or after the leading run.
	e := -1
	for i := start; i < len(name); i++ {
		if name[i] == '.' {
			e = i
		}
	}
	if e < 0 {
		return ""
	}
	return name[e:]
}

// Extname returns the file extension of the last path component (".txt", or ""
// when there is none), matching Ruby's Pathname#extname.
func (p *Pathname) Extname() string {
	return fileExtname(p.Basename().path)
}

// Split returns [dirname, basename], matching Ruby's Pathname#split.
func (p *Pathname) Split() []*Pathname {
	return []*Pathname{p.Dirname(), p.Basename()}
}

// EachFilename calls yield once per non-empty path component (leading, trailing
// and doubled separators contribute no component), matching Ruby's
// Pathname#each_filename.
func (p *Pathname) EachFilename(yield func(string)) {
	for _, f := range nonEmptyParts(p.path) {
		yield(f)
	}
}

// Filenames returns the non-empty path components as a slice, matching the
// result of Ruby's each_filename.to_a.
func (p *Pathname) Filenames() []string {
	return nonEmptyParts(p.path)
}

// ascendPaths returns the path and each parent up to the root (or the first
// relative component), matching the helper Ruby's #ascend/#descend share.
func (p *Pathname) ascendPaths() []string {
	out := []string{p.path}
	cur := p.path
	for {
		idx := strings.LastIndex(cur, Separator)
		if idx < 0 {
			break
		}
		if idx == 0 {
			if cur != Separator {
				out = append(out, Separator)
			}
			break
		}
		cur = cur[:idx]
		out = append(out, cur)
	}
	return out
}

// Ascend yields the path then each parent up to the root (or the first relative
// component), matching Ruby's Pathname#ascend.
func (p *Pathname) Ascend(yield func(*Pathname)) {
	for _, s := range p.ascendPaths() {
		yield(New(s))
	}
}

// Descend yields the shortest prefix first down to the full path — the ascend
// sequence reversed — matching Ruby's Pathname#descend.
func (p *Pathname) Descend(yield func(*Pathname)) {
	paths := p.ascendPaths()
	for i := len(paths) - 1; i >= 0; i-- {
		yield(New(paths[i]))
	}
}

// Cleanpath collapses ".", ".." and redundant separators, matching Ruby's
// Pathname#cleanpath (the default, aggressive form). A ".." that would escape
// the root of an absolute path is dropped (as MRI does); a ".." that escapes a
// relative path is preserved.
func (p *Pathname) Cleanpath() *Pathname {
	abs := p.Absolute()
	var out []string
	for _, part := range strings.Split(p.path, Separator) {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(out) > 0 && out[len(out)-1] != ".." {
				out = out[:len(out)-1]
			} else if !abs {
				out = append(out, part)
			}
			continue
		}
		out = append(out, part)
	}
	cleaned := strings.Join(out, Separator)
	if abs {
		return New(Separator + cleaned)
	}
	if cleaned == "" {
		return New(".")
	}
	return New(cleaned)
}

// SubExt replaces the file extension of the whole path with repl, matching
// Ruby's Pathname#sub_ext. When there is no extension, repl is appended.
func (p *Pathname) SubExt(repl string) *Pathname {
	e := fileExtname(p.path)
	return New(p.path[:len(p.path)-len(e)] + repl)
}

// Sub returns a Pathname whose path has the first occurrence of pattern replaced
// by repl, matching the lexical use of Ruby's Pathname#sub (a plain
// String#sub on the wrapped path, no filesystem access).
func (p *Pathname) Sub(pattern, repl string) *Pathname {
	return New(strings.Replace(p.path, pattern, repl, 1))
}

// ArgumentError mirrors the error Ruby's relative_path_from raises for an
// incompatible base directory. Its message matches MRI's wording.
type ArgumentError struct {
	Message string
}

func (e *ArgumentError) Error() string {
	return e.Message
}

// rubyInspectString renders a Go string the way Ruby's String#inspect does for
// the messages relative_path_from builds. The paths involved are plain ASCII
// path strings, so only the quote and backslash need escaping.
func rubyInspectString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		default:
			b.WriteByte(s[i])
		}
	}
	b.WriteByte('"')
	return b.String()
}

// RelativePathFrom returns this path expressed relative to baseDirectory, using
// only the lexical components (no filesystem access), matching Ruby's
// Pathname#relative_path_from. Both paths are cleaned and split, a shared prefix
// is dropped, each remaining base component contributes a ".." and the remaining
// self components follow. Mixing an absolute path with a relative one — or a
// ".." that escapes a relative base — returns an *ArgumentError, as MRI raises.
func (p *Pathname) RelativePathFrom(baseDirectory *Pathname) (*Pathname, error) {
	dest := p.Cleanpath()
	base := baseDirectory.Cleanpath()
	if dest.Absolute() != base.Absolute() {
		destPrefix := ""
		if dest.Absolute() {
			destPrefix = Separator
		}
		return nil, &ArgumentError{Message: fmt.Sprintf(
			"different prefix: %s and %s",
			rubyInspectString(destPrefix), rubyInspectString(baseDirectory.path))}
	}
	destParts := nonEmptyParts(dest.path)
	baseParts := nonEmptyParts(base.path)
	i := 0
	for i < len(destParts) && i < len(baseParts) && destParts[i] == baseParts[i] {
		i++
	}
	up := baseParts[i:]
	for _, c := range up {
		if c == ".." {
			return nil, &ArgumentError{Message: fmt.Sprintf(
				"base_directory has ..: %s", rubyInspectString(baseDirectory.path))}
		}
	}
	rel := make([]string, 0, len(up)+len(destParts)-i)
	for range up {
		rel = append(rel, "..")
	}
	rel = append(rel, destParts[i:]...)
	if len(rel) == 0 {
		return New("."), nil
	}
	return New(strings.Join(rel, Separator)), nil
}

// Cmp compares two paths lexically by their wrapped strings, matching Ruby's
// Pathname#<=>: -1, 0 or 1.
func (p *Pathname) Cmp(other *Pathname) int {
	return strings.Compare(p.path, other.path)
}

// Eql reports whether two paths are equal, matching Ruby's Pathname#== / #eql?
// (string equality of the wrapped paths).
func (p *Pathname) Eql(other *Pathname) bool {
	return p.path == other.path
}

// Hash returns the FNV-1a hash of the wrapped path. Ruby's Pathname#hash
// delegates to String#hash; this is a stable Go-side hash for use as a map key,
// equal exactly when the paths are equal.
func (p *Pathname) Hash() uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	h := uint64(offset64)
	for i := 0; i < len(p.path); i++ {
		h ^= uint64(p.path[i])
		h *= prime64
	}
	return h
}

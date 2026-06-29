// Copyright (c) the go-ruby-pathname/pathname authors
//
// SPDX-License-Identifier: BSD-3-Clause

package pathname

import (
	"reflect"
	"testing"
)

func TestNewToSStringInspect(t *testing.T) {
	p := New("a/b/c")
	if got := p.ToS(); got != "a/b/c" {
		t.Errorf("ToS = %q", got)
	}
	if got := p.String(); got != "a/b/c" {
		t.Errorf("String = %q", got)
	}
	if got := p.Inspect(); got != "#<Pathname:a/b/c>" {
		t.Errorf("Inspect = %q", got)
	}
}

func TestAbsoluteRelative(t *testing.T) {
	cases := []struct {
		path string
		abs  bool
	}{
		{"/a", true},
		{"/", true},
		{"a", false},
		{"", false},
		{"./a", false},
	}
	for _, c := range cases {
		p := New(c.path)
		if p.Absolute() != c.abs {
			t.Errorf("%q Absolute = %v, want %v", c.path, p.Absolute(), c.abs)
		}
		if p.Relative() != !c.abs {
			t.Errorf("%q Relative = %v, want %v", c.path, p.Relative(), !c.abs)
		}
	}
}

func TestRoot(t *testing.T) {
	cases := []struct {
		path string
		root bool
	}{
		{"/", true},
		{"//", true},
		{"///", true},
		{"/a", false},
		{"a", false},
		{"", false},
		{"a/", false},
	}
	for _, c := range cases {
		if got := New(c.path).Root(); got != c.root {
			t.Errorf("%q Root = %v, want %v", c.path, got, c.root)
		}
	}
}

func TestPlusAndJoin(t *testing.T) {
	cases := []struct {
		base, rel, want string
	}{
		{"a", "b", "a/b"},
		{"a/", "b", "a/b"},
		{"a", "/b", "/b"}, // absolute resets to root
		{"", "b", "b"},
		{"a", "", "a"},
		{"a", ".", "a"},
		{"/", "a", "/a"},
	}
	for _, c := range cases {
		if got := New(c.base).PlusString(c.rel).ToS(); got != c.want {
			t.Errorf("%q + %q = %q, want %q", c.base, c.rel, got, c.want)
		}
		if got := New(c.base).Plus(New(c.rel)).ToS(); got != c.want {
			t.Errorf("%q Plus %q = %q, want %q", c.base, c.rel, got, c.want)
		}
	}

	if got := New("a").JoinStrings("b", "c").ToS(); got != "a/b/c" {
		t.Errorf("JoinStrings = %q", got)
	}
	if got := New("a").Join(New("b"), New("/c"), New("d")).ToS(); got != "/c/d" {
		t.Errorf("Join with absolute reset = %q", got)
	}
	if got := New("x").Join().ToS(); got != "x" {
		t.Errorf("Join() = %q", got)
	}
}

func TestBasename(t *testing.T) {
	cases := []struct {
		path, suffix, want string
	}{
		{"/usr/bin/ruby", "", "ruby"},
		{"/usr/bin/ruby.rb", ".rb", "ruby"},
		{"/usr/bin/ruby.rb", ".*", "ruby"},
		{"/usr/bin/ruby", ".rb", "ruby"}, // suffix not present -> unchanged
		{"foo/", "", "foo"},
		{"/", "", "/"},
		{"", "", "/"},
		{"noext", ".*", "noext"}, // ".*" with no extension -> unchanged
	}
	for _, c := range cases {
		got := New(c.path).BasenameSuffix(c.suffix).ToS()
		if got != c.want {
			t.Errorf("basename(%q, %q) = %q, want %q", c.path, c.suffix, got, c.want)
		}
	}
	if got := New("/a/b").Basename().ToS(); got != "b" {
		t.Errorf("Basename = %q", got)
	}
}

func TestDirnameParent(t *testing.T) {
	cases := []struct {
		path, want string
	}{
		{"/foo/bar", "/foo"},
		{"/foo", "/"},
		{"foo", "."},
		{"foo/bar", "foo"},
	}
	for _, c := range cases {
		if got := New(c.path).Dirname().ToS(); got != c.want {
			t.Errorf("dirname(%q) = %q, want %q", c.path, got, c.want)
		}
		if got := New(c.path).Parent().ToS(); got != c.want {
			t.Errorf("parent(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestExtname(t *testing.T) {
	cases := []struct {
		path, want string
	}{
		{"foo.tar.gz", ".gz"},
		{"foo.txt", ".txt"},
		{"foo", ""},
		{".foo", ""},   // dotfile: leading dot does not start an extension
		{"foo.", "."},  // trailing dot IS the extension, per MRI File.extname
		{"foo..", "."}, // trailing dot run collapses to a single "."
		{"a..b", ".b"}, // dot run before content: extension is the last dot + content
		{"a.b/c", ""},
		{"dir/file.md", ".md"},
	}
	for _, c := range cases {
		if got := New(c.path).Extname(); got != c.want {
			t.Errorf("extname(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestSplit(t *testing.T) {
	parts := New("/a/b/c").Split()
	if len(parts) != 2 || parts[0].ToS() != "/a/b" || parts[1].ToS() != "c" {
		t.Errorf("Split = %v", []string{parts[0].ToS(), parts[1].ToS()})
	}
}

func TestEachFilename(t *testing.T) {
	var got []string
	New("/a//b/c/").EachFilename(func(f string) { got = append(got, f) })
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("EachFilename = %v, want %v", got, want)
	}
	if fns := New("a/b").Filenames(); !reflect.DeepEqual(fns, []string{"a", "b"}) {
		t.Errorf("Filenames = %v", fns)
	}
}

func TestAscendDescend(t *testing.T) {
	var asc []string
	New("/a/b/c").Ascend(func(p *Pathname) { asc = append(asc, p.ToS()) })
	wantAsc := []string{"/a/b/c", "/a/b", "/a", "/"}
	if !reflect.DeepEqual(asc, wantAsc) {
		t.Errorf("Ascend = %v, want %v", asc, wantAsc)
	}

	var desc []string
	New("/a/b/c").Descend(func(p *Pathname) { desc = append(desc, p.ToS()) })
	wantDesc := []string{"/", "/a", "/a/b", "/a/b/c"}
	if !reflect.DeepEqual(desc, wantDesc) {
		t.Errorf("Descend = %v, want %v", desc, wantDesc)
	}

	// Relative path: ascends to the first component, never to "/".
	var rel []string
	New("a/b").Ascend(func(p *Pathname) { rel = append(rel, p.ToS()) })
	if !reflect.DeepEqual(rel, []string{"a/b", "a"}) {
		t.Errorf("relative Ascend = %v", rel)
	}

	// A bare component has no separator: only itself.
	var single []string
	New("a").Ascend(func(p *Pathname) { single = append(single, p.ToS()) })
	if !reflect.DeepEqual(single, []string{"a"}) {
		t.Errorf("single Ascend = %v", single)
	}

	// Root itself: only "/".
	var root []string
	New("/").Ascend(func(p *Pathname) { root = append(root, p.ToS()) })
	if !reflect.DeepEqual(root, []string{"/"}) {
		t.Errorf("root Ascend = %v", root)
	}
}

func TestCleanpath(t *testing.T) {
	cases := []struct {
		path, want string
	}{
		{"a/./b/../c", "a/c"},
		{"a/b/../c", "a/c"},
		{"..", ".."},
		{"../a", "../a"},
		{"/..", "/"},
		{"/../a", "/a"},
		{"a/", "a"},
		{".", "."},
		{"", "."},
		{"/a/../../b", "/b"},
		{"a/../../b", "../b"},
		{"./a/./b", "a/b"},
		{"/a/b", "/a/b"},
		{"a/../..", ".."},
	}
	for _, c := range cases {
		if got := New(c.path).Cleanpath().ToS(); got != c.want {
			t.Errorf("cleanpath(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

// TestFileExtnameDirect exercises fileExtname directly, including the
// dotfile-with-no-further-dot path (leading dots skipped, then no remaining dot).
func TestFileExtnameDirect(t *testing.T) {
	cases := []struct{ name, want string }{
		{".foo", ""},   // leading dot skipped, then "foo": no extension
		{"..a", ""},    // run of leading dots skipped, then "a": no extension
		{"....", ""},   // all dots: nothing after the leading run
		{"a", ""},      // no dot at all
		{"a.b", ".b"},  // ordinary extension
		{"a.", "."},    // trailing dot
		{"a..b", ".b"}, // dot run then content
	}
	for _, c := range cases {
		if got := fileExtname(c.name); got != c.want {
			t.Errorf("fileExtname(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSubExt(t *testing.T) {
	cases := []struct {
		path, repl, want string
	}{
		{"foo", ".rb", "foo.rb"},
		{"foo.txt", ".rb", "foo.rb"},
		{"dir/foo.c", ".o", "dir/foo.o"},
		{"foo.tar.gz", ".bz2", "foo.tar.bz2"},
	}
	for _, c := range cases {
		if got := New(c.path).SubExt(c.repl).ToS(); got != c.want {
			t.Errorf("sub_ext(%q, %q) = %q, want %q", c.path, c.repl, got, c.want)
		}
	}
}

func TestSub(t *testing.T) {
	if got := New("/usr/bin/ruby").Sub("ruby", "perl").ToS(); got != "/usr/bin/perl" {
		t.Errorf("Sub = %q", got)
	}
	if got := New("a/a/a").Sub("a", "b").ToS(); got != "b/a/a" {
		t.Errorf("Sub first-only = %q", got)
	}
}

func TestRelativePathFrom(t *testing.T) {
	cases := []struct {
		dest, base, want string
	}{
		{"/a/b", "/a", "b"},
		{"a/b/c", "a/d", "../b/c"},
		{"/foo/bar", "/foo/bar", "."},
		{"a/b", "a/b", "."},
		{"/a/b/c", "/a/x/y", "../../b/c"},
		{"a", "a/b/c", "../.."},
		{"/", "/", "."},
	}
	for _, c := range cases {
		got, err := New(c.dest).RelativePathFrom(New(c.base))
		if err != nil {
			t.Errorf("relative_path_from(%q, %q) err: %v", c.dest, c.base, err)
			continue
		}
		if got.ToS() != c.want {
			t.Errorf("relative_path_from(%q, %q) = %q, want %q", c.dest, c.base, got.ToS(), c.want)
		}
	}
}

func TestRelativePathFromErrors(t *testing.T) {
	errCases := []struct {
		dest, base, msg string
	}{
		{"/a", "b", `different prefix: "/" and "b"`},
		{"a", "/b", `different prefix: "" and "/b"`},
		{"a", "../b", `base_directory has ..: "../b"`},
		// A path with a quote and a backslash exercises the inspect escaping in
		// the message (matching Ruby's String#inspect for those two bytes).
		{`a"\b`, "/b", `different prefix: "" and "/b"`},
		{"a", `../b"\c`, `base_directory has ..: "../b\"\\c"`},
	}
	for _, c := range errCases {
		_, err := New(c.dest).RelativePathFrom(New(c.base))
		if err == nil {
			t.Errorf("relative_path_from(%q, %q): expected error", c.dest, c.base)
			continue
		}
		ae, ok := err.(*ArgumentError)
		if !ok {
			t.Errorf("error type = %T, want *ArgumentError", err)
			continue
		}
		if ae.Error() != c.msg {
			t.Errorf("error msg = %q, want %q", ae.Error(), c.msg)
		}
	}
}

func TestCmpEqlHash(t *testing.T) {
	a := New("a")
	b := New("b")
	a2 := New("a")
	if a.Cmp(b) != -1 {
		t.Errorf("a <=> b = %d", a.Cmp(b))
	}
	if b.Cmp(a) != 1 {
		t.Errorf("b <=> a = %d", b.Cmp(a))
	}
	if a.Cmp(a2) != 0 {
		t.Errorf("a <=> a = %d", a.Cmp(a2))
	}
	if !a.Eql(a2) {
		t.Error("a should eql a2")
	}
	if a.Eql(b) {
		t.Error("a should not eql b")
	}
	if a.Hash() != a2.Hash() {
		t.Error("equal paths must hash equal")
	}
	if a.Hash() == b.Hash() {
		t.Error("different paths hash equal (collision in test corpus)")
	}
	if New("").Hash() == 0 {
		t.Error("empty hash should be FNV offset, not zero")
	}
}

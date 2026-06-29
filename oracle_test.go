// Copyright (c) the go-ruby-pathname/pathname authors
//
// SPDX-License-Identifier: BSD-3-Clause

package pathname

import (
	"os/exec"
	"strings"
	"testing"
)

// rubyBin locates a usable `ruby` once and confirms it is MRI >= 4.0 (the
// oracle's reference version). The oracle tests skip themselves when ruby is
// absent (the qemu cross-arch lanes and the Windows lane) or too old, so the
// deterministic suite alone drives the 100% gate there.
func rubyBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping MRI oracle")
	}
	out, err := exec.Command(path, "-e", `print(RUBY_VERSION >= "4.0")`).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		t.Skipf("ruby too old or unusable (RUBY_VERSION >= 4.0 required); skipping MRI oracle")
	}
	return path
}

// rubyEval runs a Ruby script under `ruby -rpathname` and returns its stdout.
// The preamble binmodes both stdin and stdout so Windows text-mode does not
// translate the bytes (the go-ruby-erb lesson), and every value is printed with
// a trailing newline we trim. The paths in every script are lexical "/"-based
// literals — no OS path is embedded — so the oracle is identical on every OS.
func rubyEval(t *testing.T, bin, script string) string {
	t.Helper()
	pre := "$stdout.binmode\n$stdin.binmode\n"
	cmd := exec.Command(bin, "-rpathname", "-e", pre+script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return strings.TrimRight(string(out), "\n")
}

// TestOracleCleanpath checks Cleanpath against MRI's Pathname#cleanpath over the
// fiddly corpus (trailing slash, "." / ".." collapsing, ".." past the root).
func TestOracleCleanpath(t *testing.T) {
	bin := rubyBin(t)
	paths := []string{
		"a/./b/../c", "a/b/../c", "..", "../a", "/..", "/../a", "a/",
		".", "", "/a/../../b", "a/../../b", "./a/./b", "/a/b", "a/../..",
		"foo/bar/", "//a//b//", "/", "x/y/z",
	}
	for _, path := range paths {
		want := rubyEval(t, bin, "print Pathname.new("+rb(path)+").cleanpath.to_s")
		if got := New(path).Cleanpath().ToS(); got != want {
			t.Errorf("cleanpath(%q) = %q, MRI = %q", path, got, want)
		}
	}
}

// TestOracleBasenameExtname checks Basename (with and without suffix) and Extname
// against MRI.
func TestOracleBasenameExtname(t *testing.T) {
	bin := rubyBin(t)
	type bc struct{ path, suffix string }
	cases := []bc{
		{"/usr/bin/ruby", ""}, {"/usr/bin/ruby.rb", ".rb"}, {"/usr/bin/ruby.rb", ".*"},
		{"/usr/bin/ruby", ".rb"}, {"foo/", ""}, {"/", ""}, {"noext", ".*"},
		{"a/b/c.tar.gz", ".*"}, {"file.TXT", ".txt"},
	}
	for _, c := range cases {
		want := rubyEval(t, bin, "print Pathname.new("+rb(c.path)+").basename("+rb(c.suffix)+").to_s")
		if got := New(c.path).BasenameSuffix(c.suffix).ToS(); got != want {
			t.Errorf("basename(%q,%q) = %q, MRI = %q", c.path, c.suffix, got, want)
		}
	}

	extPaths := []string{"foo.tar.gz", "foo.txt", "foo", ".foo", "foo.", "a.b/c", "dir/file.md", "/x/y.JSON"}
	for _, path := range extPaths {
		want := rubyEval(t, bin, "print Pathname.new("+rb(path)+").extname")
		if got := New(path).Extname(); got != want {
			t.Errorf("extname(%q) = %q, MRI = %q", path, got, want)
		}
	}
}

// TestOracleDirname checks Dirname against MRI.
func TestOracleDirname(t *testing.T) {
	bin := rubyBin(t)
	for _, path := range []string{"/foo/bar", "/foo", "foo", "foo/bar", "/", "a/b/c"} {
		want := rubyEval(t, bin, "print Pathname.new("+rb(path)+").dirname.to_s")
		if got := New(path).Dirname().ToS(); got != want {
			t.Errorf("dirname(%q) = %q, MRI = %q", path, got, want)
		}
	}
}

// TestOraclePlusJoin checks Plus and Join against MRI's #+ and #join.
func TestOraclePlusJoin(t *testing.T) {
	bin := rubyBin(t)
	type pc struct{ base, rel string }
	for _, c := range []pc{{"a", "b"}, {"a/", "b"}, {"a", "/b"}, {"/", "a"}, {"a", "."}, {"foo/bar", "baz"}} {
		want := rubyEval(t, bin, "print (Pathname.new("+rb(c.base)+") + "+rb(c.rel)+").to_s")
		if got := New(c.base).PlusString(c.rel).ToS(); got != want {
			t.Errorf("%q + %q = %q, MRI = %q", c.base, c.rel, got, want)
		}
	}
	want := rubyEval(t, bin, `print Pathname.new("a").join("b", "/c", "d").to_s`)
	if got := New("a").JoinStrings("b", "/c", "d").ToS(); got != want {
		t.Errorf("join = %q, MRI = %q", got, want)
	}
}

// TestOracleRelativePathFrom checks RelativePathFrom and its ArgumentError
// messages against MRI.
func TestOracleRelativePathFrom(t *testing.T) {
	bin := rubyBin(t)
	type rc struct{ dest, base string }
	ok := []rc{
		{"/a/b", "/a"}, {"a/b/c", "a/d"}, {"/foo/bar", "/foo/bar"},
		{"/a/b/c", "/a/x/y"}, {"a", "a/b/c"}, {"/", "/"}, {"a/b", "a/b"},
	}
	for _, c := range ok {
		want := rubyEval(t, bin, "print Pathname.new("+rb(c.dest)+").relative_path_from("+rb(c.base)+").to_s")
		got, err := New(c.dest).RelativePathFrom(New(c.base))
		if err != nil {
			t.Errorf("relative_path_from(%q,%q) err: %v (MRI=%q)", c.dest, c.base, err, want)
			continue
		}
		if got.ToS() != want {
			t.Errorf("relative_path_from(%q,%q) = %q, MRI = %q", c.dest, c.base, got.ToS(), want)
		}
	}

	bad := []rc{{"/a", "b"}, {"a", "/b"}, {"a", "../b"}}
	for _, c := range bad {
		script := "begin; Pathname.new(" + rb(c.dest) + ").relative_path_from(" + rb(c.base) +
			"); rescue ArgumentError => e; print e.message; end"
		want := rubyEval(t, bin, script)
		_, err := New(c.dest).RelativePathFrom(New(c.base))
		if err == nil {
			t.Errorf("relative_path_from(%q,%q): expected error, MRI msg %q", c.dest, c.base, want)
			continue
		}
		if err.Error() != want {
			t.Errorf("relative_path_from(%q,%q) err = %q, MRI = %q", c.dest, c.base, err.Error(), want)
		}
	}
}

// TestOracleSubExtAscendRoot checks SubExt, the ascend sequence, root? and the
// comparison operator against MRI.
func TestOracleSubExtAscendRoot(t *testing.T) {
	bin := rubyBin(t)
	type sc struct{ path, repl string }
	for _, c := range []sc{{"foo", ".rb"}, {"foo.txt", ".rb"}, {"dir/foo.c", ".o"}, {"foo.tar.gz", ".bz2"}} {
		want := rubyEval(t, bin, "print Pathname.new("+rb(c.path)+").sub_ext("+rb(c.repl)+").to_s")
		if got := New(c.path).SubExt(c.repl).ToS(); got != want {
			t.Errorf("sub_ext(%q,%q) = %q, MRI = %q", c.path, c.repl, got, want)
		}
	}

	for _, path := range []string{"/a/b/c", "a/b", "/", "a", "/x"} {
		want := rubyEval(t, bin, "print Pathname.new("+rb(path)+").ascend.map(&:to_s).join(\"|\")")
		var got []string
		New(path).Ascend(func(p *Pathname) { got = append(got, p.ToS()) })
		if strings.Join(got, "|") != want {
			t.Errorf("ascend(%q) = %q, MRI = %q", path, strings.Join(got, "|"), want)
		}
	}

	for _, path := range []string{"/", "//", "///", "/a", "a", "", "a/"} {
		want := rubyEval(t, bin, "print Pathname.new("+rb(path)+").root?")
		if got := New(path).Root(); boolStr(got) != want {
			t.Errorf("root?(%q) = %v, MRI = %q", path, got, want)
		}
	}

	type cc struct{ a, b string }
	for _, c := range []cc{{"a", "b"}, {"b", "a"}, {"a", "a"}, {"abc", "abd"}} {
		want := rubyEval(t, bin, "print (Pathname.new("+rb(c.a)+") <=> Pathname.new("+rb(c.b)+"))")
		if got := New(c.a).Cmp(New(c.b)); intStr(got) != want {
			t.Errorf("(%q <=> %q) = %d, MRI = %q", c.a, c.b, got, want)
		}
	}
}

// TestOracleEachFilename checks the component split against MRI's each_filename.
func TestOracleEachFilename(t *testing.T) {
	bin := rubyBin(t)
	for _, path := range []string{"/a//b/c/", "a/b/c", "/", "", "a", "//x//"} {
		want := rubyEval(t, bin, "print Pathname.new("+rb(path)+").each_filename.to_a.join(\"|\")")
		got := strings.Join(New(path).Filenames(), "|")
		if got != want {
			t.Errorf("each_filename(%q) = %q, MRI = %q", path, got, want)
		}
	}
}

// rb renders a Go string as a Ruby double-quoted string literal for embedding in
// an oracle script. The corpus is plain ASCII path text, so only the quote and
// backslash need escaping.
func rb(s string) string {
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

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func intStr(i int) string {
	switch {
	case i < 0:
		return "-1"
	case i > 0:
		return "1"
	default:
		return "0"
	}
}

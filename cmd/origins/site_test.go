package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

var update = flag.Bool("update", false, "rewrite the site's generated pages")

// site is the site's source, from this package's directory.
var site = filepath.Join("..", "..", "site")

// subcommand is the word after the command in a usage line, where it names one.
var subcommand = regexp.MustCompile(`^origins [a-z]+ ([a-z]+) `)

// commandsPage is the site's command reference: what origins help prints, then each
// command's help, then the flags of each of its subcommands.
func commandsPage(t *testing.T) string {
	t.Helper()
	var page strings.Builder
	page.WriteString("<!-- Written by go test ./cmd/origins -run TestTheSitesGeneratedPages -update from cmd/origins/help.go. -->\n\n")
	page.WriteString("# Commands\n\n")
	page.WriteString("This page shows the help that Origins prints for each command. You can see the\n")
	page.WriteString("same text in a terminal with `origins help <command>` or `origins <command> -h`.\n\n")
	page.WriteString("## origins help\n\n")
	fence(t, &page, "help")
	for _, group := range helps {
		for _, one := range group {
			first, size := utf8.DecodeRuneInString(one.summary)
			fmt.Fprintf(&page, "## origins %s\n\n%c%s.\n\n", one.name, unicode.ToUpper(first), one.summary[size:])
			fence(t, &page, one.name, "-h")
			for _, line := range one.usage {
				sub := subcommand.FindStringSubmatch(line)
				if sub == nil {
					continue
				}
				fmt.Fprintf(&page, "### origins %s %s\n\n", one.name, sub[1])
				_, flags, found := strings.Cut(run1(t, one.name, sub[1], "-h"), "\nFlags:\n")
				if !found {
					t.Fatalf("origins %s %s -h lists no flags", one.name, sub[1])
				}
				fmt.Fprintf(&page, "```text\nFlags:\n%s```\n\n", flags)
			}
		}
	}
	return strings.TrimSuffix(page.String(), "\n")
}

func fence(t *testing.T, page *strings.Builder, args ...string) {
	t.Helper()
	fmt.Fprintf(page, "```text\n%s```\n\n", run1(t, args...))
}

func run1(t *testing.T, args ...string) string {
	t.Helper()
	code, stdout, stderr := invoke(t, args...)
	if code != exitOK {
		t.Fatalf("origins %s exited %d: %s", strings.Join(args, " "), code, stderr)
	}
	return stdout
}

// schemaPages are the JSON Schema of each kind as files the site serves, and the page
// that lists them.
func schemaPages(t *testing.T) map[string]string {
	t.Helper()
	var all map[string]struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(run1(t, "schema")), &all); err != nil {
		t.Fatal(err)
	}
	kinds := []string{"person", "family", "place", "source", "repository", "note", "hypothesis", "project", "common"}
	if len(kinds) != len(all) {
		t.Fatalf("origins schema gives %d schemas, and the page lists %d", len(all), len(kinds))
	}
	files := map[string]string{}
	var page strings.Builder
	page.WriteString("<!-- Written by go test ./cmd/origins -run TestTheSitesGeneratedPages -update from origins schema. -->\n\n")
	page.WriteString("# JSON Schema\n\n")
	page.WriteString("`origins schema <kind>` prints the JSON Schema of one kind of record, and the\n")
	page.WriteString("files below are the same schemas. An editor that reads JSON Schema can use them to\n")
	page.WriteString("complete and check the frontmatter of a record. Each schema refers to\n")
	page.WriteString("`common.json` for the values that every kind shares, so keep the files together.\n\n")
	page.WriteString("Origins doesn't read a corpus through these schemas, and `origins check` doesn't\n")
	page.WriteString("use them. A file that the schema accepts can still be refused by Origins, for\n")
	page.WriteString("example when a reference names a record that doesn't exist.\n\n")
	page.WriteString("| Kind | Schema | Description |\n|---|---|---|\n")
	for _, kind := range kinds {
		one, ok := all[kind]
		if !ok {
			t.Fatalf("origins schema gives no %s", kind)
		}
		name := kind
		if kind == "project" {
			name = "project (`origins.yaml`)"
		}
		fmt.Fprintf(&page, "| %s | [%s.json](/schema/%s.json){target=\"_self\"} | %s |\n", name, kind, kind, one.Description)
		files[filepath.Join("public", "schema", kind+".json")] = run1(t, "schema", kind)
	}
	files[filepath.Join("reference", "schema.md")] = page.String()
	return files
}

func TestTheSitesGeneratedPages(t *testing.T) {
	t.Parallel()
	files := schemaPages(t)
	files[filepath.Join("reference", "commands.md")] = commandsPage(t)
	if *update {
		for path, content := range files {
			full := filepath.Join(site, path)
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	for path, content := range files {
		committed, err := os.ReadFile(filepath.Join(site, path))
		if err != nil || string(committed) != content {
			t.Errorf("site/%s is not what generation writes; run go test ./cmd/origins -run TestTheSitesGeneratedPages -update", filepath.ToSlash(path))
		}
	}
	served, err := filepath.Glob(filepath.Join(site, "public", "schema", "*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range served {
		rel, _ := filepath.Rel(site, path)
		if _, ok := files[rel]; !ok {
			t.Errorf("site/%s is no schema origins gives", filepath.ToSlash(rel))
		}
	}
}

// A block is a fenced block of a page: its info string, its lines, and the lines of
// the page its fences are on.
type block struct {
	info       string
	lines      []string
	open, shut int
}

func blocks(lines []string) []block {
	var found []block
	for i := 0; i < len(lines); i++ {
		info, ok := strings.CutPrefix(lines[i], "```")
		if !ok {
			continue
		}
		one := block{info: strings.TrimSpace(info), open: i}
		for i++; i < len(lines) && lines[i] != "```"; i++ {
			one.lines = append(one.lines, lines[i])
		}
		one.shut = i
		found = append(found, one)
	}
	return found
}

// words splits a command as the shell would, for the quoting the site's commands use.
func words(t *testing.T, line string) []string {
	t.Helper()
	var out []string
	var word strings.Builder
	var quote rune
	started := false
	for _, r := range line {
		switch {
		case quote != 0 && r == quote:
			quote = 0
		case quote != 0:
			word.WriteRune(r)
		case r == '\'' || r == '"':
			quote, started = r, true
		case r == ' ':
			if started {
				out = append(out, word.String())
				word.Reset()
				started = false
			}
		default:
			word.WriteRune(r)
			started = true
		}
	}
	if quote != 0 {
		t.Fatalf("unclosed quote in %q", line)
	}
	if started {
		out = append(out, word.String())
	}
	return out
}

// console runs each command of a console block in the working directory, and gives the
// block again with what each printed.
func console(t *testing.T, lines []string) []string {
	t.Helper()
	var out []string
	for i := 0; i < len(lines); {
		line, ok := strings.CutPrefix(lines[i], "$ ")
		if !ok {
			t.Fatalf("console block line %q follows no command", lines[i])
		}
		var shown []string
		for i++; i < len(lines) && !strings.HasPrefix(lines[i], "$ "); i++ {
			shown = append(shown, lines[i])
		}
		var printed string
		switch args := words(t, line); args[0] {
		case "cd":
			t.Chdir(args[1])
		case "cat":
			file, err := os.ReadFile(args[1])
			if err != nil {
				t.Fatal(err)
			}
			printed = string(file)
		case "origins":
			_, stdout, stderr := invoke(t, args[1:]...)
			printed = stdout + stderr
		default:
			t.Fatalf("the site runs no %s", args[0])
		}
		var got []string
		if printed != "" {
			got = strings.Split(strings.TrimSuffix(printed, "\n"), "\n")
		}
		if !*update && strings.Join(got, "\n") != strings.Join(shown, "\n") {
			t.Errorf("$ %s\nprinted:\n%s\nthe page shows:\n%s", line, strings.Join(got, "\n"), strings.Join(shown, "\n"))
		}
		out = append(append(out, "$ "+line), got...)
	}
	return out
}

// The site's Getting started is run as a script: its commands in an empty directory,
// with each file it has the reader write written where it says, and what each command
// prints held to what the page shows. With -update the page is given what they printed.
func TestGettingStartedRunsAsShown(t *testing.T) {
	path, err := filepath.Abs(filepath.Join(site, "getting-started.md"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	// one clock for every commit, so that the hashes the page shows are the ones made
	t.Setenv("GIT_AUTHOR_DATE", "@1792400400 +0000")
	t.Setenv("GIT_COMMITTER_DATE", "@1792400400 +0000")

	lines := strings.Split(string(page), "\n")
	var out []string
	next := 0
	for _, one := range blocks(lines) {
		out = append(out, lines[next:one.open+1]...)
		switch kind, file, _ := strings.Cut(one.info, " "); kind {
		case "console":
			out = append(out, console(t, one.lines)...)
		case "corpus":
			if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte(strings.Join(one.lines, "\n")+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			out = append(out, one.lines...)
		default:
			out = append(out, one.lines...)
		}
		next = one.shut
	}
	out = append(out, lines[next:]...)
	if *update {
		if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// Every block of the site marked corpus is a file of a corpus, and the examples are the
// fixture's records: each page's blocks are written into a copy of the fixture, which
// must still be read, canonical, and contradict itself nowhere. Getting started makes a
// corpus of its own, which its own test runs.
func TestTheSitesCorpusBlocksAreSound(t *testing.T) {
	t.Parallel()
	fixture := theFixture(t)
	err := filepath.WalkDir(site, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == "node_modules" || strings.HasPrefix(entry.Name(), ".")) {
			return filepath.SkipDir
		}
		if entry.IsDir() || filepath.Ext(path) != ".md" || entry.Name() == "getting-started.md" {
			return nil
		}
		page, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files := map[string]string{}
		for _, one := range blocks(strings.Split(string(page), "\n")) {
			if file, ok := strings.CutPrefix(one.info, "corpus "); ok {
				files[file] = strings.Join(one.lines, "\n") + "\n"
			}
		}
		if len(files) == 0 {
			return nil
		}
		name, _ := filepath.Rel(site, path)
		t.Run(filepath.ToSlash(name), func(t *testing.T) {
			t.Parallel()
			root := filepath.Join(t.TempDir(), "corpus")
			if err := os.CopyFS(root, os.DirFS(fixture)); err != nil {
				t.Fatal(err)
			}
			for file, content := range files {
				full := filepath.Join(root, filepath.FromSlash(file))
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if code, stdout, stderr := invoke(t, "fmt", "--list", root); code != exitOK {
				t.Errorf("not canonical, or not read:\n%s%s", stdout, stderr)
			}
			if code, stdout, stderr := invoke(t, "check", root); code != exitOK {
				t.Errorf("check exited %d:\n%s%s", code, grepErrors(stdout), stderr)
			}
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// grepErrors is the errors of check's report, without the warnings the fixture has.
func grepErrors(report string) string {
	var out strings.Builder
	for line := range strings.Lines(report) {
		if !strings.HasPrefix(line, "warning: ") {
			out.WriteString(line)
		}
	}
	return out.String()
}

// reported is every rule a file of the check package reports, by the severity it
// reports it with.
var reported = regexp.MustCompile(`r\.(fail|warn)\("([a-z-]+)"`)

// ruleNames are the rules a row of the findings page names in its first cell.
var ruleNames = regexp.MustCompile("^[|] (`[a-z-]+`(?:, `[a-z-]+`)*) [|]")

// The findings page names every rule check and todo report, each under the heading its
// severity gives it, and no rule that is not reported.
func TestTheSitesFindingsAreTheRules(t *testing.T) {
	t.Parallel()
	sources, err := filepath.Glob(filepath.Join("..", "..", "internal", "check", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	severity, research := map[string]string{}, map[string]bool{}
	for _, file := range sources {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		code, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, found := range reported.FindAllStringSubmatch(string(code), -1) {
			severity[found[2]] = found[1]
			if filepath.Base(file) == "research.go" {
				research[found[2]] = true
			}
		}
	}

	page, err := os.ReadFile(filepath.Join(site, "reference", "check.md"))
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]map[string]bool{"Errors": {}, "Warnings": {}, "Research": {}}
	section := ""
	for line := range strings.Lines(string(page)) {
		if heading, ok := strings.CutPrefix(strings.TrimSpace(line), "## "); ok {
			section = heading
			continue
		}
		cell := ruleNames.FindStringSubmatch(line)
		if cell == nil || listed[section] == nil {
			continue
		}
		for name := range strings.SplitSeq(cell[1], ", ") {
			listed[section][strings.Trim(name, "`")] = true
		}
	}

	for name := range listed["Errors"] {
		if severity[name] != "fail" {
			t.Errorf("%s is listed among the errors, and check reports no such error", name)
		}
	}
	for _, heading := range []string{"Warnings", "Research"} {
		for name := range listed[heading] {
			if severity[name] != "warn" {
				t.Errorf("%s is listed under %s, and check reports no such warning", name, heading)
			}
		}
	}
	for name, how := range severity {
		switch {
		case research[name] && !listed["Research"][name]:
			t.Errorf("%s is research that todo lists, and the page does not list it under Research", name)
		case !research[name] && how == "fail" && !listed["Errors"][name]:
			t.Errorf("%s is an error that check reports, and the page does not list it", name)
		case !research[name] && how == "warn" && !listed["Warnings"][name]:
			t.Errorf("%s is a warning that check reports, and the page does not list it", name)
		}
	}
}

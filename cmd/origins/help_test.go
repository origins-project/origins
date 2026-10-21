package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// dispatched is every command run answers to, read from the source: the cases of
// dispatch's switch and the keys of changes.
func dispatched(t *testing.T) []string {
	t.Helper()
	files := token.NewFileSet()
	var commands []string
	for _, name := range []string{"main.go", "change.go"} {
		file, err := parser.ParseFile(files, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.FuncDecl:
				return node.Name.Name == "dispatch"
			case *ast.GenDecl:
				for _, spec := range node.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok || value.Names[0].Name != "changes" {
						continue
					}
					for _, element := range value.Values[0].(*ast.CompositeLit).Elts {
						commands = append(commands, literal(t, element.(*ast.KeyValueExpr).Key))
					}
				}
				return false
			case *ast.SwitchStmt:
				if ident, ok := node.Tag.(*ast.Ident); !ok || ident.Name != "command" {
					return true
				}
				for _, clause := range node.Body.List {
					for _, expr := range clause.(*ast.CaseClause).List {
						if command := literal(t, expr); !strings.HasPrefix(command, "-") {
							commands = append(commands, command)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(commands)
	return commands
}

func literal(t *testing.T, expr ast.Expr) string {
	t.Helper()
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		t.Fatalf("%T is not a string literal", expr)
	}
	text, err := strconv.Unquote(lit.Value)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func TestEveryCommandHasHelpAndEveryHelpACommand(t *testing.T) {
	t.Parallel()
	var helped []string
	for _, group := range helps {
		for _, one := range group {
			helped = append(helped, one.name)
		}
	}
	slices.Sort(helped)
	commands := dispatched(t)
	if len(commands) < 30 {
		t.Fatalf("read only %v out of dispatch", commands)
	}
	for _, command := range commands {
		if _, found := slices.BinarySearch(helped, command); !found {
			t.Errorf("origins %s has no help", command)
		}
	}
	for _, name := range helped {
		if _, found := slices.BinarySearch(commands, name); !found {
			t.Errorf("there is help for %s, which run does not dispatch", name)
		}
	}
}

func TestHelpAndDashHAnswerAlike(t *testing.T) {
	t.Parallel()
	for _, group := range helps {
		for _, one := range group {
			code, asked, stderr := invoke(t, one.name, "-h")
			if code != exitOK || !strings.HasPrefix(asked, "usage: "+one.usage[0]+"\n") {
				t.Errorf("origins %s -h exited %d\nstdout: %s\nstderr: %s", one.name, code, asked, stderr)
				continue
			}
			if !strings.Contains(asked, "\nExample:\n  "+one.example+"\n") {
				t.Errorf("origins %s -h gives no example:\n%s", one.name, asked)
			}
			if code, helped, _ := invoke(t, "help", one.name); code != exitOK || helped != asked {
				t.Errorf("origins help %s exited %d and differs from -h:\n%s", one.name, code, helped)
			}
		}
	}
}

func TestASubcommandListsItsOwnFlags(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"hypothesis", "accept", "-h"},
		{"hypothesis", "propose", "--help"},
		{"remote", "add", "-h"},
		{"import", "gramps", "-h"},
		{"export", "chart", "-h"},
	} {
		code, stdout, _ := invoke(t, args...)
		if code != exitOK || !strings.HasPrefix(stdout, "usage: origins "+args[0]) ||
			!strings.Contains(stdout, "\nFlags:\n") {
			t.Errorf("%v exited %d:\n%s", args, code, stdout)
		}
	}
	if _, stdout, _ := invoke(t, "import", "gramps", "-h"); !strings.Contains(stdout, "--media directory") {
		t.Errorf("import gramps -h does not list --media:\n%s", stdout)
	}
}

func TestHelpRefusesWhatIsNoCommand(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"help", "fly"}, {"help", "check", "load"}} {
		if code, _, _ := invoke(t, args...); code != exitMisuse {
			t.Errorf("%v exited %d, want %d", args, code, exitMisuse)
		}
	}
	code, _, stderr := invoke(t, "check", "--bogus")
	if code != exitMisuse || !strings.Contains(stderr, "usage: origins check") {
		t.Errorf("check --bogus exited %d:\n%s", code, stderr)
	}
}

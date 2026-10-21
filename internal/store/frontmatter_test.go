package store

import "testing"

func TestSplit(t *testing.T) {
	cases := map[string]struct {
		in          string
		frontmatter string
		body        string
	}{
		"ordinary": {
			in:          "---\nname: Giovanni\n---\n\nHe appears as \"Gio. Rossi\".\n",
			frontmatter: "name: Giovanni\n",
			body:        "He appears as \"Gio. Rossi\".",
		},
		"no body": {
			in:          "---\nname: Firenze\n---\n",
			frontmatter: "name: Firenze\n",
			body:        "",
		},
		"no body and no trailing newline": {
			in:          "---\nname: Firenze\n---",
			frontmatter: "name: Firenze\n",
			body:        "",
		},
		"empty frontmatter": {
			in:          "---\n---\nbody\n",
			frontmatter: "",
			body:        "body",
		},
		"carriage returns are normalised": {
			in:          "---\r\nname: Giovanni\r\n---\r\n\r\nBody\r\n",
			frontmatter: "name: Giovanni\n",
			body:        "Body",
		},
		"a byte order mark is tolerated": {
			in:          "\uFEFF---\nname: Giovanni\n---\nBody\n",
			frontmatter: "name: Giovanni\n",
			body:        "Body",
		},
		"a thematic break in the body is not a fence": {
			in:          "---\nname: Giovanni\n---\n\nFirst\n\n---\n\nSecond\n",
			frontmatter: "name: Giovanni\n",
			body:        "First\n\n---\n\nSecond",
		},
		"blank lines around the body are trimmed": {
			in:          "---\nname: Giovanni\n---\n\n\n\nBody\n\n\n",
			frontmatter: "name: Giovanni\n",
			body:        "Body",
		},
	}
	for what, c := range cases {
		t.Run(what, func(t *testing.T) {
			frontmatter, body, err := Split([]byte(c.in))
			if err != nil {
				t.Fatalf("Split: %v", err)
			}
			if string(frontmatter) != c.frontmatter {
				t.Errorf("frontmatter = %q, want %q", frontmatter, c.frontmatter)
			}
			if string(body) != c.body {
				t.Errorf("body = %q, want %q", body, c.body)
			}
		})
	}
}

func TestSplitRejects(t *testing.T) {
	for what, in := range map[string]string{
		"empty":               "",
		"no fence":            "name: Giovanni\n",
		"unclosed":            "---\nname: Giovanni\n",
		"nothing but a fence": "---\n",
		"fence not first":     "\nname: Giovanni\n---\n",
		"indented fence":      "  ---\nname: Giovanni\n---\n",
	} {
		t.Run(what, func(t *testing.T) {
			if _, _, err := Split([]byte(in)); err == nil {
				t.Errorf("Split(%q) should have failed", in)
			}
		})
	}
}

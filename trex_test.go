package trex

import (
	"regexp"
	"testing"
)

var (
	tcMatch = []struct {
		pattern string
		text    []string
	}{
		{
			`.*`,
			[]string{
				" ",
				"abcd",
			},
		},
		{
			`^[a-z]+\[[0-9]+\]$`,
			[]string{
				"adam[23]",
				"eve[7]",
				"Job[48]",
				"snakey",
			},
		},
		{
			`日本語+`,
			[]string{
				"日本語",
				"日本語語語語",
				"",
			},
		},
		{
			`/$`,
			[]string{
				"/abc/",
				"/abc",
			},
		},
		{
			`[a\-\]z]+`,
			[]string{
				"az]-bcz",
				"abcd\n",
				"abcd",
				"ab1234cd",
			},
		},
		{
			`foo.*`,
			[]string{
				"seafood",
			},
		},
		{
			`^abcd$`,
			[]string{
				"abcd",
				"abcde",
			},
		},
		{
			`[\w\.+-]+@[\w\.-]+\.[\w\.-]+`,
			[]string{
				"kuba--@noreplay.github.com",
				"kuba--(at)noreplay.github.com",
			},
		},
		{
			`[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}[-a-zA-Z0-9@:%_\+.~#?&//=]*`,
			[]string{
				"http://www.foufos.gr",
				"https://www.foufos.gr",
				"http://foufos.gr",
				"http://www.foufos.gr/kino",
				"http://werer.gr",
				"www.foufos.gr",
				"www.mp3.com",
				"www.t.co",
				"http://t.co",
				"http://www.t.co",
				"https://www.t.co",
				"www.aa.com",
				"http://aa.com",
				"http://www.aa.com",
				"https://www.aa.com",
				"www.foufos",
				"www.foufos-.gr",
				"www.-foufos.gr",
				"foufos.gr",
				"http://www.foufos",
				"http://foufos",
				"www.mp3#.com",
			},
		},
	}
)

func TestCompileAndMatch(t *testing.T) {
	for _, tc := range tcMatch {
		t.Run(tc.pattern, func(t *testing.T) {
			gorex := regexp.MustCompile(tc.pattern)
			trex, err := Compile(tc.pattern)
			if err != nil {
				t.Fatalf("Compile: %v", err)
			}
			t.Logf("\n%s\n", trex.String())

			for _, txt := range tc.text {
				exp := gorex.MatchString(txt)
				act := trex.Match([]byte(txt))
				if act != exp {
					t.Errorf("txt: %s, exp: %v, act: %v\n", txt, exp, act)
				}
			}
		})
	}
}

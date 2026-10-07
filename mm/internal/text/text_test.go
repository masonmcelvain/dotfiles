package text

import "testing"

func TestFit(t *testing.T) {
	cases := []struct {
		s    string
		w    int
		want string
	}{
		{"abc", 5, "abc  "},
		{"abcdef", 4, "abc…"},
		{"abcd", 4, "abcd"},
		// Wide characters take two columns each.
		{"日本語", 6, "日本語"},
		{"日本語", 5, "日本…"},
		{"日本語x", 8, "日本語x "},
		{"x", 0, ""},
	}
	for _, c := range cases {
		if got := Fit(c.s, c.w); got != c.want {
			t.Errorf("Fit(%q, %d) = %q, want %q", c.s, c.w, got, c.want)
		}
		if c.w > 0 && Width(Fit(c.s, c.w)) != c.w {
			t.Errorf("Fit(%q, %d) is %d columns wide", c.s, c.w, Width(Fit(c.s, c.w)))
		}
	}
}

func TestAge(t *testing.T) {
	cases := map[int64]string{
		-5:       "0s",
		42:       "42s",
		60:       "1m",
		3599:     "59m",
		7200:     "2h",
		86400:    "1d",
		2592000:  "1mo",
		31536000: "1y",
	}
	for s, want := range cases {
		if got := Age(s); got != want {
			t.Errorf("Age(%d) = %q, want %q", s, got, want)
		}
	}
}

func TestParseDuration(t *testing.T) {
	cases := map[string]int64{
		"3days 1h 24m 8s":                 3*86400 + 3600 + 24*60 + 8,
		"4months 21days 23h 51m 25s":      4*2592000 + 21*86400 + 23*3600 + 51*60 + 25,
		"1month 27days 15h 28m 24s":       2592000 + 27*86400 + 15*3600 + 28*60 + 24,
		"1day 6h 27m 45s ago] (EXITED - ": 86400 + 6*3600 + 27*60 + 45,
		"1year 2weeks":                    31536000 + 2*604800,
		"":                                0,
	}
	for s, want := range cases {
		if got := ParseDuration(s); got != want {
			t.Errorf("ParseDuration(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestTilde(t *testing.T) {
	t.Setenv("HOME", "/home/me")
	cases := map[string]string{
		"/home/me":       "~",
		"/home/me/Code":  "~/Code",
		"/home/meow":     "/home/meow",
		"/srv/home/me/x": "/srv/home/me/x",
	}
	for in, want := range cases {
		if got := Tilde(in); got != want {
			t.Errorf("Tilde(%q) = %q, want %q", in, got, want)
		}
	}
}

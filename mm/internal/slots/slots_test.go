package slots

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestNum(t *testing.T) {
	cases := map[string]string{
		"/home/mmcelvain/Code":    "1",
		"/home/mmcelvain-4/Code":  "4",
		"/home/mmcelvain-12/Code": "12",
		"/home/some-user/Code":    "1",
	}
	for in, want := range cases {
		if got := Num(in); got != want {
			t.Errorf("Num(%q) = %q, want %q", in, got, want)
		}
	}
	if got := Name("/home/mmcelvain-3/Code"); got != "s3" {
		t.Errorf("Name = %q", got)
	}
}

func TestOrder(t *testing.T) {
	now := time.Now()
	s := []slot{
		{dir: "b", stamp: now},
		{dir: "a", stamp: now.Add(-time.Hour)},
		{dir: "c", stamp: now.Add(time.Hour)},
	}
	order(s)
	var got []string
	for _, x := range s {
		got = append(got, x.dir)
	}
	if want := []string{"a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestAll(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"u/Code", "u-2/Code", "u-3"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("MM_SLOTS", filepath.Join(root, "u*/Code"))
	want := []string{filepath.Join(root, "u/Code"), filepath.Join(root, "u-2/Code")}
	if got := All(); !reflect.DeepEqual(got, want) {
		t.Errorf("All = %v, want %v", got, want)
	}
	if !Available() {
		t.Error("Available = false")
	}
	t.Setenv("MM_SLOTS", filepath.Join(root, "nope*/Code"))
	if Available() {
		t.Error("Available = true with no slots")
	}
}

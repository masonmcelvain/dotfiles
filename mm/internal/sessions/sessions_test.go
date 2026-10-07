package sessions

import (
	"reflect"
	"testing"
	"time"
)

func TestWidths(t *testing.T) {
	cases := []struct {
		in   [4]int
		want [3]int
	}{
		// Fits: nothing changes.
		{in: [4]int{10, 20, 30, 100}, want: [3]int{10, 20, 30}},
		// The path yields first, down to 12.
		{in: [4]int{10, 20, 30, 50}, want: [3]int{10, 20, 20}},
		// Then the branch, down to 12.
		{in: [4]int{10, 20, 30, 38}, want: [3]int{10, 16, 12}},
		// Then the name, down to 8.
		{in: [4]int{10, 20, 30, 33}, want: [3]int{9, 12, 12}},
		// Past every floor, the path absorbs the rest, down to 4.
		{in: [4]int{10, 20, 30, 28}, want: [3]int{8, 12, 8}},
		{in: [4]int{10, 20, 30, 10}, want: [3]int{8, 12, 4}},
	}
	for _, c := range cases {
		n, b, p := Widths(c.in[0], c.in[1], c.in[2], c.in[3])
		if got := [3]int{n, b, p}; got != c.want {
			t.Errorf("Widths%v = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSort(t *testing.T) {
	now := time.Now()
	rows := []Row{
		{Name: "old-exited", Exited: true, Used: now.Add(-48 * time.Hour)},
		{Name: "idle", Used: now.Add(-time.Hour)},
		{Name: "new-exited", Exited: true, Used: now.Add(-time.Hour)},
		{Name: "busy", Used: now},
	}
	Sort(rows)
	var got []string
	for _, r := range rows {
		got = append(got, r.Name)
	}
	if want := []string{"busy", "idle", "new-exited", "old-exited"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Sort = %v, want %v", got, want)
	}
}

func TestFindBranch(t *testing.T) {
	t.Setenv("ZELLIJ_SESSION_NAME", "")
	rows := []Row{
		{Name: "live", Branch: "main"},
		{Name: "exited", Exited: true, Branch: "main"},
		{Name: "other", Branch: "fix"},
	}
	if name, ok := FindBranch(rows, "main"); !ok || name != "live" {
		t.Errorf("FindBranch(main) = %q, %v", name, ok)
	}
	if _, ok := FindBranch(rows, "nope"); ok {
		t.Error("FindBranch found a missing branch")
	}
}

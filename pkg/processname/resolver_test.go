package processname

import (
	"strings"
	"testing"
)

func testResolver() Resolver {
	r := Resolver{byID: map[int64]string{}, byCode: map[string]int64{}, names: map[string]struct{}{}}
	add := func(id int64, code, name string) {
		r.byID[id] = name
		r.byCode[code] = id
		r.names[strings.ToLower(name)] = struct{}{}
	}
	add(2, "BEND", "Bending")
	add(54, "STAMPING+SUBCON", "Stamping")
	add(204, "Stamping + Subcon", "Stamping")
	add(69, "Stamping 110T", "Stamping 110T")
	return r
}

func TestName(t *testing.T) {
	r := testResolver()
	cases := []struct {
		id         int64
		code, snap string
		want       string
	}{
		{0, "", "BEND", "Bending"},                 // legacy snapshot holding the code
		{54, "STAMPING+SUBCON", "Old", "Stamping"}, // new snapshot
		{0, "", "Unknown", "Unknown"},
	}
	for _, c := range cases {
		if got := r.Name(c.id, c.code, c.snap); got != c.want {
			t.Errorf("Name(%d,%q,%q) = %q, want %q", c.id, c.code, c.snap, got, c.want)
		}
	}
	if (Resolver{}).Name(0, "", " BEND ") != "BEND" {
		t.Error("zero Resolver must fall back to the snapshot name")
	}
}

func TestSameProcess(t *testing.T) {
	r := testResolver()
	cases := []struct {
		machine int64
		snap    string
		want    bool
	}{
		{2, "BEND", true},              // same id
		{204, "STAMPING+SUBCON", true}, // different id, same name "Stamping"
		{69, "STAMPING+SUBCON", false}, // Stamping 110T vs Stamping
		{2, "STAMPING+SUBCON", false},  // Bending vs Stamping
		{0, "BEND", false},             // machine without process
		{2, "bending", true},           // snapshot holding a name
	}
	for _, c := range cases {
		if got := r.SameProcess(c.machine, 0, "", c.snap); got != c.want {
			t.Errorf("SameProcess(%d,%q) = %v, want %v", c.machine, c.snap, got, c.want)
		}
	}
}

func TestKnown(t *testing.T) {
	r := testResolver()
	if !r.Known(0, "", "BEND") || !r.Known(0, "", "stamping") {
		t.Error("registered process must be known")
	}
	if r.Known(0, "", "RM Processing") || r.Known(0, "", "") {
		t.Error("pseudo/empty process must not be known")
	}
}

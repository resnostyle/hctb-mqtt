package hctb

import (
	"testing"
	"time"

	hcb "github.com/resnostyle/hctb-mqtt/internal/lib/hctb"
)

func TestStudentSlug(t *testing.T) {
	if got := StudentSlug(hcb.Student{FirstName: "Kaylen"}); got != "kaylen" {
		t.Fatalf("got %q", got)
	}
	if got := StudentSlug(hcb.Student{StudentID: "ABCDEF123456"}); got != "abcdef12" {
		t.Fatalf("id slug = %q", got)
	}
}

func TestPickPrimaryStop(t *testing.T) {
	stops := []hcb.StudentStop{
		{Name: "School", StopType: "School"},
		{Name: "Home", StopType: "Stop", Latitude: 1, Longitude: 2},
	}
	s := PickPrimaryStop(stops)
	if s == nil || s.Name != "Home" {
		t.Fatalf("got %+v", s)
	}
}

func TestBuildDistanceETA(t *testing.T) {
	v := &hcb.VehicleLocation{Latitude: 34.79, Longitude: -86.78, Speed: 30}
	stop := &hcb.StudentStop{Latitude: 34.80773, Longitude: -86.74986}
	d := BuildDistance(v, stop, nil, nil, time.Now())
	if d.ToStopMiles == nil || *d.ToStopMiles <= 0 {
		t.Fatalf("miles = %v", d.ToStopMiles)
	}
	if d.ETASeconds == nil || *d.ETASeconds <= 0 {
		t.Fatalf("eta = %v", d.ETASeconds)
	}
}

func TestActiveRoute(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	s := Settings{
		Location:       loc,
		MorningStart:   timeOfDayClock{6, 0},
		MorningEnd:     timeOfDayClock{9, 0},
		AfternoonStart: timeOfDayClock{14, 0},
		AfternoonEnd:   timeOfDayClock{18, 0},
	}
	am := time.Date(2024, 10, 31, 7, 30, 0, 0, loc)
	label, id := s.ActiveRoute(am)
	if label != "AM" || id != hcb.AMID {
		t.Fatalf("am = %s %s", label, id)
	}
	pm := time.Date(2024, 10, 31, 15, 0, 0, 0, loc)
	label, id = s.ActiveRoute(pm)
	if label != "PM" || id != hcb.PMID {
		t.Fatalf("pm = %s %s", label, id)
	}
}

func TestInActiveWindow(t *testing.T) {
	loc := time.UTC
	s := Settings{
		Location:       loc,
		MorningStart:   timeOfDayClock{6, 0},
		MorningEnd:     timeOfDayClock{9, 0},
		AfternoonStart: timeOfDayClock{14, 0},
		AfternoonEnd:   timeOfDayClock{18, 0},
	}
	if !s.InActiveWindow(time.Date(2024, 1, 1, 7, 0, 0, 0, loc)) {
		t.Fatal("expected morning active")
	}
	if s.InActiveWindow(time.Date(2024, 1, 1, 12, 0, 0, 0, loc)) {
		t.Fatal("expected midday idle")
	}
}

func TestFilterStudents(t *testing.T) {
	all := []hcb.Student{
		{StudentID: "1", FirstName: "A"},
		{StudentID: "2", FirstName: "B"},
	}
	got := FilterStudents(all, map[string]struct{}{"2": {}})
	if len(got) != 1 || got[0].StudentID != "2" {
		t.Fatalf("got %+v", got)
	}
	got = FilterStudents(all, map[string]struct{}{"a": {}})
	if len(got) != 1 || got[0].FirstName != "A" {
		t.Fatalf("slug filter %+v", got)
	}
}

func TestNextInterval(t *testing.T) {
	loc := time.UTC
	s := Settings{
		Location:                loc,
		PollIntervalSeconds:     90,
		IdlePollIntervalSeconds: 900,
		MorningStart:            timeOfDayClock{6, 0},
		MorningEnd:              timeOfDayClock{9, 0},
		AfternoonStart:          timeOfDayClock{14, 0},
		AfternoonEnd:            timeOfDayClock{18, 0},
	}
	idle := NextInterval(s, time.Date(2024, 1, 1, 12, 0, 0, 0, loc), false)
	if idle != 900*time.Second {
		t.Fatalf("idle = %s", idle)
	}
	active := NextInterval(s, time.Date(2024, 1, 1, 7, 0, 0, 0, loc), false)
	if active != 90*time.Second {
		t.Fatalf("active = %s", active)
	}
	live := NextInterval(s, time.Date(2024, 1, 1, 12, 0, 0, 0, loc), true)
	if live != 60*time.Second {
		t.Fatalf("live = %s", live)
	}
}

func TestBuildDiscoveryConfigs(t *testing.T) {
	students := []hcb.Student{{StudentID: "1", FirstName: "Test1"}}
	cfgs := BuildDiscoveryConfigs("home/bus", students)
	if len(cfgs) < 10 {
		t.Fatalf("configs = %d", len(cfgs))
	}
	foundTracker := false
	for _, c := range cfgs {
		if c.Component == "device_tracker" {
			foundTracker = true
		}
	}
	if !foundTracker {
		t.Fatal("missing device_tracker")
	}
}

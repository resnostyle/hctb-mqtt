package hctb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseSchoolID(t *testing.T) {
	id, err := ParseSchoolID(mustRead(t, "s1100.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if id != "5C9756CEEA1B438A80C65C20D2720E23" {
		t.Fatalf("school id = %q", id)
	}
}

func TestParseAccount(t *testing.T) {
	acct, err := ParseAccount(mustRead(t, "s1157.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if acct.AccountID != "55503A65D2F448AC8DB5F8181B305281" {
		t.Fatalf("account = %q", acct.AccountID)
	}
	if len(acct.Students) != 2 {
		t.Fatalf("students = %d", len(acct.Students))
	}
	if acct.Students[0].FirstName != "Test1" {
		t.Fatalf("first student = %+v", acct.Students[0])
	}
	if len(acct.Times) != 3 {
		t.Fatalf("times = %d", len(acct.Times))
	}
	if acct.Times[0].ID != AMID {
		t.Fatalf("am id = %q", acct.Times[0].ID)
	}
}

func TestParseStopInfoAM(t *testing.T) {
	info, err := ParseStopInfo(mustRead(t, "s1158_AM.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.VehicleLocation == nil {
		t.Fatal("expected vehicle")
	}
	if info.VehicleLocation.Speed != 35 {
		t.Fatalf("speed = %d", info.VehicleLocation.Speed)
	}
	if !info.VehicleLocation.LogTime.IsZero() && info.VehicleLocation.LogTime.Hour() != 7 {
		t.Fatalf("log hour = %v", info.VehicleLocation.LogTime)
	}
	if len(info.StudentStops) != 2 {
		t.Fatalf("stops = %d", len(info.StudentStops))
	}
	if info.StudentStops[1].StopType != "Stop" {
		t.Fatalf("stop type = %q", info.StudentStops[1].StopType)
	}
}

func TestParseStopInfoPM(t *testing.T) {
	info, err := ParseStopInfo(mustRead(t, "s1158_PM.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.VehicleLocation == nil {
		t.Fatal("expected vehicle")
	}
	if len(info.StudentStops) != 2 {
		t.Fatalf("stops = %d", len(info.StudentStops))
	}
	if info.StudentStops[0].TimeOfDayID != PMID {
		t.Fatalf("tod = %q", info.StudentStops[0].TimeOfDayID)
	}
}

func TestParseAccountEmpty(t *testing.T) {
	_, err := ParseAccount([]byte(`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body></s:Body></s:Envelope>`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClientGetSchoolID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("SOAPAction") != "http://tempuri.org/ISynoviaApi/s1100" {
			t.Errorf("soapaction = %q", r.Header.Get("SOAPAction"))
		}
		w.Write(mustRead(t, "s1100.xml"))
	}))
	defer srv.Close()

	c := New(srv.Client())
	c.URL = srv.URL
	id, err := c.GetSchoolID(context.Background(), "88552")
	if err != nil {
		t.Fatal(err)
	}
	if id != "5C9756CEEA1B438A80C65C20D2720E23" {
		t.Fatalf("id = %q", id)
	}
}

func TestClientGetParentInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(mustRead(t, "s1157.xml"))
	}))
	defer srv.Close()

	c := New(srv.Client())
	c.URL = srv.URL
	acct, err := c.GetParentInfo(context.Background(), "school", "user", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if len(acct.Students) != 2 {
		t.Fatalf("students = %d", len(acct.Students))
	}
}

func TestClientGetStopInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(mustRead(t, "s1158_AM.xml"))
	}))
	defer srv.Close()

	c := New(srv.Client())
	c.URL = srv.URL
	info, err := c.GetStopInfo(context.Background(), "s", "p", "st", AMID)
	if err != nil {
		t.Fatal(err)
	}
	if info.VehicleLocation == nil || info.VehicleLocation.Address == "" {
		t.Fatalf("vehicle = %+v", info.VehicleLocation)
	}
}

func TestClientAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New(srv.Client())
	c.URL = srv.URL
	_, err := c.GetSchoolID(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*APIError); !ok {
		t.Fatalf("want APIError, got %T", err)
	}
}

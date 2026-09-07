package hctb

import "time"

const (
	// AMID is the Synovia time-of-day ID for morning routes.
	AMID = "55632A13-35C5-4169-B872-F5ABDC25DF6A"
	// PMID is the Synovia time-of-day ID for afternoon routes.
	PMID = "6E7A050E-0295-4200-8EDC-3611BB5DE1C1"

	AppVersion = "3.6.0"
	DefaultURL = "https://api.synovia.com/SynoviaApi.svc"
)

// Student is a linked student on a parent account.
type Student struct {
	StudentID string
	FirstName string
	LastName  string
}

// TimeOfDay is a route window returned by parent login.
type TimeOfDay struct {
	ID        string
	Name      string
	BeginTime string // HH:MM:SS
	EndTime   string
}

// Account is the parent login result.
type Account struct {
	AccountID string
	Students  []Student
	Times     []TimeOfDay
}

// StudentStop is a scheduled stop on a route.
type StudentStop struct {
	Name                     string
	Latitude                 float64
	Longitude                float64
	StartTime                string
	StopType                 string
	SubstituteVehicleName    string
	VehicleName              string
	StopID                   string
	ArrivalTime              string
	TimeOfDayID              string
	VehicleID                string
	ESN                      string
	TierStartTime            string
	BusVisibilityStartOffset int
}

// VehicleLocation is the live bus GPS payload (nil when not tracking).
type VehicleLocation struct {
	Name           string
	Latitude       float64
	Longitude      float64
	LogTime        time.Time
	Ignition       bool
	Latent         bool
	TimeZoneOffset int
	Heading        string
	Speed          int
	Address        string
	MessageCode    int
	DisplayOnMap   bool
}

// StopInfo is the s1158 response.
type StopInfo struct {
	VehicleLocation *VehicleLocation
	StudentStops    []StudentStop
}

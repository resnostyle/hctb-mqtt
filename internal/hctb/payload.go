package hctb

import (
	"math"
	"regexp"
	"strings"
	"time"

	hcb "github.com/resnostyle/hctb-mqtt/internal/lib/hctb"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// StudentSlug builds a stable MQTT path segment from first name (fallback: id prefix).
func StudentSlug(s hcb.Student) string {
	base := strings.ToLower(strings.TrimSpace(s.FirstName))
	base = nonSlug.ReplaceAllString(base, "_")
	base = strings.Trim(base, "_")
	if base == "" {
		id := strings.ToLower(s.StudentID)
		if len(id) > 8 {
			id = id[:8]
		}
		return id
	}
	return base
}

// SummaryPayload is published to home/bus/summary.
type SummaryPayload struct {
	OK           bool              `json:"ok"`
	UpdatedAt    string            `json:"updated_at"`
	StudentCount int               `json:"student_count"`
	Route        string            `json:"route"`
	Error        string            `json:"error,omitempty"`
	Students     []StudentListItem `json:"students"`
}

// StudentListItem is a roster entry on the summary topic.
type StudentListItem struct {
	Slug      string `json:"slug"`
	StudentID string `json:"student_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// CurrentPayload is the active route snapshot for one student.
type CurrentPayload struct {
	Route          string `json:"route"`
	TimeOfDayID    string `json:"time_of_day_id"`
	StopName       string `json:"stop_name"`
	VehiclePresent bool   `json:"vehicle_present"`
	VehicleName    string `json:"vehicle_name,omitempty"`
	UpdatedAt      string `json:"updated_at"`
}

// VehiclePayload is live GPS (or stale marker).
type VehiclePayload struct {
	Available    bool    `json:"available"`
	Name         string  `json:"name,omitempty"`
	Latitude     float64 `json:"latitude,omitempty"`
	Longitude    float64 `json:"longitude,omitempty"`
	Speed        int     `json:"speed,omitempty"`
	Heading      string  `json:"heading,omitempty"`
	Address      string  `json:"address,omitempty"`
	Ignition     bool    `json:"ignition"`
	DisplayOnMap bool    `json:"display_on_map"`
	LogTime      string  `json:"log_time,omitempty"`
	UpdatedAt    string  `json:"updated_at"`
}

// StopPayload lists stops for the active route.
type StopPayload struct {
	Stops     []StopItem `json:"stops"`
	UpdatedAt string     `json:"updated_at"`
}

// StopItem is one scheduled stop.
type StopItem struct {
	Name        string  `json:"name"`
	StopType    string  `json:"stop_type"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	ArrivalTime string  `json:"arrival_time"`
	VehicleName string  `json:"vehicle_name"`
	StopID      string  `json:"stop_id"`
}

// DistancePayload is distance/ETA to stop and optional home.
type DistancePayload struct {
	ToStopMeters *float64 `json:"to_stop_meters"`
	ToStopMiles  *float64 `json:"to_stop_miles"`
	ToHomeMeters *float64 `json:"to_home_meters,omitempty"`
	ToHomeMiles  *float64 `json:"to_home_miles,omitempty"`
	ETASeconds   *int     `json:"eta_seconds"`
	UpdatedAt    string   `json:"updated_at"`
}

// PickPrimaryStop prefers StopType=Stop, else first stop.
func PickPrimaryStop(stops []hcb.StudentStop) *hcb.StudentStop {
	for i := range stops {
		if strings.EqualFold(stops[i].StopType, "Stop") {
			return &stops[i]
		}
	}
	if len(stops) == 0 {
		return nil
	}
	return &stops[0]
}

func BuildCurrent(route, todID string, stop *hcb.StudentStop, vehicle *hcb.VehicleLocation, now time.Time) CurrentPayload {
	p := CurrentPayload{
		Route:       route,
		TimeOfDayID: todID,
		UpdatedAt:   now.UTC().Format(time.RFC3339),
	}
	if stop != nil {
		p.StopName = stop.Name
		p.VehicleName = stop.VehicleName
	}
	if vehicle != nil {
		p.VehiclePresent = true
		if vehicle.Name != "" {
			p.VehicleName = vehicle.Name
		}
	}
	return p
}

func BuildVehicle(vehicle *hcb.VehicleLocation, now time.Time) VehiclePayload {
	p := VehiclePayload{
		Available: vehicle != nil,
		UpdatedAt: now.UTC().Format(time.RFC3339),
	}
	if vehicle == nil {
		return p
	}
	p.Name = vehicle.Name
	p.Latitude = vehicle.Latitude
	p.Longitude = vehicle.Longitude
	p.Speed = vehicle.Speed
	p.Heading = vehicle.Heading
	p.Address = vehicle.Address
	p.Ignition = vehicle.Ignition
	p.DisplayOnMap = vehicle.DisplayOnMap
	if !vehicle.LogTime.IsZero() {
		p.LogTime = vehicle.LogTime.UTC().Format(time.RFC3339)
	}
	return p
}

func BuildStop(stops []hcb.StudentStop, now time.Time) StopPayload {
	items := make([]StopItem, 0, len(stops))
	for _, s := range stops {
		items = append(items, StopItem{
			Name:        s.Name,
			StopType:    s.StopType,
			Latitude:    s.Latitude,
			Longitude:   s.Longitude,
			ArrivalTime: s.ArrivalTime,
			VehicleName: s.VehicleName,
			StopID:      s.StopID,
		})
	}
	return StopPayload{Stops: items, UpdatedAt: now.UTC().Format(time.RFC3339)}
}

func BuildDistance(vehicle *hcb.VehicleLocation, stop *hcb.StudentStop, homeLat, homeLon *float64, now time.Time) DistancePayload {
	p := DistancePayload{UpdatedAt: now.UTC().Format(time.RFC3339)}
	if vehicle == nil {
		return p
	}
	if stop != nil && (stop.Latitude != 0 || stop.Longitude != 0) {
		m := haversineMeters(vehicle.Latitude, vehicle.Longitude, stop.Latitude, stop.Longitude)
		miles := m / 1609.344
		p.ToStopMeters = &m
		p.ToStopMiles = &miles
		if vehicle.Speed > 0 {
			eta := int(math.Round(miles / float64(vehicle.Speed) * 3600))
			p.ETASeconds = &eta
		}
	}
	if homeLat != nil && homeLon != nil {
		m := haversineMeters(vehicle.Latitude, vehicle.Longitude, *homeLat, *homeLon)
		miles := m / 1609.344
		p.ToHomeMeters = &m
		p.ToHomeMiles = &miles
	}
	return p
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earth = 6371000.0
	toRad := math.Pi / 180
	dLat := (lat2 - lat1) * toRad
	dLon := (lon2 - lon1) * toRad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*toRad)*math.Cos(lat2*toRad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earth * math.Asin(math.Sqrt(a))
}

func RosterKey(students []hcb.Student) string {
	parts := make([]string, 0, len(students))
	for _, s := range students {
		parts = append(parts, s.StudentID+":"+StudentSlug(s))
	}
	return strings.Join(parts, "|")
}

func FilterStudents(all []hcb.Student, allow map[string]struct{}) []hcb.Student {
	if len(allow) == 0 {
		return all
	}
	out := make([]hcb.Student, 0, len(all))
	for _, s := range all {
		if _, ok := allow[s.StudentID]; ok {
			out = append(out, s)
			continue
		}
		if _, ok := allow[StudentSlug(s)]; ok {
			out = append(out, s)
		}
	}
	return out
}

package hctb

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	hcb "github.com/resnostyle/hctb-mqtt/internal/lib/hctb"
	"github.com/resnostyle/mqttkit/env"
)

const (
	defaultTopicPrefix = "home/bus"
	defaultClientID    = "hctb-mqtt"
)

// Settings is runtime configuration for hctb-mqtt.
type Settings struct {
	env.MQTT
	SchoolCode              string
	Username                string
	Password                string
	PollIntervalSeconds     int
	IdlePollIntervalSeconds int
	Timezone                string
	Location                *time.Location
	MorningStart            timeOfDayClock
	MorningEnd              timeOfDayClock
	AfternoonStart          timeOfDayClock
	AfternoonEnd            timeOfDayClock
	HomeLat                 *float64
	HomeLon                 *float64
	StudentIDs              map[string]struct{} // empty = all
}

type timeOfDayClock struct {
	Hour   int
	Minute int
}

func (t timeOfDayClock) minutes() int {
	return t.Hour*60 + t.Minute
}

// FromEnv loads settings from the environment.
func FromEnv() (Settings, error) {
	mqtt, err := env.LoadMQTT(defaultTopicPrefix, defaultClientID)
	if err != nil {
		return Settings{}, err
	}
	school, err := env.Require("HCTB_SCHOOL_CODE")
	if err != nil {
		return Settings{}, err
	}
	user, err := env.Require("HCTB_USERNAME")
	if err != nil {
		return Settings{}, err
	}
	pass, err := env.Require("HCTB_PASSWORD")
	if err != nil {
		return Settings{}, err
	}
	poll, err := env.Int("HCTB_POLL_INTERVAL_SECONDS", 90)
	if err != nil {
		return Settings{}, err
	}
	if poll < 30 {
		return Settings{}, fmt.Errorf("HCTB_POLL_INTERVAL_SECONDS must be >= 30")
	}
	idle, err := env.Int("HCTB_IDLE_POLL_INTERVAL_SECONDS", 900)
	if err != nil {
		return Settings{}, err
	}
	if idle < 60 {
		return Settings{}, fmt.Errorf("HCTB_IDLE_POLL_INTERVAL_SECONDS must be >= 60")
	}
	tzName := env.Get("HCTB_TIMEZONE", "America/New_York")
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return Settings{}, fmt.Errorf("HCTB_TIMEZONE: %w", err)
	}
	morningStart, err := parseClock("HCTB_MORNING_START", "06:00")
	if err != nil {
		return Settings{}, err
	}
	morningEnd, err := parseClock("HCTB_MORNING_END", "09:00")
	if err != nil {
		return Settings{}, err
	}
	afternoonStart, err := parseClock("HCTB_AFTERNOON_START", "14:00")
	if err != nil {
		return Settings{}, err
	}
	afternoonEnd, err := parseClock("HCTB_AFTERNOON_END", "18:00")
	if err != nil {
		return Settings{}, err
	}

	homeLat, err := optionalFloat("HCTB_HOME_LAT")
	if err != nil {
		return Settings{}, err
	}
	homeLon, err := optionalFloat("HCTB_HOME_LON")
	if err != nil {
		return Settings{}, err
	}
	if (homeLat == nil) != (homeLon == nil) {
		return Settings{}, fmt.Errorf("HCTB_HOME_LAT and HCTB_HOME_LON must both be set or both unset")
	}

	ids := map[string]struct{}{}
	for _, part := range strings.Split(os.Getenv("HCTB_STUDENT_IDS"), ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			ids[part] = struct{}{}
		}
	}

	return Settings{
		MQTT:                    mqtt,
		SchoolCode:              school,
		Username:                user,
		Password:                pass,
		PollIntervalSeconds:     poll,
		IdlePollIntervalSeconds: idle,
		Timezone:                tzName,
		Location:                loc,
		MorningStart:            morningStart,
		MorningEnd:              morningEnd,
		AfternoonStart:          afternoonStart,
		AfternoonEnd:            afternoonEnd,
		HomeLat:                 homeLat,
		HomeLon:                 homeLon,
		StudentIDs:              ids,
	}, nil
}

func parseClock(envName, fallback string) (timeOfDayClock, error) {
	raw := env.Get(envName, fallback)
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return timeOfDayClock{}, fmt.Errorf("invalid %s %q (want HH:MM)", envName, raw)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return timeOfDayClock{}, fmt.Errorf("invalid %s hour in %q", envName, raw)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return timeOfDayClock{}, fmt.Errorf("invalid %s minute in %q", envName, raw)
	}
	return timeOfDayClock{Hour: h, Minute: m}, nil
}

func optionalFloat(name string) (*float64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil, nil
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", name, err)
	}
	return &f, nil
}

// InActiveWindow reports whether local now is inside morning or afternoon windows.
func (s Settings) InActiveWindow(now time.Time) bool {
	local := now.In(s.Location)
	mins := local.Hour()*60 + local.Minute()
	return inRange(mins, s.MorningStart, s.MorningEnd) || inRange(mins, s.AfternoonStart, s.AfternoonEnd)
}

// ActiveRoute returns "AM" or "PM" and the Synovia time-of-day ID for local now.
func (s Settings) ActiveRoute(now time.Time) (label, timeOfDayID string) {
	local := now.In(s.Location)
	mins := local.Hour()*60 + local.Minute()
	if inRange(mins, s.MorningStart, s.MorningEnd) || mins < s.AfternoonStart.minutes() {
		return "AM", hcb.AMID
	}
	return "PM", hcb.PMID
}

func inRange(mins int, start, end timeOfDayClock) bool {
	a, b := start.minutes(), end.minutes()
	if a <= b {
		return mins >= a && mins < b
	}
	// wraps midnight
	return mins >= a || mins < b
}

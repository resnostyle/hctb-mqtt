package hctb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	hcb "github.com/resnostyle/hctb-mqtt/internal/lib/hctb"
	"github.com/resnostyle/mqttkit/mqttpub"
)

// API is the Synovia SOAP surface used by the poller.
type API interface {
	GetSchoolID(ctx context.Context, schoolCode string) (string, error)
	GetParentInfo(ctx context.Context, schoolID, username, password string) (hcb.Account, error)
	GetStopInfo(ctx context.Context, schoolID, parentID, studentID, timeOfDayID string) (hcb.StopInfo, error)
}

// Publisher holds session cache and discovery fingerprint across polls.
type Publisher struct {
	schoolID      string
	account       *hcb.Account
	lastRosterKey string
	failStreak    int
}

func NewPublisher() *Publisher {
	return &Publisher{}
}

func (p *Publisher) invalidateSession() {
	p.schoolID = ""
	p.account = nil
}

func (p *Publisher) ensureSession(ctx context.Context, settings Settings, api API) (string, hcb.Account, error) {
	if p.schoolID == "" {
		id, err := api.GetSchoolID(ctx, settings.SchoolCode)
		if err != nil {
			return "", hcb.Account{}, err
		}
		p.schoolID = id
		slog.Info("resolved school id", "school_code", settings.SchoolCode)
	}
	if p.account == nil {
		acct, err := api.GetParentInfo(ctx, p.schoolID, settings.Username, settings.Password)
		if err != nil {
			p.invalidateSession()
			return "", hcb.Account{}, err
		}
		p.account = &acct
		slog.Info("logged in", "account_id", acct.AccountID, "students", len(acct.Students))
	}
	return p.schoolID, *p.account, nil
}

// FetchAndPublish refreshes account (as needed), polls stops, and publishes retained topics.
// Returns whether any vehicle is actively tracking (for adaptive polling).
func (p *Publisher) FetchAndPublish(ctx context.Context, settings Settings, api API, mqtt mqttpub.Sink) (activeVehicle bool, err error) {
	now := time.Now()
	route, todID := settings.ActiveRoute(now)

	schoolID, acct, err := p.ensureSession(ctx, settings, api)
	if err != nil {
		p.failStreak++
		_ = publishSummary(mqtt, SummaryPayload{
			OK:        false,
			UpdatedAt: now.UTC().Format(time.RFC3339),
			Route:     route,
			Error:     err.Error(),
		})
		return false, err
	}

	students := FilterStudents(acct.Students, settings.StudentIDs)
	list := make([]StudentListItem, 0, len(students))
	for _, s := range students {
		list = append(list, StudentListItem{
			Slug:      StudentSlug(s),
			StudentID: s.StudentID,
			FirstName: s.FirstName,
			LastName:  s.LastName,
		})
	}

	anyVehicle := false
	for _, s := range students {
		info, serr := api.GetStopInfo(ctx, schoolID, acct.AccountID, s.StudentID, todID)
		if serr != nil {
			slog.Warn("stop info failed", "student", s.StudentID, "err", serr)
			_ = publishStudentError(mqtt, StudentSlug(s), route, todID, now, serr)
			continue
		}
		if info.VehicleLocation != nil {
			anyVehicle = true
			if info.VehicleLocation.Ignition || info.VehicleLocation.DisplayOnMap {
				activeVehicle = true
			}
		}
		if err := publishStudent(mqtt, settings, s, route, todID, info, now); err != nil {
			return anyVehicle, err
		}
	}

	if err := publishSummary(mqtt, SummaryPayload{
		OK:           true,
		UpdatedAt:    now.UTC().Format(time.RFC3339),
		StudentCount: len(students),
		Route:        route,
		Students:     list,
	}); err != nil {
		return activeVehicle, err
	}

	if settings.MQTTDiscoveryEnabled {
		key := RosterKey(students)
		if key != p.lastRosterKey {
			configs := BuildDiscoveryConfigs(settings.MQTTTopicPrefix, students)
			if err := mqtt.PublishDiscovery(configs, settings.MQTTDiscoveryPrefix); err != nil {
				return activeVehicle, fmt.Errorf("publish discovery: %w", err)
			}
			p.lastRosterKey = key
			slog.Info("published mqtt discovery", "entities", len(configs))
		}
	}

	p.failStreak = 0
	slog.Info("published bus snapshot",
		"students", len(students),
		"route", route,
		"vehicle", anyVehicle,
	)
	return activeVehicle, nil
}

func publishSummary(mqtt mqttpub.Sink, payload SummaryPayload) error {
	return mqtt.Publish("summary", payload, true)
}

func publishStudentError(mqtt mqttpub.Sink, slug, route, todID string, now time.Time, err error) error {
	_ = mqtt.Publish(slug+"/current", CurrentPayload{
		Route:       route,
		TimeOfDayID: todID,
		UpdatedAt:   now.UTC().Format(time.RFC3339),
	}, true)
	_ = mqtt.Publish(slug+"/vehicle", BuildVehicle(nil, now), true)
	return mqtt.Publish(slug+"/distance", DistancePayload{
		UpdatedAt: now.UTC().Format(time.RFC3339),
	}, true)
}

func publishStudent(mqtt mqttpub.Sink, settings Settings, s hcb.Student, route, todID string, info hcb.StopInfo, now time.Time) error {
	slug := StudentSlug(s)
	stop := PickPrimaryStop(info.StudentStops)
	if err := mqtt.Publish(slug+"/current", BuildCurrent(route, todID, stop, info.VehicleLocation, now), true); err != nil {
		return err
	}
	if err := mqtt.Publish(slug+"/vehicle", BuildVehicle(info.VehicleLocation, now), true); err != nil {
		return err
	}
	if err := mqtt.Publish(slug+"/stop", BuildStop(info.StudentStops, now), true); err != nil {
		return err
	}
	if err := mqtt.Publish(slug+"/distance", BuildDistance(info.VehicleLocation, stop, settings.HomeLat, settings.HomeLon, now), true); err != nil {
		return err
	}
	return nil
}

// PublishBootstrapDiscovery publishes bridge entities before the first successful poll.
func PublishBootstrapDiscovery(settings Settings, mqtt mqttpub.Sink) error {
	if !settings.MQTTDiscoveryEnabled {
		return nil
	}
	configs := BuildDiscoveryConfigs(settings.MQTTTopicPrefix, nil)
	if err := mqtt.PublishDiscovery(configs, settings.MQTTDiscoveryPrefix); err != nil {
		return err
	}
	slog.Info("published bootstrap discovery", "entities", len(configs))
	return nil
}

// NextInterval chooses active vs idle poll delay; shortens when a bus is live.
func NextInterval(settings Settings, now time.Time, activeVehicle bool) time.Duration {
	if activeVehicle {
		d := time.Duration(settings.PollIntervalSeconds) * time.Second
		if d > 60*time.Second {
			return 60 * time.Second
		}
		return d
	}
	if settings.InActiveWindow(now) {
		return time.Duration(settings.PollIntervalSeconds) * time.Second
	}
	return time.Duration(settings.IdlePollIntervalSeconds) * time.Second
}

// Backoff after consecutive failures (caps at 5 minutes).
func (p *Publisher) Backoff() time.Duration {
	n := p.failStreak
	if n < 1 {
		n = 1
	}
	d := time.Duration(n) * 30 * time.Second
	if d > 5*time.Minute {
		return 5 * time.Minute
	}
	return d
}

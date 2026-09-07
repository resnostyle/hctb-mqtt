package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/resnostyle/hctb-mqtt/internal/hctb"
	hcb "github.com/resnostyle/hctb-mqtt/internal/lib/hctb"
	"github.com/resnostyle/mqttkit/logx"
	"github.com/resnostyle/mqttkit/mqttpub"
	"github.com/resnostyle/mqttkit/poll"
)

func main() {
	settings, err := hctb.FromEnv()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	logx.Configure(settings.LogLevel, false)

	slog.Info("starting hctb-mqtt",
		"school_code", settings.SchoolCode,
		"poll_interval", settings.PollIntervalSeconds,
		"idle_poll_interval", settings.IdlePollIntervalSeconds,
		"timezone", settings.Timezone,
		"mqtt", settings.MQTTHost,
		"port", settings.MQTTPort,
		"prefix", settings.MQTTTopicPrefix,
		"discovery", settings.MQTTDiscoveryEnabled,
	)

	ctx, cancel := poll.NotifyContext()
	defer cancel()

	mqtt, err := mqttpub.New(
		settings.MQTTHost,
		settings.MQTTPort,
		settings.MQTTClientID,
		settings.MQTTUsername,
		settings.MQTTPassword,
		settings.MQTTTopicPrefix,
	)
	if err != nil {
		slog.Error("mqtt connect failed", "err", err)
		os.Exit(1)
	}
	defer mqtt.Close()

	if err := hctb.PublishBootstrapDiscovery(settings, mqtt); err != nil {
		slog.Error("mqtt discovery publish failed", "err", err)
	}

	client := hcb.New(nil)
	pub := hctb.NewPublisher()

	for ctx.Err() == nil {
		active, err := pub.FetchAndPublish(ctx, settings, client, mqtt)
		if err != nil {
			slog.Error("fetch/publish failed", "err", err)
			poll.Wait(ctx, pub.Backoff())
			continue
		}
		interval := hctb.NextInterval(settings, time.Now(), active)
		slog.Debug("waiting", "interval_s", int(interval.Seconds()))
		poll.Wait(ctx, interval)
	}
}

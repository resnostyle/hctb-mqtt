package hctb

import (
	"fmt"
	"strings"

	hcb "github.com/resnostyle/hctb-mqtt/internal/lib/hctb"
	"github.com/resnostyle/mqttkit/hadisc"
	"github.com/resnostyle/mqttkit/mqttpub"
)

const deviceManufacturer = "hctb-mqtt"

func studentDevice(slug, display string) map[string]any {
	uid := "hctb_" + slug
	return hadisc.Device([]string{uid}, fmt.Sprintf("%s Bus", display), deviceManufacturer, "Here Comes The Bus")
}

// BuildDiscoveryConfigs returns one HA device per student plus summary sensors.
func BuildDiscoveryConfigs(topicPrefix string, students []hcb.Student) []mqttpub.Config {
	summaryTopic := mqttpub.Join(topicPrefix, "summary")
	bridgeDevice := hadisc.Device([]string{"hctb_mqtt"}, "HCTB MQTT", deviceManufacturer, "Poller")

	configs := []mqttpub.Config{
		{
			ObjectID:  "hctb_mqtt_ok",
			Component: "binary_sensor",
			Payload: map[string]any{
				"name":                  "HCTB OK",
				"unique_id":             "hctb_mqtt_ok",
				"state_topic":           summaryTopic,
				"value_template":        "{{ value_json.ok }}",
				"payload_on":            "true",
				"payload_off":           "false",
				"device":                bridgeDevice,
				"object_id":             "hctb_mqtt_ok",
				"icon":                  "mdi:bus-school",
				"json_attributes_topic": summaryTopic,
			},
		},
		{
			ObjectID:  "hctb_mqtt_last_update",
			Component: "sensor",
			Payload: map[string]any{
				"name":                  "HCTB Last Update",
				"unique_id":             "hctb_mqtt_last_update",
				"state_topic":           summaryTopic,
				"value_template":        "{{ value_json.updated_at }}",
				"device":                bridgeDevice,
				"object_id":             "hctb_mqtt_last_update",
				"device_class":          "timestamp",
				"icon":                  "mdi:clock-check-outline",
				"json_attributes_topic": summaryTopic,
			},
		},
	}

	for _, s := range students {
		configs = append(configs, studentDiscovery(topicPrefix, s)...)
	}
	return configs
}

func studentDiscovery(topicPrefix string, s hcb.Student) []mqttpub.Config {
	slug := StudentSlug(s)
	display := strings.TrimSpace(s.FirstName)
	if display == "" {
		display = slug
	}
	device := studentDevice(slug, display)
	prefix := "hctb_" + slug
	currentTopic := mqttpub.Join(topicPrefix, slug+"/current")
	vehicleTopic := mqttpub.Join(topicPrefix, slug+"/vehicle")
	stopTopic := mqttpub.Join(topicPrefix, slug+"/stop")
	distanceTopic := mqttpub.Join(topicPrefix, slug+"/distance")

	return []mqttpub.Config{
		{
			ObjectID:  prefix + "_tracker",
			Component: "device_tracker",
			Payload: map[string]any{
				"name":                  fmt.Sprintf("%s Bus", display),
				"unique_id":             prefix + "_tracker",
				"json_attributes_topic": vehicleTopic,
				"source_type":           "gps",
				"device":                device,
				"object_id":             prefix + "_tracker",
				"icon":                  "mdi:bus",
			},
		},
		{
			ObjectID:  prefix + "_route",
			Component: "sensor",
			Payload: map[string]any{
				"name":                  fmt.Sprintf("%s Route", display),
				"unique_id":             prefix + "_route",
				"state_topic":           currentTopic,
				"value_template":        "{{ value_json.route }}",
				"device":                device,
				"object_id":             prefix + "_route",
				"icon":                  "mdi:routes",
				"json_attributes_topic": currentTopic,
			},
		},
		{
			ObjectID:  prefix + "_stop_name",
			Component: "sensor",
			Payload: map[string]any{
				"name":                  fmt.Sprintf("%s Stop", display),
				"unique_id":             prefix + "_stop_name",
				"state_topic":           currentTopic,
				"value_template":        "{{ value_json.stop_name }}",
				"device":                device,
				"object_id":             prefix + "_stop_name",
				"icon":                  "mdi:bus-stop",
				"json_attributes_topic": stopTopic,
			},
		},
		{
			ObjectID:  prefix + "_speed",
			Component: "sensor",
			Payload: map[string]any{
				"name":                  fmt.Sprintf("%s Bus Speed", display),
				"unique_id":             prefix + "_speed",
				"state_topic":           vehicleTopic,
				"value_template":        "{{ value_json.speed }}",
				"unit_of_measurement":   "mph",
				"device":                device,
				"object_id":             prefix + "_speed",
				"icon":                  "mdi:speedometer",
				"state_class":           "measurement",
				"json_attributes_topic": vehicleTopic,
			},
		},
		{
			ObjectID:  prefix + "_address",
			Component: "sensor",
			Payload: map[string]any{
				"name":                  fmt.Sprintf("%s Bus Address", display),
				"unique_id":             prefix + "_address",
				"state_topic":           vehicleTopic,
				"value_template":        "{{ value_json.address }}",
				"device":                device,
				"object_id":             prefix + "_address",
				"icon":                  "mdi:map-marker",
				"json_attributes_topic": vehicleTopic,
			},
		},
		{
			ObjectID:  prefix + "_heading",
			Component: "sensor",
			Payload: map[string]any{
				"name":           fmt.Sprintf("%s Bus Heading", display),
				"unique_id":      prefix + "_heading",
				"state_topic":    vehicleTopic,
				"value_template": "{{ value_json.heading }}",
				"device":         device,
				"object_id":      prefix + "_heading",
				"icon":           "mdi:compass",
			},
		},
		{
			ObjectID:  prefix + "_last_update",
			Component: "sensor",
			Payload: map[string]any{
				"name":           fmt.Sprintf("%s Bus Last Update", display),
				"unique_id":      prefix + "_last_update",
				"state_topic":    vehicleTopic,
				"value_template": "{{ value_json.updated_at }}",
				"device":         device,
				"object_id":      prefix + "_last_update",
				"device_class":   "timestamp",
				"icon":           "mdi:clock-outline",
			},
		},
		{
			ObjectID:  prefix + "_distance",
			Component: "sensor",
			Payload: map[string]any{
				"name":                  fmt.Sprintf("%s Distance to Stop", display),
				"unique_id":             prefix + "_distance",
				"state_topic":           distanceTopic,
				"value_template":        "{{ value_json.to_stop_miles }}",
				"unit_of_measurement":   "mi",
				"device":                device,
				"object_id":             prefix + "_distance",
				"icon":                  "mdi:map-marker-distance",
				"state_class":           "measurement",
				"json_attributes_topic": distanceTopic,
			},
		},
		{
			ObjectID:  prefix + "_eta",
			Component: "sensor",
			Payload: map[string]any{
				"name":                  fmt.Sprintf("%s ETA", display),
				"unique_id":             prefix + "_eta",
				"state_topic":           distanceTopic,
				"value_template":        "{{ value_json.eta_seconds }}",
				"unit_of_measurement":   "s",
				"device":                device,
				"object_id":             prefix + "_eta",
				"icon":                  "mdi:timer-sand",
				"state_class":           "measurement",
				"json_attributes_topic": distanceTopic,
			},
		},
		{
			ObjectID:  prefix + "_ignition",
			Component: "binary_sensor",
			Payload: map[string]any{
				"name":           fmt.Sprintf("%s Ignition", display),
				"unique_id":      prefix + "_ignition",
				"state_topic":    vehicleTopic,
				"value_template": "{{ value_json.ignition }}",
				"payload_on":     "true",
				"payload_off":    "false",
				"device":         device,
				"object_id":      prefix + "_ignition",
				"icon":           "mdi:engine",
			},
		},
		{
			ObjectID:  prefix + "_on_map",
			Component: "binary_sensor",
			Payload: map[string]any{
				"name":           fmt.Sprintf("%s On Map", display),
				"unique_id":      prefix + "_on_map",
				"state_topic":    vehicleTopic,
				"value_template": "{{ value_json.display_on_map }}",
				"payload_on":     "true",
				"payload_off":    "false",
				"device":         device,
				"object_id":      prefix + "_on_map",
				"icon":           "mdi:map-check",
			},
		},
		{
			ObjectID:  prefix + "_vehicle_available",
			Component: "binary_sensor",
			Payload: map[string]any{
				"name":           fmt.Sprintf("%s Vehicle Available", display),
				"unique_id":      prefix + "_vehicle_available",
				"state_topic":    vehicleTopic,
				"value_template": "{{ value_json.available }}",
				"payload_on":     "true",
				"payload_off":    "false",
				"device":         device,
				"object_id":      prefix + "_vehicle_available",
				"icon":           "mdi:bus-alert",
			},
		},
	}
}

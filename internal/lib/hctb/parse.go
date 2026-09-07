package hctb

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

func attr(se xml.StartElement, name string) string {
	for _, a := range se.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func parseYN(v string) bool {
	v = strings.ToUpper(strings.TrimSpace(v))
	return strings.HasPrefix(v, "Y")
}

func parseFloat(v string) float64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return f
}

func parseInt(v string) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

func parseLogTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, fmt.Errorf("empty log time")
	}
	layouts := []string{
		"1/2/2006 3:04:05 PM",
		"01/02/2006 3:04:05 PM",
		"1/2/2006 15:04:05",
		time.RFC3339,
	}
	var last error
	for _, layout := range layouts {
		t, err := time.Parse(layout, v)
		if err == nil {
			return t, nil
		}
		last = err
	}
	return time.Time{}, fmt.Errorf("parse log time %q: %w", v, last)
}

// ParseSchoolID extracts Customer/@ID from an s1100 SOAP response.
func ParseSchoolID(body []byte) (string, error) {
	dec := xml.NewDecoder(strings.NewReader(string(body)))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Customer" {
			continue
		}
		id := attr(se, "ID")
		if id == "" {
			return "", fmt.Errorf("Customer missing ID")
		}
		return id, nil
	}
	return "", fmt.Errorf("Customer element not found")
}

// ParseAccount extracts Account/Student/TimeOfDay from an s1157 SOAP response.
func ParseAccount(body []byte) (Account, error) {
	dec := xml.NewDecoder(strings.NewReader(string(body)))
	var out Account
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Account{}, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "Account":
			out.AccountID = attr(se, "ID")
		case "Student":
			out.Students = append(out.Students, Student{
				StudentID: attr(se, "EntityID"),
				FirstName: attr(se, "FirstName"),
				LastName:  attr(se, "LastName"),
			})
		case "TimeOfDay":
			out.Times = append(out.Times, TimeOfDay{
				ID:        attr(se, "ID"),
				Name:      attr(se, "Name"),
				BeginTime: attr(se, "BeginTime"),
				EndTime:   attr(se, "EndTime"),
			})
		}
	}
	if out.AccountID == "" {
		return Account{}, fmt.Errorf("Account ID not found (auth may have failed)")
	}
	return out, nil
}

// ParseStopInfo extracts VehicleLocation and StudentStop from an s1158 SOAP response.
func ParseStopInfo(body []byte) (StopInfo, error) {
	dec := xml.NewDecoder(strings.NewReader(string(body)))
	var out StopInfo
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return StopInfo{}, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "VehicleLocation":
			logTime, err := parseLogTime(attr(se, "LogTime"))
			if err != nil {
				logTime = time.Time{}
			}
			out.VehicleLocation = &VehicleLocation{
				Name:           attr(se, "Name"),
				Latitude:       parseFloat(attr(se, "Latitude")),
				Longitude:      parseFloat(attr(se, "Longitude")),
				LogTime:        logTime,
				Ignition:       parseYN(attr(se, "Ignition")),
				Latent:         parseYN(attr(se, "Latent")),
				TimeZoneOffset: parseInt(attr(se, "TimeZoneOffset")),
				Heading:        strings.TrimSpace(attr(se, "Heading")),
				Speed:          parseInt(attr(se, "Speed")),
				Address:        attr(se, "Address"),
				MessageCode:    parseInt(attr(se, "MessageCode")),
				DisplayOnMap:   parseYN(attr(se, "DisplayOnMap")),
			}
		case "StudentStop":
			out.StudentStops = append(out.StudentStops, StudentStop{
				Name:                     attr(se, "Name"),
				Latitude:                 parseFloat(attr(se, "Latitude")),
				Longitude:                parseFloat(attr(se, "Longitude")),
				StartTime:                attr(se, "StartTime"),
				StopType:                 attr(se, "StopType"),
				SubstituteVehicleName:    attr(se, "SubstituteVehicleName"),
				VehicleName:              attr(se, "VehicleName"),
				StopID:                   attr(se, "StopId"),
				ArrivalTime:              attr(se, "ArrivalTime"),
				TimeOfDayID:              attr(se, "TimeOfDayId"),
				VehicleID:                attr(se, "VehicleId"),
				ESN:                      attr(se, "Esn"),
				TierStartTime:            attr(se, "TierStartTime"),
				BusVisibilityStartOffset: parseInt(attr(se, "BusVisibilityStartOffset")),
			})
		}
	}
	return out, nil
}

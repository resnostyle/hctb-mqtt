package hctb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// APIError is a non-2xx Synovia response.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("hctb api status %d: %s", e.StatusCode, e.Body)
}

// Client talks to the unofficial Synovia / Here Comes The Bus SOAP API.
type Client struct {
	URL        string
	HTTPClient *http.Client
}

// New returns a Client pointed at the default Synovia endpoint.
func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{URL: DefaultURL, HTTPClient: httpClient}
}

func soapEnvelope(method string, params [][2]string) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">`)
	b.WriteString(`<soap:Body>`)
	b.WriteString(`<`)
	b.WriteString(method)
	b.WriteString(` xmlns="http://tempuri.org/">`)
	for _, p := range params {
		b.WriteString(`<`)
		b.WriteString(p[0])
		b.WriteString(`>`)
		b.WriteString(xmlEscape(p[1]))
		b.WriteString(`</`)
		b.WriteString(p[0])
		b.WriteString(`>`)
	}
	b.WriteString(`</`)
	b.WriteString(method)
	b.WriteString(`>`)
	b.WriteString(`</soap:Body></soap:Envelope>`)
	return []byte(b.String())
}

func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}

func (c *Client) request(ctx context.Context, method string, params [][2]string) ([]byte, error) {
	url := c.URL
	if url == "" {
		url = DefaultURL
	}
	body := soapEnvelope(method, params)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml")
	req.Header.Set("SOAPAction", "http://tempuri.org/ISynoviaApi/"+method)
	req.Header.Set("app-version", AppVersion)
	req.Header.Set("app-name", "hctb")
	req.Header.Set("client-version", AppVersion)
	req.Header.Set("User-Agent", "hctb/"+AppVersion+" App-Press/"+AppVersion)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Cookie", "SRV=prdweb1")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		snippet := string(raw)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Body: snippet}
	}
	return raw, nil
}

// GetSchoolID looks up the Synovia school/customer ID from a district school code.
func (c *Client) GetSchoolID(ctx context.Context, schoolCode string) (string, error) {
	raw, err := c.request(ctx, "s1100", [][2]string{{"P1", schoolCode}})
	if err != nil {
		return "", err
	}
	return ParseSchoolID(raw)
}

// GetParentInfo authenticates and returns account + students.
func (c *Client) GetParentInfo(ctx context.Context, schoolID, username, password string) (Account, error) {
	raw, err := c.request(ctx, "s1157", [][2]string{
		{"P1", schoolID},
		{"P2", username},
		{"P3", password},
		{"P4", "LookupItem_Source_Android"},
		{"P5", "Android"},
		{"P6", AppVersion},
		{"P7", ""},
	})
	if err != nil {
		return Account{}, err
	}
	return ParseAccount(raw)
}

// GetStopInfo returns stops and optional vehicle location for a student/route.
func (c *Client) GetStopInfo(ctx context.Context, schoolID, parentID, studentID, timeOfDayID string) (StopInfo, error) {
	raw, err := c.request(ctx, "s1158", [][2]string{
		{"P1", schoolID},
		{"P2", parentID},
		{"P3", studentID},
		{"P4", timeOfDayID},
		{"P5", "true"},
		{"P6", "false"},
		{"P7", "10"},
		{"P8", "14"},
		{"P9", "english"},
	})
	if err != nil {
		return StopInfo{}, err
	}
	return ParseStopInfo(raw)
}

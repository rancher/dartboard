/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Sample is a single (timestamp, value) pair from a Prometheus range vector.
type Sample struct {
	Timestamp time.Time
	Value     float64
}

// Series is one labelled time-series returned by a query_range call.
type Series struct {
	Labels  map[string]string
	Samples []Sample
}

type promResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
	Data      struct {
		ResultType string            `json:"resultType"`
		Result     []json.RawMessage `json:"result"`
	} `json:"data"`
}

type matrixEntry struct {
	Metric map[string]string `json:"metric"`
	Values [][2]any          `json:"values"`
}

// QueryRange runs an /api/v1/query_range request and returns one Series per
// matrix entry. Empty result sets are returned as an empty slice (no error).
func (c *Client) QueryRange(promQL string, start, end time.Time, step time.Duration) ([]Series, error) {
	q := url.Values{}
	q.Set("query", promQL)
	q.Set("start", strconv.FormatFloat(float64(start.UnixNano())/1e9, 'f', -1, 64))
	q.Set("end", strconv.FormatFloat(float64(end.UnixNano())/1e9, 'f', -1, 64))
	q.Set("step", strconv.FormatFloat(step.Seconds(), 'f', -1, 64))

	endpoint := c.baseURL + "/api/v1/query_range?" + q.Encode()
	resp, err := c.http.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("query_range GET failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read query_range body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query_range HTTP %d: %s", resp.StatusCode, string(body))
	}

	var pr promResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("decode query_range response: %w", err)
	}
	if pr.Status != "success" {
		return nil, fmt.Errorf("prometheus error %s: %s", pr.ErrorType, pr.Error)
	}
	if pr.Data.ResultType != "matrix" {
		return nil, fmt.Errorf("expected matrix result, got %q", pr.Data.ResultType)
	}

	out := make([]Series, 0, len(pr.Data.Result))
	for _, raw := range pr.Data.Result {
		var m matrixEntry
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("decode matrix entry: %w", err)
		}
		samples := make([]Sample, 0, len(m.Values))
		for _, v := range m.Values {
			ts, ok := v[0].(float64)
			if !ok {
				continue
			}
			valStr, ok := v[1].(string)
			if !ok {
				continue
			}
			f, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				continue
			}
			samples = append(samples, Sample{
				Timestamp: time.Unix(0, int64(ts*1e9)).UTC(),
				Value:     f,
			})
		}
		out = append(out, Series{Labels: m.Metric, Samples: samples})
	}
	return out, nil
}

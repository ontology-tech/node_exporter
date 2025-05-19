// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !noneo
// +build !noneo

package collector

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	neoURL = kingpin.Flag("collector.neo.rpc", "neo rpc url").Default("http://127.0.0.1:10332").Envar("NEORPC_URL").String()
)

type neoCollector struct {
	neoHeight typedDesc
	logger    *slog.Logger
}

func init() {
	registerCollector("neo", defaultEnabled, NewNeoCollector)
}

// NewTimeCollector returns a new Collector exposing the current system time in
// seconds since epoch.
func NewNeoCollector(logger *slog.Logger) (Collector, error) {
	const subsystem = "neo"
	return &neoCollector{
		neoHeight: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "height"),
			"neo node block height",
			nil, nil,
		), prometheus.GaugeValue},
		logger: logger,
	}, nil
}

type NeoHeightResponse struct {
	ID      int    `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Localrootindex     int64 `json:"localrootindex"`
		Validatedrootindex int64 `json:"validatedrootindex"`
	} `json:"result"`
}

func (c *neoCollector) Update(ch chan<- prometheus.Metric) error {
	ctx, cf := context.WithTimeout(context.Background(), time.Second*3)
	defer cf()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *neoURL, strings.NewReader(`{ "jsonrpc": "2.0", "method": "getstateheight", "params": [], "id": 1 }`))
	if err != nil {
		c.logger.Info("can not create neo height req")
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.logger.Error("can not get valid response from rpc: ", "rpc", *neoURL)
		return nil
	}

	defer resp.Body.Close()
	var r NeoHeightResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		c.logger.Error("can not parse neo response")
		return nil
	}
	ch <- c.neoHeight.mustNewConstMetric(float64(r.Result.Localrootindex))

	return nil
}

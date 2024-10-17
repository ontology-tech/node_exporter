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

//go:build !ontology
// +build !ontology

package collector

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
)

var ontologyRpc = kingpin.Flag("collector.ontology.rpc", "specify ontology node rpc target, default to http://127.0.0.1:20336").Default("http://127.0.0.1:20336").String()

type ontologyCollector struct {
	height  typedDesc
	logger  log.Logger
}

func init() {
	registerCollector("ontology", defaultEnabled, NewOntologyCollector)
}

// NewTimeCollector returns a new Collector exposing the current system time in
// seconds since epoch.
func NewOntologyCollector(logger log.Logger) (Collector, error) {
	const subsystem = "consensus"
	return &ontologyCollector{
		height: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "height"),
			"ontology blockchain consensus node height",
			[]string{"version"}, nil,
		), prometheus.GaugeValue},
		logger: logger,
	}, nil
}

type OntologyHeightResp struct {
	Desc    string `json:"desc"`
	Error   int    `json:"error"`
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  uint64 `json:"result"`
}


type OntologyVersionResp struct {
	Desc    string `json:"desc"`
	Error   int    `json:"error"`
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  string `json:"result"`
}

func getVersion() (string, error) {
	ctx, cf := context.WithTimeout(context.Background(), time.Second*3)
	defer cf()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *ontologyRpc, strings.NewReader(`{"jsonrpc":"2.0","id":"1","method":"getversion","params":[]}`))
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var r OntologyVersionResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}

	return r.Result, nil
}

func getHeight() (uint64, error) {
	ctx, cf := context.WithTimeout(context.Background(), time.Second*3)
	defer cf()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *ontologyRpc, strings.NewReader(`{"jsonrpc":"2.0","id":"1","method":"getblockcount","params":[]}`))
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var r OntologyHeightResp

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, err
	}

	return r.Result, nil
}

func (c *ontologyCollector) Update(ch chan<- prometheus.Metric) error {
	height, err := getHeight()
	if err != nil {
		// let alert manager detect this error
		ch <- c.height.mustNewConstMetric(float64(0.0))
		return err
	}
	
	version, err := getVersion()
	if err != nil {
		return err
	}

	level.Debug(c.logger).Log("target", *ontologyRpc, "hight", height, "version", version)
	ch <- c.height.mustNewConstMetric(float64(height), version)

	return nil
}

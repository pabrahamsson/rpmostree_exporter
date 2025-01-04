package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/coreos/rpmostree-client-go/pkg/client"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
)

const (
	namespace = "rpmostree"
)

var (
	clientId      = "rpmostree-exporter"
	rpmostreeInfo = prometheus.NewDesc(prometheus.BuildFQName(namespace, "version", "info"), "rpm-ostree info.", []string{"release_date", "version"}, nil)
	rpmostreeUp   = prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "up"), "Was the last scrape of rpm-ostree successful.", nil, nil)
)

type Exporter struct {
	client           client.Client
	mutex            sync.RWMutex
	up               prometheus.Gauge
	totalScrapes     prometheus.Counter
	updatesAvailable prometheus.Gauge
	logger           log.Logger
}

func NewExporter(logger log.Logger) (*Exporter, error) {
	return &Exporter{
		client: client.NewClient(clientId),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_scrapes_total",
			Help:      "Current total rpm-ostree scrapes.",
		}),
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the last scrape of rpm-ostree successful.",
		}),
		updatesAvailable: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "updates_available",
			Help:      "Is there a staged rpm-ostree deployment available.",
		}),
		logger: logger,
	}, nil
}

// Describe describes all the metrics ever exported by the rpmostree exporter.
// It implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- rpmostreeInfo
	ch <- rpmostreeUp
	ch <- e.totalScrapes.Desc()
}

// Collect fetches the metrics and delivers them as Prometheus metrics.
// It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	up := e.scrape(ch)

	ch <- prometheus.MustNewConstMetric(rpmostreeUp, prometheus.GaugeValue, up)
	ch <- e.totalScrapes
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) (up float64) {
	e.totalScrapes.Inc()
	var err error
	var packages, booted_version, staged_version string
	value := float64(0)

	if err = fetchUpdates(e.client, &packages, &booted_version, &staged_version); err != nil {
		return value
	}

	if staged_version != "" {
		value = 1
	}
	desc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "updates_available"), "Is there a staged rpm-ostree deployment available.", []string{"booted", "staged", "packages"}, nil)
	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, []string{booted_version, staged_version, packages}...)

	return value
}

type Diff struct {
	OstreeCommitFrom string              `json:"ostree-commit-from"`
	OstreeCommitTo   string              `json:"ostree-commit-to"`
	PkgDiff          [][]json.RawMessage `json:"pkgdiff"`
}

type Package struct {
	PreviousPackage []string
	NewPackage      []string
}

func newCmd(clientid string, args ...string) *exec.Cmd {
	r := exec.Command("rpm-ostree", args...)
	r.Env = append(r.Env, "RPMOSTREE_CLIENT_ID", clientid)
	return r
}

func fetchUpdates(client client.Client, packages, booted_version, staged_version *string) error {
	status, err := client.QueryStatus()
	if err != nil {
		return err
	}

	booted, _ := status.GetBootedDeployment()
	staged := status.GetStagedDeployment()
	*booted_version = booted.Version
	if staged != nil {
		*staged_version = staged.Version
		c := newCmd(clientId, "db", "diff", "--format=json", booted.GetBaseChecksum(), staged.GetBaseChecksum())
		buf, err := c.Output()
		if err != nil {
			return err
		}

		var diff Diff
		if err := json.Unmarshal(buf, &diff); err != nil {
			return err
		}

		pkgs := []string{"\n"}
		for _, pkg := range diff.PkgDiff {
			var p Package
			if err = json.Unmarshal(pkg[2], &p); err != nil {
				return err
			}
			pkgs = append(pkgs, fmt.Sprintf("%s: %s -> %s", p.PreviousPackage[0], p.PreviousPackage[1], p.NewPackage[1]))
		}
		*packages = strings.Join(pkgs, "\n")
	}

	return nil
}

func main() {
	promlogConfig := &promlog.Config{}
	logger := promlog.New(promlogConfig)
	level.Info(logger).Log("msg", "Schtarrrrting!!")
	exporter, err := NewExporter(logger)
	if err != nil {
		level.Error(logger).Log("msg", "Error creating an exporter", "err", err)
		os.Exit(1)
	}
	prometheus.MustRegister(exporter)

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8888", nil)
}
package metrics

import (
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/artefactual/archivematica/hack/ccp/internal/version"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// We need to balance reasonably accurate tracking with high cardinality
	// here, as this is used with script_name labels and there are already over
	// 100 scripts.
	taskDurationPackages = []float64{
		2.0,
		5.0,
		10.0,
		20.0,
		30.0,
		60.0,
		120.0,  // 2 min
		300.0,  // 5 min
		600.0,  // 10 min
		1800.0, // 30 min
		3600.0, // 1 hour
		math.Inf(1),
	}
	packageTypes = []string{"Transfer", "SIP", "DIP"}
)

// Metrics is a container of application metrics exposed via Prometheus.
type Metrics struct {
	reg *prometheus.Registry

	// ArchivematicaInfo captures the version of Archivematica.
	ArchivematicaInfo *prometheus.GaugeVec

	// EnvironmentInfo captures environment information.
	EnvironmentInfo *prometheus.GaugeVec

	// GearmanActiveJobsGauge tracks the number of job batches that are
	// currently being processed by Gearman.
	GearmanActiveJobsGauge prometheus.Gauge

	// GearmanPendingJobsGauge tracks the number of job batches that are waiting
	// to be submitted to Gearman.
	GearmanPendingJobsGauge prometheus.Gauge

	// TaskCounter counts the number of tasks that have been completed.
	TaskCounter *prometheus.CounterVec

	// TaskSuccessTimestamp records the timestamp when a task successfully
	// completes.
	TaskSuccessTimestamp *prometheus.GaugeVec

	// TaskDurationHistogram measures the duration of tasks.
	TaskDurationHistogram *prometheus.HistogramVec

	// ActivePackageGauge tracks the number of active packages being processed.
	ActivePackageGauge prometheus.Gauge

	// ActiveJobsGauge tracks the number of active jobs currently being
	// processed.
	//
	// TODO: unused in CCP because we don't throttle jobs belonging to packages.
	ActiveJobsGauge prometheus.Gauge

	// JobQueueLengthGauge tracks the current length of the job queue.
	//
	// TODO: unused in CCP because we don't throttle jobs belonging to packages.
	JobQueueLengthGauge prometheus.Gauge

	// PackageQueueLengthGauge tracks the length of the package queue, segmented
	// by package type (DIP, SIP, Transfer).
	PackageQueueLengthGauge *prometheus.GaugeVec
}

func NewMetrics(wf *workflow.Document) *Metrics {
	m := &Metrics{
		reg: prometheus.NewRegistry(),
		ArchivematicaInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "archivematica_version",
			Help: "Archivematica version info",
		}, []string{"version"}),
		EnvironmentInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "environment_variables",
			Help: "Environment Variables",
		}, []string{"key", "value"}),
		GearmanActiveJobsGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mcpserver_gearman_active_jobs",
			Help: "Number of gearman jobs currently being processed",
		}),
		GearmanPendingJobsGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mcpserver_gearman_pending_jobs",
			Help: "Number of gearman jobs pending submission",
		}),
		TaskCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "mcpserver_task_total",
			Help: "Number of tasks processed, labeled by task group, task name",
		}, []string{"task_group_name", "task_name"}),
		TaskSuccessTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "mcpserver_task_success_timestamp",
			Help: "Most recent successfully processed task, labeled by task group, task name",
		}, []string{"task_group_name", "task_name"}),
		TaskDurationHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "mcpserver_task_duration_seconds",
			Help:    "Histogram of task processing durations in seconds, labeled by script name",
			Buckets: taskDurationPackages,
		}, []string{"script_name"}),
		ActivePackageGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mcpserver_active_packages",
			Help: "Number of currently active packages",
		}),
		ActiveJobsGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mcpserver_active_jobs",
			Help: "Number of currently active jobs",
		}),
		JobQueueLengthGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mcpserver_active_package_job_queue_length",
			Help: "Number of queued jobs related to currently active packages",
		}),
		PackageQueueLengthGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "mcpserver_package_queue_length",
			Help: "Number of queued packages",
		}, []string{"package_type"}),
	}

	m.initLabels(wf)

	m.reg.MustRegister(
		m.ArchivematicaInfo,
		m.EnvironmentInfo,
		m.GearmanActiveJobsGauge,
		m.GearmanPendingJobsGauge,
		m.TaskCounter,
		m.TaskSuccessTimestamp,
		m.TaskDurationHistogram,
		m.ActivePackageGauge,
		m.ActiveJobsGauge,
		m.JobQueueLengthGauge,
		m.PackageQueueLengthGauge,
		collectors.NewBuildInfoCollector(),
	)

	return m
}

func (m *Metrics) initLabels(wf *workflow.Document) {
	m.ArchivematicaInfo.WithLabelValues(version.Version()).Set(1)

	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		m.EnvironmentInfo.WithLabelValues(pair[0], pair[1]).Set(1)
	}

	for _, pt := range packageTypes {
		m.PackageQueueLengthGauge.With(prometheus.Labels{"package_type": pt}).Set(0)
	}

	if wf != nil {
		for _, ln := range wf.Links {
			linkGroup := ln.Group.String()
			linkDesc := ln.Description.String()
			var scriptName string
			if config, ok := ln.Config.(workflow.LinkStandardTaskConfig); ok {
				scriptName = config.Execute
			}

			m.TaskCounter.WithLabelValues(linkGroup, linkDesc)
			m.TaskSuccessTimestamp.WithLabelValues(linkGroup, linkDesc)
			m.TaskDurationHistogram.WithLabelValues(scriptName)
		}
	}
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(
		m.reg,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
			Registry:          m.reg,
		},
	)
}

func (m *Metrics) TaskCompleted(startedAt, finishedAt time.Time, scriptName, linkGroup, linkDesc string) {
	if finishedAt.IsZero() {
		return
	}

	m.TaskCounter.WithLabelValues(linkGroup, linkDesc).Inc()
	m.TaskSuccessTimestamp.WithLabelValues(linkGroup, linkDesc).SetToCurrentTime()
	m.TaskDurationHistogram.WithLabelValues(scriptName).Observe(finishedAt.Sub(startedAt).Seconds())
}

package agent

import (
	"context"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dockerstatsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/fluentforwardreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8sclusterreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8seventsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kubeletstatsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/loggingexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.uber.org/zap"
)

// KubeAgent implements Agent interface for Kubernetes
type KubeAgent struct {
	apiKey string
	target string

	enableSytheticMonitoring bool
	configCheckInterval      string

	apiURLForConfigCheck string

	logger         *zap.Logger
	dockerEndpoint string
}

// KubeOptions takes in various options for KubeAgent
type KubeOptions func(h *KubeAgent)

// WithKubeAgentApiKey sets api key for interacting with
// the Middleware backend
func WithKubeAgentApiKey(key string) KubeOptions {
	return func(h *KubeAgent) {
		h.apiKey = key
	}
}

// WithKubeAgentTarget sets target URL for sending insights
// to the Middlware backend.
func WithKubeAgentTarget(t string) KubeOptions {
	return func(h *KubeAgent) {
		h.target = t
	}
}

// WithKubeAgentEnableSyntheticMonitoring enables synthetic
// monitoring to be performed from the agent.
// Note: This is currently not supported in KubeAgent
func WithKubeAgentEnableSyntheticMonitoring(e bool) KubeOptions {
	return func(h *KubeAgent) {
		h.enableSytheticMonitoring = e
	}
}

// WithKubeAgentConfigCheckInterval sets the duration for checking with
// the Middleware backend for configuration update.
func WithKubeAgentConfigCheckInterval(c string) KubeOptions {
	return func(h *KubeAgent) {
		h.configCheckInterval = c
	}
}

// WithKubeAgentApiURLForConfigCheck sets the URL for the periodic
// configuration check.
func WithKubeAgentApiURLForConfigCheck(u string) KubeOptions {
	return func(h *KubeAgent) {
		h.apiURLForConfigCheck = u
	}
}

// WithKubeAgentLogger sets the logger to be used with agent logs
func WithKubeAgentLogger(logger *zap.Logger) KubeOptions {
	return func(h *KubeAgent) {
		h.logger = logger
	}
}

// WithKubeAgentDockerEndpoint sets the endpoint for docker so that
// the agent can figure out if it needs to send docker logs & metrics.
func WithKubeAgentDockerEndpoint(endpoint string) KubeOptions {
	return func(h *KubeAgent) {
		h.dockerEndpoint = endpoint
	}
}

// NewKubeAgent returns new agent for Kubernetes with given options.
func NewKubeAgent(opts ...KubeOptions) *KubeAgent {
	var cfg KubeAgent
	for _, apply := range opts {
		apply(&cfg)
	}

	if cfg.logger == nil {
		cfg.logger, _ = zap.NewProduction()
	}

	return &cfg
}

// GetUpdatedYAMLPath gets the correct otel configuration file.
func (k *KubeAgent) GetUpdatedYAMLPath() (string, error) {
	yamlPath := "/app/otel-config.yaml"
	dockerSocketPath := strings.Split(k.dockerEndpoint, "//")

	if len(dockerSocketPath) != 2 || !isSocketFn(dockerSocketPath[1]) {
		yamlPath = "/app/otel-config-nodocker.yaml"
	}

	return yamlPath, nil
}

// GetFactories get otel factories for KubeAgent
func (k *KubeAgent) GetFactories(ctx context.Context) (otelcol.Factories, error) {
	var err error
	factories := otelcol.Factories{}
	factories.Extensions, err = extension.MakeFactoryMap(
	//healthcheckextension.NewFactory(),
	// frontend.NewAuthFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	factories.Receivers, err = receiver.MakeFactoryMap([]receiver.Factory{
		otlpreceiver.NewFactory(),
		fluentforwardreceiver.NewFactory(),
		filelogreceiver.NewFactory(),
		dockerstatsreceiver.NewFactory(),
		hostmetricsreceiver.NewFactory(),
		k8sclusterreceiver.NewFactory(),
		k8seventsreceiver.NewFactory(),
		kubeletstatsreceiver.NewFactory(),
		prometheusreceiver.NewFactory(),
	}...)
	if err != nil {
		return otelcol.Factories{}, err
	}

	factories.Exporters, err = exporter.MakeFactoryMap([]exporter.Factory{
		loggingexporter.NewFactory(),
		otlpexporter.NewFactory(),
		otlphttpexporter.NewFactory(),
	}...)
	if err != nil {
		return otelcol.Factories{}, err
	}

	factories.Processors, err = processor.MakeFactoryMap([]processor.Factory{
		// frontend.NewProcessorFactory(),
		batchprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
		filterprocessor.NewFactory(),
		attributesprocessor.NewFactory(),
		resourceprocessor.NewFactory(),
		resourcedetectionprocessor.NewFactory(),
		k8sattributesprocessor.NewFactory(),
	}...)
	if err != nil {
		return otelcol.Factories{}, err
	}

	return factories, nil
}
package main

import (
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
)

const (
	severityLevel      = 2 //TODO: decide on the severity of the service
	healthCheckTimeout = 10 * time.Second
)

type HealthService struct {
	config       *HealthConfig
	healthChecks []fthealth.Check
	gtgChecks    []gtg.StatusChecker
}

type HealthConfig struct {
	appSystemCode  string
	appName        string
	appDescription string
}

func NewHealthService(appSystemCode string, appName string, appDescription string) *HealthService {
	hc := &HealthService{
		config: &HealthConfig{
			appSystemCode:  appSystemCode,
			appName:        appName,
			appDescription: appDescription,
		},
	}
	hc.healthChecks = []fthealth.Check{hc.sampleCheck()}

	check := func() gtg.Status {
		return gtgCheck(hc.sampleChecker)
	}
	hc.gtgChecks = append(hc.gtgChecks, check)

	return hc
}

func (hs *HealthService) Health() fthealth.HC {
	return &fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  hs.config.appSystemCode,
			Name:        hs.config.appName,
			Description: hs.config.appDescription,
			Checks:      hs.healthChecks,
		},
		Timeout: healthCheckTimeout,
	}
}

func (hs *HealthService) sampleCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Sample healthcheck has no impact",
		Name:             "Sample healthcheck",
		PanicGuide:       "https://runbooks.ftops.tech/combined-content-search",
		Severity:         severityLevel,
		TechnicalSummary: "Sample healthcheck has no technical details",
		Checker:          hs.sampleChecker,
	}
}

func (hs *HealthService) sampleChecker() (string, error) {
	return "Sample is healthy", nil
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}

	return gtg.Status{GoodToGo: true}
}

func (hs *HealthService) GTG() gtg.Status {
	return gtg.FailFastParallelCheck(hs.gtgChecks)()
}

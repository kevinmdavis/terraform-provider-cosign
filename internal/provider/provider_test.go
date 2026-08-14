package provider

import (
	"testing"

	"github.com/chainguard-dev/terraform-provider-cosign/pkg/private/secant"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cosign": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

// newSecantMetricsRegistry registers secant's collectors into a fresh registry
// so a test can read sign/attest duration observations back after signing.
func newSecantMetricsRegistry(t *testing.T) *prometheus.Registry {
	t.Helper()
	reg := prometheus.NewRegistry()
	if err := secant.RegisterMetrics(reg); err != nil {
		t.Fatalf("registering secant metrics: %v", err)
	}
	return reg
}

// operationDurationCount returns how many secant_operation_duration_seconds
// samples g has gathered for the given operation, across all signature formats.
func operationDurationCount(t *testing.T, g prometheus.Gatherer, operation string) uint64 {
	t.Helper()
	families, err := g.Gather()
	if err != nil {
		t.Fatalf("gathering metrics: %v", err)
	}
	var count uint64
	for _, mf := range families {
		if mf.GetName() != "secant_operation_duration_seconds" {
			continue
		}
		for _, m := range mf.GetMetric() {
			if metricLabel(m, "operation") == operation {
				count += m.GetHistogram().GetSampleCount()
			}
		}
	}
	return count
}

func metricLabel(m *dto.Metric, name string) string {
	for _, l := range m.GetLabel() {
		if l.GetName() == name {
			return l.GetValue()
		}
	}
	return ""
}

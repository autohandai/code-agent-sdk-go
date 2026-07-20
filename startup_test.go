package autohand

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const packageInitProbeEnv = "AUTOHAND_GO_PACKAGE_INIT_PROBE"

func TestMain(m *testing.M) {
	if os.Getenv(packageInitProbeEnv) == "1" {
		fmt.Printf("PACKAGE_INIT_NS=%d\n", packageInitializationElapsed.Nanoseconds())
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func percentile95(samples []time.Duration) time.Duration {
	ordered := append([]time.Duration(nil), samples...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return ordered[(95*len(ordered)+99)/100-1]
}

func median(samples []time.Duration) time.Duration {
	ordered := append([]time.Duration(nil), samples...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return ordered[len(ordered)/2]
}

func maxDuration(samples []time.Duration) time.Duration {
	maximum := time.Duration(0)
	for _, sample := range samples {
		if sample > maximum {
			maximum = sample
		}
	}
	return maximum
}

func measureGoPackageInitialization(t *testing.T) time.Duration {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^$")
	cmd.Env = append(os.Environ(), packageInitProbeEnv+"=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("package-initialization probe: %v: %s", err, output)
	}
	line := strings.TrimSpace(string(output))
	value := strings.TrimPrefix(line, "PACKAGE_INIT_NS=")
	nanos, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		t.Fatalf("parse package-initialization probe %q: %v", line, err)
	}
	return time.Duration(nanos)
}

func measureGoUsableStart(t *testing.T, cli string) time.Duration {
	t.Helper()
	sdk := NewSDK(&Config{CLIPath: cli, Timeout: 2_000})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := time.Now()
	if err := sdk.Start(ctx); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	if err := sdk.Stop(); err != nil {
		t.Fatal(err)
	}
	return elapsed
}

func measureGoFixtureFirstRPC(t *testing.T, cli string) time.Duration {
	t.Helper()
	config := &Config{CLIPath: cli, Timeout: 2_000}
	client := NewRPCClient(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := time.Now()
	if err := client.Start(ctx, config); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetState(ctx); err != nil {
		_ = client.Stop()
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	if err := client.Stop(); err != nil {
		t.Fatal(err)
	}
	return elapsed
}

func TestStartupBudgets(t *testing.T) {
	if testing.Short() {
		t.Skip("55-process startup budget")
	}
	cli := writeLifecycleCLI(t, false)
	for range 5 {
		_ = measureGoPackageInitialization(t)
		_ = measureGoUsableStart(t, cli)
		_ = measureGoFixtureFirstRPC(t, cli)
	}
	packageLoad := make([]time.Duration, 0, 50)
	sdkStartReturn := make([]time.Duration, 0, 50)
	fixtureFirstRPC := make([]time.Duration, 0, 50)
	for range 50 {
		packageLoad = append(packageLoad, measureGoPackageInitialization(t))
		sdkStartReturn = append(sdkStartReturn, measureGoUsableStart(t, cli))
		fixtureFirstRPC = append(fixtureFirstRPC, measureGoFixtureFirstRPC(t, cli))
	}
	packageP95 := percentile95(packageLoad)
	sdkStartP95 := percentile95(sdkStartReturn)
	fixtureP95 := percentile95(fixtureFirstRPC)
	type metricResult struct {
		Samples  int     `json:"samples"`
		MedianMS float64 `json:"medianMs"`
		P95MS    float64 `json:"p95Ms"`
		MaxMS    float64 `json:"maxMs"`
		Passed   bool    `json:"passed"`
	}
	type benchmarkResult struct {
		Language string                  `json:"language"`
		BudgetMS int                     `json:"budgetMs"`
		Metrics  map[string]metricResult `json:"metrics"`
		Passed   bool                    `json:"passed"`
	}
	toMS := func(value time.Duration) float64 { return float64(value) / float64(time.Millisecond) }
	metrics := map[string]metricResult{
		"publicImportMs": {
			Samples: 50, MedianMS: toMS(median(packageLoad)), P95MS: toMS(packageP95),
			MaxMS: toMS(maxDuration(packageLoad)), Passed: packageP95 < 50*time.Millisecond,
		},
		"sdkStartReturnMs": {
			Samples: 50, MedianMS: toMS(median(sdkStartReturn)), P95MS: toMS(sdkStartP95),
			MaxMS: toMS(maxDuration(sdkStartReturn)), Passed: sdkStartP95 < 50*time.Millisecond,
		},
		"fixtureSpawnToFirstRpcMs": {
			Samples: 50, MedianMS: toMS(median(fixtureFirstRPC)), P95MS: toMS(fixtureP95),
			MaxMS: toMS(maxDuration(fixtureFirstRPC)), Passed: fixtureP95 < 50*time.Millisecond,
		},
	}
	report := benchmarkResult{
		Language: "go",
		BudgetMS: 50,
		Metrics:  metrics,
		Passed:   metrics["publicImportMs"].Passed && metrics["sdkStartReturnMs"].Passed && metrics["fixtureSpawnToFirstRpcMs"].Passed,
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(encoded))
	if packageP95 >= 50*time.Millisecond {
		t.Errorf("cold package-load p95 %s exceeds 50ms", packageP95)
	}
	if sdkStartP95 >= 50*time.Millisecond {
		t.Errorf("SDK.Start p95 %s exceeds 50ms", sdkStartP95)
	}
	if fixtureP95 >= 50*time.Millisecond {
		t.Errorf("fixture spawn-to-first-RPC p95 %s exceeds 50ms", fixtureP95)
	}
}

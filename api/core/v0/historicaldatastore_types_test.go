package v0

import (
	"encoding/json"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
)

func TestHistoricalAutoScalerTypeConstants(t *testing.T) {
	tests := map[AutoScalerType]string{
		CONCURRENCY: "concurrency",
		RPS:         "rps",
		CPU:         "cpu",
		MEMORY:      "memory",
	}
	for got, want := range tests {
		if string(got) != want {
			t.Errorf("AutoScalerType constant %v = %q; want %q", got, string(got), want)
		}
	}
}

func TestHistoricalPostgresConfigJSON(t *testing.T) {
	pc := PostgresConfig{
		Host:       "db.example.com",
		Port:       5432,
		User:       "user",
		Database:   "dbname",
		SecretName: "secret",
	}
	data, err := json.Marshal(pc)
	if err != nil {
		t.Fatalf("failed to marshal PostgresConfig: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}
	if m["host"] != pc.Host {
		t.Errorf("host = %v; want %v", m["host"], pc.Host)
	}
	if int(m["port"].(float64)) != pc.Port {
		t.Errorf("port = %v; want %v", m["port"], pc.Port)
	}
	var pc2 PostgresConfig
	if err := json.Unmarshal(data, &pc2); err != nil {
		t.Fatalf("failed to unmarshal into struct: %v", err)
	}
	if pc2 != pc {
		t.Errorf("PostgresConfig round-trip = %+v; want %+v", pc2, pc)
	}
}

func TestHistoricalDataStoreAutoScalingJSON(t *testing.T) {
	min := 1
	max := 5
	target := 3
	percent := 75
	parallelism := 2
	hdsa := HistoricalDataStoreAutoScaling{
		MinScale:                    &min,
		MaxScale:                    &max,
		Target:                      &target,
		TargetUtilizationPercentage: &percent,
		Parallelism:                 &parallelism,
		Metric:                      RPS,
	}
	data, err := json.Marshal(hdsa)
	if err != nil {
		t.Fatalf("failed to marshal HistoricalDataStoreAutoScaling: %v", err)
	}
	var hdsa2 HistoricalDataStoreAutoScaling
	if err := json.Unmarshal(data, &hdsa2); err != nil {
		t.Fatalf("failed to unmarshal HistoricalDataStoreAutoScaling: %v", err)
	}
	if *hdsa2.MinScale != min || *hdsa2.MaxScale != max ||
		*hdsa2.Target != target || *hdsa2.TargetUtilizationPercentage != percent ||
		*hdsa2.Parallelism != parallelism || hdsa2.Metric != RPS {
		t.Errorf("HistoricalDataStoreAutoScaling round-trip = %+v; want %+v", hdsa2, hdsa)
	}
}

func TestHistoricalDataStoreSpecJSONOmitEmpty(t *testing.T) {
	timeout := 20
	spec := HistoricalDataStoreSpec{
		Timeout: &timeout,
		Postgres: PostgresConfig{
			Host:       "h",
			Port:       1,
			User:       "u",
			Database:   "d",
			SecretName: "s",
		},
	}
	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("failed to marshal HistoricalDataStoreSpec: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"timeout":20`) {
		t.Errorf("JSON = %s; want timeout field", s)
	}
	if !strings.Contains(s, `"autoScaling":{}`) {
		t.Errorf("JSON = %s; want autoScaling field as empty object when empty", s)
	}
	if !strings.Contains(s, `"resources":{}`) {
		t.Errorf("JSON = %s; want resources field as empty object when empty", s)
	}
	if !strings.Contains(s, `"dispatcherResources":{}`) {
		t.Errorf("JSON = %s; want dispatcherResources field as empty object when empty", s)
	}
}

func TestHistoricalDataStoreAndListJSON(t *testing.T) {
	hds := HistoricalDataStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hist",
			Namespace: "ns",
		},
		Spec: HistoricalDataStoreSpec{
			Postgres: PostgresConfig{
				Host:       "h",
				Port:       1,
				User:       "u",
				Database:   "d",
				SecretName: "s",
			},
		},
	}
	data, err := json.Marshal(hds)
	if err != nil {
		t.Fatalf("failed to marshal HistoricalDataStore: %v", err)
	}
	var hds2 HistoricalDataStore
	if err := json.Unmarshal(data, &hds2); err != nil {
		t.Fatalf("failed to unmarshal HistoricalDataStore: %v", err)
	}
	if hds2.Name != hds.Name || hds2.Namespace != hds.Namespace {
		t.Errorf("round-trip metadata mismatch: %+v vs %+v", hds2.ObjectMeta, hds.ObjectMeta)
	}
	list := HistoricalDataStoreList{
		Items: []HistoricalDataStore{hds},
	}
	dataList, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("failed to marshal HistoricalDataStoreList: %v", err)
	}
	var list2 HistoricalDataStoreList
	if err := json.Unmarshal(dataList, &list2); err != nil {
		t.Fatalf("failed to unmarshal HistoricalDataStoreList: %v", err)
	}
	if len(list2.Items) != 1 || list2.Items[0].Name != "hist" {
		t.Errorf("round-trip list mismatch: %+v", list2.Items)
	}
}

func TestHistoricalAddToScheme(t *testing.T) {
	sch := kruntime.NewScheme()
	if err := AddToScheme(sch); err != nil {
		t.Fatalf("failed to add to scheme: %v", err)
	}
}

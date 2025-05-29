package v0

import (
	"encoding/json"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
)

func TestAutoScalerTypeConstants(t *testing.T) {
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

func TestPostgresConfigJSON(t *testing.T) {
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
	// round-trip
	var pc2 PostgresConfig
	if err := json.Unmarshal(data, &pc2); err != nil {
		t.Fatalf("failed to unmarshal into struct: %v", err)
	}
	if pc2 != pc {
		t.Errorf("PostgresConfig round-trip = %+v; want %+v", pc2, pc)
	}
}

func TestEventStoreAutoScalingJSON(t *testing.T) {
	min := 1
	max := 5
	target := 3
	percent := 75
	parallelism := 2
	esc := EventStoreAutoScaling{
		MinScale:                    &min,
		MaxScale:                    &max,
		Target:                      &target,
		TargetUtilizationPercentage: &percent,
		Parallelism:                 &parallelism,
		Metric:                      CPU,
	}
	data, err := json.Marshal(esc)
	if err != nil {
		t.Fatalf("failed to marshal EventStoreAutoScaling: %v", err)
	}
	var esc2 EventStoreAutoScaling
	if err := json.Unmarshal(data, &esc2); err != nil {
		t.Fatalf("failed to unmarshal EventStoreAutoScaling: %v", err)
	}
	if *esc2.MinScale != min || *esc2.MaxScale != max ||
		*esc2.Target != target || *esc2.TargetUtilizationPercentage != percent ||
		*esc2.Parallelism != parallelism || esc2.Metric != CPU {
		t.Errorf("EventStoreAutoScaling round-trip = %+v; want %+v", esc2, esc)
	}
}

func TestEventStoreSpecJSONOmitEmpty(t *testing.T) {
    timeout := 10
    spec := EventStoreSpec{
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
        t.Fatalf("failed to marshal EventStoreSpec: %v", err)
    }

    s := string(data)
    if !strings.Contains(s, `"timeout":10`) {
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


func TestEventStoreAndListJSON(t *testing.T) {
	es := EventStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: EventStoreSpec{
			Postgres: PostgresConfig{
				Host:       "h",
				Port:       1,
				User:       "u",
				Database:   "d",
				SecretName: "s",
			},
		},
	}
	data, err := json.Marshal(es)
	if err != nil {
		t.Fatalf("failed to marshal EventStore: %v", err)
	}
	var es2 EventStore
	if err := json.Unmarshal(data, &es2); err != nil {
		t.Fatalf("failed to unmarshal EventStore: %v", err)
	}
	if es2.Name != es.Name || es2.Namespace != es.Namespace {
		t.Errorf("round-trip metadata mismatch: %+v vs %+v", es2.ObjectMeta, es.ObjectMeta)
	}
	list := EventStoreList{
		Items: []EventStore{es},
	}
	dataList, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("failed to marshal EventStoreList: %v", err)
	}
	var list2 EventStoreList
	if err := json.Unmarshal(dataList, &list2); err != nil {
		t.Fatalf("failed to unmarshal EventStoreList: %v", err)
	}
	if len(list2.Items) != 1 || list2.Items[0].Name != "foo" {
		t.Errorf("round-trip list mismatch: %+v", list2.Items)
	}
}

func TestAddToScheme(t *testing.T) {
	sch := kruntime.NewScheme()
	if err := AddToScheme(sch); err != nil {
		t.Fatalf("failed to add to scheme: %v", err)
	}
}

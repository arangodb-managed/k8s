package k8s

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os/exec"
	"testing"

	"github.com/ericchiang/k8s/api/v1"
)

func newTestClient(t *testing.T) *Client {
	cmd := exec.Command("kubectl", "config", "view", "-o", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'kubectl config view -o json': %v %s", err, out)
	}

	config := new(Config)
	if err := json.Unmarshal(out, config); err != nil {
		t.Fatalf("parse kubeconfig: %v '%s'", err, out)
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return client
}

func newName() string {
	b := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func TestNewTestClient(t *testing.T) {
	newTestClient(t)
}

func TestListNodes(t *testing.T) {
	client := newTestClient(t)
	if _, err := client.CoreV1().ListNodes(context.Background()); err != nil {
		t.Fatal("failed to list nodes: %v", err)
	}
}

func TestConfigMaps(t *testing.T) {
	client := newTestClient(t).CoreV1()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	name := newName()
	labelVal := newName()

	cm := &v1.ConfigMap{
		Metadata: &v1.ObjectMeta{
			Name:      String(name),
			Namespace: String("default"),
			Labels: map[string]string{
				"testLabel": labelVal,
			},
		},
		Data: map[string]string{
			"foo": "bar",
		},
	}
	got, err := client.CreateConfigMap(ctx, cm)
	if err != nil {
		t.Fatalf("create config map: %v", err)
	}
	got.Data["zam"] = "spam"
	_, err = client.UpdateConfigMap(ctx, got)
	if err != nil {
		t.Fatalf("update config map: %v", err)
	}

	tests := []struct {
		labelVal string
		expNum   int
	}{
		{labelVal, 1},
		{newName(), 0},
	}
	for _, test := range tests {
		l := new(LabelSelector)
		l.Eq("testLabel", test.labelVal)

		configMaps, err := client.ListConfigMaps(ctx, "default", l.Selector())
		if err != nil {
			t.Errorf("failed to list configmaps: %v", err)
			continue
		}
		got := len(configMaps.Items)
		if got != test.expNum {
			t.Errorf("expected selector to return %d items got %d", test.expNum, got)
		}
	}

	if err := client.DeleteConfigMap(ctx, *cm.Metadata.Name, *cm.Metadata.Namespace); err != nil {
		t.Fatalf("delete config map: %v", err)
	}

}

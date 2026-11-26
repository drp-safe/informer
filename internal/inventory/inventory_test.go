package inventory

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func pod(name string, init, main, ephemeral []string) *corev1.Pod {
	p := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "pra"}}
	for _, img := range init {
		p.Spec.InitContainers = append(p.Spec.InitContainers, corev1.Container{Name: "i", Image: img})
	}
	for _, img := range main {
		p.Spec.Containers = append(p.Spec.Containers, corev1.Container{Name: "c", Image: img})
	}
	for _, img := range ephemeral {
		p.Spec.EphemeralContainers = append(p.Spec.EphemeralContainers,
			corev1.EphemeralContainer{EphemeralContainerCommon: corev1.EphemeralContainerCommon{Name: "e", Image: img}})
	}
	return p
}

func TestFromPodsDeduplicatesAndKeepsDistinctTags(t *testing.T) {
	// Passe par le client factice : la même liste que celle lue en cluster.
	client := fake.NewClientset(
		pod("backend-1", []string{"busybox:1.36"}, []string{"ghcr.io/obeyler/pra-backend:2.41.0"}, nil),
		pod("backend-2", []string{"busybox:1.36"}, []string{"ghcr.io/obeyler/pra-backend:2.41.0"}, nil),
		// Déploiement en cours : deux tags d'une même image coexistent.
		pod("backend-3", nil, []string{"ghcr.io/obeyler/pra-backend:2.42.0"}, nil),
		pod("debug", nil, []string{"nginx"}, []string{"nicolaka/netshoot:v0.13"}),
	)
	list, err := client.CoreV1().Pods("pra").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}

	want := []Image{
		{Image: "busybox", Tag: "1.36"},
		{Image: "ghcr.io/obeyler/pra-backend", Tag: "2.41.0"},
		{Image: "ghcr.io/obeyler/pra-backend", Tag: "2.42.0"},
		{Image: "nginx", Tag: "latest"},
		{Image: "nicolaka/netshoot", Tag: "v0.13"},
	}
	if got := FromPods(list.Items); !reflect.DeepEqual(got, want) {
		t.Errorf("FromPods =\n%+v\nattendu\n%+v", got, want)
	}
}

func TestFromPodsEmpty(t *testing.T) {
	if got := FromPods(nil); got == nil || len(got) != 0 {
		t.Errorf("namespace vide : liste vide non nulle attendue (JSON []), obtenu %#v", got)
	}
}

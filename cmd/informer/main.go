// Commande informer : liste les images des pods d'un namespace et envoie le
// résultat à une API. Prévue pour un CronJob : un code de sortie non nul fait
// passer le Job en échec.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/drp-safe/informer/internal/config"
	"github.com/drp-safe/informer/internal/inventory"
	"github.com/drp-safe/informer/internal/report"
)

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Printf("informer : %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	client, err := newClient()
	if err != nil {
		return fmt.Errorf("client Kubernetes : %w", err)
	}
	pods, err := listPods(ctx, client, cfg.Namespace)
	if err != nil {
		return fmt.Errorf("liste des pods de %s : %w", cfg.Namespace, err)
	}

	payload := report.Payload{
		Namespace:   cfg.Namespace,
		CollectedAt: time.Now().UTC().Truncate(time.Second),
		Images:      inventory.FromPods(pods),
	}
	log.Printf("namespace %s : %d pods, %d images distinctes", cfg.Namespace, len(pods), len(payload.Images))

	if cfg.DryRun {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	status, err := report.NewSender(cfg.ReportURL, cfg.ReportToken, cfg.HTTPTimeout).Send(ctx, payload)
	if err != nil {
		return fmt.Errorf("envoi du listing : %w", err)
	}
	log.Printf("listing envoyé (HTTP %d)", status)
	return nil
}

// newClient utilise le compte de service du pod, ou à défaut le kubeconfig
// local pour une exécution depuis un poste.
func newClient() (kubernetes.Interface, error) {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		restCfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return nil, err
		}
	}
	return kubernetes.NewForConfig(restCfg)
}

// listPods lit les pods par pages de 200 pour borner la mémoire d'une réponse.
func listPods(ctx context.Context, client kubernetes.Interface, namespace string) ([]corev1.Pod, error) {
	var pods []corev1.Pod
	opts := metav1.ListOptions{Limit: 200}
	for {
		list, err := client.CoreV1().Pods(namespace).List(ctx, opts)
		if err != nil {
			return nil, err
		}
		pods = append(pods, list.Items...)
		if list.Continue == "" {
			return pods, nil
		}
		opts.Continue = list.Continue
	}
}

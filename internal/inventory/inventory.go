// Package inventory dresse la liste dédoublonnée des images utilisées par des pods.
package inventory

import (
	"sort"

	corev1 "k8s.io/api/core/v1"
)

// Image est une entrée du listing : une image n'apparaît qu'une fois par tag.
type Image struct {
	Image  string `json:"image"`
	Tag    string `json:"tag"`
	Digest string `json:"digest,omitempty"`
}

// FromPods renvoie les images de tous les conteneurs des pods — principaux,
// d'initialisation et éphémères —, dédoublonnées sur (image, tag) et triées.
//
// On lit l'image demandée dans la spec, pas status.imageID : c'est elle que
// l'on maintient dans les manifestes, et le statut n'existe pas encore pour un
// pod en attente.
func FromPods(pods []corev1.Pod) []Image {
	type key struct{ name, tag string }
	seen := make(map[key]Image)

	add := func(ref string) {
		if ref == "" {
			return
		}
		r := ParseReference(ref)
		k := key{r.Name, r.Tag}
		if prev, ok := seen[k]; ok {
			// Même image et même tag avec plusieurs digests : on garde le
			// premier rencontré, sans multiplier les entrées.
			if prev.Digest == "" && r.Digest != "" {
				prev.Digest = r.Digest
				seen[k] = prev
			}
			return
		}
		seen[k] = Image{Image: r.Name, Tag: r.Tag, Digest: r.Digest}
	}

	for i := range pods {
		spec := &pods[i].Spec
		for _, c := range spec.InitContainers {
			add(c.Image)
		}
		for _, c := range spec.Containers {
			add(c.Image)
		}
		for _, c := range spec.EphemeralContainers {
			add(c.Image)
		}
	}

	images := make([]Image, 0, len(seen))
	for _, img := range seen {
		images = append(images, img)
	}
	sort.Slice(images, func(i, j int) bool {
		if images[i].Image != images[j].Image {
			return images[i].Image < images[j].Image
		}
		return images[i].Tag < images[j].Tag
	})
	return images
}

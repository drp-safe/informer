package inventory

import "testing"

func TestParseReference(t *testing.T) {
	cases := []struct {
		in   string
		want Reference
	}{
		{"nginx", Reference{Name: "nginx", Tag: "latest"}},
		{"nginx:1.27", Reference{Name: "nginx", Tag: "1.27"}},
		{"ghcr.io/obeyler/pra-backend:2.41.0", Reference{Name: "ghcr.io/obeyler/pra-backend", Tag: "2.41.0"}},
		// Le « : » du port de registre n'est pas un tag.
		{"registry:5000/app", Reference{Name: "registry:5000/app", Tag: "latest"}},
		{"registry:5000/app:1.2", Reference{Name: "registry:5000/app", Tag: "1.2"}},
		// Digest seul : image épinglée, sans tag.
		{"postgres@sha256:abc", Reference{Name: "postgres", Digest: "sha256:abc"}},
		{"registry:5000/app:1.2@sha256:abc", Reference{Name: "registry:5000/app", Tag: "1.2", Digest: "sha256:abc"}},
		{" alpine:3.20 ", Reference{Name: "alpine", Tag: "3.20"}},
	}
	for _, c := range cases {
		if got := ParseReference(c.in); got != c.want {
			t.Errorf("ParseReference(%q) = %+v, attendu %+v", c.in, got, c.want)
		}
	}
}

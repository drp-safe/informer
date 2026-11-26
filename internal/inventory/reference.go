package inventory

import "strings"

// Reference est une référence d'image découpée en nom, tag et digest.
type Reference struct {
	Name   string
	Tag    string
	Digest string
}

// ParseReference découpe une référence telle qu'écrite dans un pod
// (`registry:5000/org/app:1.2@sha256:…`).
//
// Le tag est ce qui suit le dernier « : » situé après le dernier « / » :
// le « : » d'un port de registre se trouve toujours avant un « / », il n'est
// donc jamais pris pour un tag. Sans tag ni digest, Kubernetes tire `latest` ;
// avec un digest seul, l'image est épinglée et n'a pas de tag.
func ParseReference(ref string) Reference {
	ref = strings.TrimSpace(ref)
	var r Reference

	if i := strings.Index(ref, "@"); i >= 0 {
		r.Digest = ref[i+1:]
		ref = ref[:i]
	}

	lastSlash := strings.LastIndex(ref, "/")
	if i := strings.LastIndex(ref, ":"); i > lastSlash {
		r.Tag = ref[i+1:]
		ref = ref[:i]
	}
	r.Name = ref

	if r.Tag == "" && r.Digest == "" {
		r.Tag = "latest"
	}
	return r
}

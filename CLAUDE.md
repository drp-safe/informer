Les règles communes (Git, commits, PR, SOPS, sécurité, langues) sont définies dans ../CLAUDE.md.

- Commits conventionnels obligatoires (`feat:`, `fix:`, `ci:`, `docs:`…) : semantic-release en déduit la version.
- Toute modification de `deploy/kustomize/**/networkpolicy*.yaml` passe par l'agent `k8s-netpol-reviewer` avant la PR.
- Le conteneur ne tourne jamais en root (`USER appuser` dans le Dockerfile).

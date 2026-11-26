# informer

Liste les images Docker utilisées par les pods d'un namespace Kubernetes et
envoie ce listing à une API, par un `POST` JSON authentifié. Le programme est
prévu pour tourner dans un **CronJob**.

## Ce qui est listé

- Toutes les images des conteneurs de chaque pod : principaux, d'initialisation
  (`initContainers`) et éphémères (`ephemeralContainers`, conteneurs de debug).
- L'image **demandée dans la spec** du pod, pas celle résolue dans son statut.
- Chaque couple (image, tag) n'apparaît **qu'une fois**. Une même image apparaît
  plusieurs fois seulement si plusieurs tags tournent en même temps (par exemple
  pendant un déploiement progressif).
- Une image sans tag est notée `latest`. Une image épinglée par digest seul a un
  tag vide et un champ `digest`.

## Charge utile envoyée

```http
POST <REPORT_URL>
Content-Type: application/json
Authorization: Bearer <REPORT_TOKEN>
```

```json
{
  "namespace": "pra",
  "collected_at": "2026-11-26T14:42:36Z",
  "images": [
    { "image": "alpine", "tag": "3" },
    { "image": "alpine", "tag": "3.20" },
    { "image": "ghcr.io/obeyler/pra2-backend", "tag": "2.40.0" },
    { "image": "postgres", "tag": "", "digest": "sha256:…" }
  ]
}
```

Une réponse 2xx vaut succès. Une erreur réseau ou une réponse 5xx déclenche une
nouvelle tentative (3 au total, attente de 2 s puis 4 s). Une réponse 4xx (jeton
refusé, corps rejeté) échoue aussitôt. En cas d'échec, le programme sort avec un
code non nul : le Job passe en *Failed* et l'échec reste visible.

## Configuration

| Variable | Obligatoire | Défaut | Rôle |
|---|---|---|---|
| `REPORT_URL` | oui (sauf `DRY_RUN`) | — | URL du `POST`, en `https://` |
| `REPORT_TOKEN` | oui (sauf `DRY_RUN`) | — | Jeton Bearer, à fournir par un Secret ; il n'est jamais journalisé |
| `TARGET_NAMESPACE` | non | namespace du pod | Namespace à inventorier |
| `HTTP_TIMEOUT` | non | `30s` | Délai maximal d'un appel |
| `DRY_RUN` | non | `false` | Affiche le JSON sur la sortie standard sans l'envoyer |
| `REPORT_ALLOW_INSECURE` | non | `false` | Autorise une URL en `http://` (tests locaux uniquement) |

## Essai depuis un poste

Hors du cluster, le programme utilise le kubeconfig courant :

```sh
DRY_RUN=true TARGET_NAMESPACE=pra go run ./cmd/informer
```

## Déploiement (kustomize)

- `deploy/kustomize/base/` contient :
  - la ServiceAccount ;
  - un Role limité à `list pods` et son RoleBinding ;
  - le CronJob, non-root, en système de fichiers en lecture seule, sans capacités ;
  - la NetworkPolicy.
- `deploy/kustomize/example/` est un overlay d'exemple : namespace `pra`,
  adresses réelles, Secret.

```sh
cp deploy/kustomize/example/informer.secrets.env.example deploy/kustomize/example/informer.secrets.env
# renseigner REPORT_TOKEN, adapter REPORT_URL et les adresses du patch NetworkPolicy
kubectl apply -k deploy/kustomize/example/
```

### NetworkPolicy

Aucune connexion entrante n'est permise. En sortie, seuls trois flux sont ouverts :

1. **DNS** : pods CoreDNS de `kube-system` (`k8s-app: kube-dns`), port 53 en UDP et TCP.
2. **API server** : IP des nœuds du plan de contrôle, port **6443**. La règle est
   évaluée après la traduction d'adresse du service `kubernetes`. Viser sa
   ClusterIP ou le port 443 ne laisse rien passer. Les IP se lisent avec :
   `kubectl get endpointslices -n default -l kubernetes.io/service-name=kubernetes`.
3. **API destinataire** : son IP en `/32`, port 443. Deux cas particuliers sont
   décrits en commentaire de `networkpolicy.yaml` : une cible interne au
   cluster, et une cible jointe par son nom public via Envoy, où le port réel est 10443.

Ne jamais utiliser `commonLabels` (ni `labels` avec `includeSelectors: true`) :
kustomize injecterait le label dans le sélecteur de la règle DNS, qui vise un
autre namespace, et la règle ne correspondrait plus à rien, sans aucune erreur.

### Inventorier un autre namespace que celui du CronJob

Renseigner `TARGET_NAMESPACE`. Le **Role** et le **RoleBinding** doivent alors
être créés dans le namespace cible. Le sujet du RoleBinding reste la
ServiceAccount `informer` de son propre namespace (`namespace:` explicite).

## Supervision

Si une règle réseau bloque un flux, l'échec est silencieux : l'appel expire, le
Job échoue au bout de 5 minutes, et rien d'autre ne le signale. Tout
déploiement réel doit donc s'accompagner d'une alerte, sur le modèle de celles
de `k8s/monitoring/base/grafana-alerting/rules.yaml`. Exemple pour un passage
quotidien (en heures, seuil 26) :

```promql
(time() - kube_cronjob_status_last_successful_time{cronjob="informer"}) / 3600 > 26
```

## Publication

Chaque merge sur `main` passe par semantic-release : les commits conventionnels
déterminent la version. L'image est ensuite publiée sur
`ghcr.io/drp-safe/informer:<version>`.

## Licence

Apache 2.0 : voir [LICENSE](LICENSE).

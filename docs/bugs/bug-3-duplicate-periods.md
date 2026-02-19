# Bug 3 — Périodes dupliquées si la même resource apparaît dans deux flow items

## Statut
- [ ] À corriger

## Sévérité
**Faible** — edge-case de configuration, pas de crash mais comportement incorrect

## Fichier concerné
`internal/controller/flow/service/resource_mapper.go` — méthode `findAssociatedPeriods`, lignes 143-170

## Description

`findAssociatedPeriods` itère sur tous les `flow.Spec.Flows` et appende le
`PeriodWithDelay` sans vérification de doublon :

```go
for _, flowItem := range flow.Spec.Flows {
    for _, resource := range flowItem.Resources {
        if resource.Name != resourceName {
            continue
        }
        // ...
        periodsWithDelay = append(periodsWithDelay, periodWithDelay)  // toujours
    }
}
```

Si la même resource est listée deux fois dans deux `flowItem` référençant le même
`periodName`, la période est ajoutée deux fois dans `periodsWithDelay`.
`buildPeriods` (resource_creator.go) ne déduplique pas non plus.

## Conséquence

Le child K8s CR reçoit la même période en double dans son `spec.periods`.
Le K8s controller la traitera deux fois lors de chaque réconciliation :
- Le scale-down est appliqué deux fois (idempotent, pas de crash)
- Le status contiendra des entrées dupliquées dans `success` / `failed`
- Les logs affichent le même scaling deux fois

## Configuration déclenchante

```yaml
flows:
  - periodName: scale-down-night
    resources:
      - name: api-backend-group   # ← même resource
        startTimeDelay: "5m"
  - periodName: scale-down-night
    resources:
      - name: api-backend-group   # ← listée une deuxième fois
        startTimeDelay: "5m"
```

## Ce qui n'est pas validé

Le `flow_validator.go` ne détecte pas ce cas : `ExtractFlowData` utilise une map
(`resourceNames[resource.Name] = true`) qui déduplique les noms, mais la validation
ne vérifie pas qu'une resource n'apparaît qu'une seule fois par `periodName`.

## Fix attendu

Dans `findAssociatedPeriods`, utiliser un set pour détecter les doublons et retourner
une erreur, ou dédupliquer silencieusement :

```go
// Option A — erreur explicite (recommandé)
seen := make(map[string]bool)  // clé : "periodName/resourceName"
for _, flowItem := range flow.Spec.Flows {
    for _, resource := range flowItem.Resources {
        if resource.Name != resourceName {
            continue
        }
        key := flowItem.PeriodName + "/" + resource.Name
        if seen[key] {
            return nil, fmt.Errorf(
                "resource %s appears more than once for period %s in flows",
                resource.Name, flowItem.PeriodName,
            )
        }
        seen[key] = true
        // ... append ...
    }
}
```

Alternativement, ajouter la validation dans `flow_validator.go` / `ExtractFlowData`
en vérifiant l'unicité des paires `(periodName, resourceName)` dans `flow.Spec.Flows`.

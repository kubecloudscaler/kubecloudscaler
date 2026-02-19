# Bug 2 — `calculateTotalDelay` : validation incorrecte des délais dans le Flow

## Statut
- [ ] À corriger

## Sévérité
**Haute** — peut rejeter des configurations valides et bloquer le déploiement du Flow

## Fichier concerné
`internal/controller/flow/service/flow_validator.go` — méthode `calculateTotalDelay`, lignes 98-126

## Description

Le validateur additionne **tous** les `startTimeDelay` et `endTimeDelay` de toutes les
resources pour une période donnée, puis compare cette somme à la durée de la période :

```go
// Implémentation actuelle — INCORRECTE
for _, resource := range flowItem.Resources {
    totalDelay += startDelay   // resource A : 30m
    totalDelay += endDelay     // resource A : 0m
    totalDelay += startDelay   // resource B : 0m
    totalDelay += endDelay     // resource B : 30m
    // totalDelay = 60m → rejeté si période = 60m
    // alors que les deux resources ont individuellement des fenêtres valides
}

if totalDelay > periodDuration {
    return fmt.Errorf("total delay %v exceeds period duration %v", ...)
}
```

La seule contrainte correcte est **par resource** :

```
pour chaque resource : startTimeDelay + endTimeDelay < periodDuration
```

## Exemple concret de rejet incorrect

Période `22:00`–`23:00` (60 min), 4 resources avec chacune `startTimeDelay: 10m`
et `endTimeDelay: 10m` :

| Resource            | startDelay | endDelay | Fenêtre ajustée      | Valide ? |
|---------------------|-----------|---------|----------------------|----------|
| statefulsets-group  | 10m        | 10m      | 22:10 → 22:50 (40m)  | ✓        |
| api-backend-group   | 10m        | 10m      | 22:10 → 22:50 (40m)  | ✓        |
| frontend-group      | 10m        | 10m      | 22:10 → 22:50 (40m)  | ✓        |
| worker-group        | 10m        | 10m      | 22:10 → 22:50 (40m)  | ✓        |

Total des délais : 4 × (10+10) = 80m > 60m → **rejeté par le validateur**
Pourtant chaque resource est individuellement valide.

## Comportement sur les faux négatifs

La validation actuelle ne laisse pas passer de configurations réellement invalides :
si une resource a `startDelay + endDelay > periodDuration`, sa contribution au total
est supérieure à `periodDuration`, ce qui déclenche le rejet. Il n'y a donc pas de
faux négatifs, seulement des faux positifs.

## Fix attendu

Remplacer la logique de somme globale par une vérification individuelle par resource :

```go
// flow_validator.go — logique corrigée
func (v *FlowValidatorService) validateResourceDelays(
    flow *kubecloudscalerv1alpha3.Flow,
    periodName string,
    periodDuration time.Duration,
) error {
    for _, flowItem := range flow.Spec.Flows {
        if flowItem.PeriodName != periodName {
            continue
        }
        for _, resource := range flowItem.Resources {
            startDelay, _ := time.ParseDuration(resource.StartTimeDelay)
            endDelay, _   := time.ParseDuration(resource.EndTimeDelay)
            if startDelay + endDelay >= periodDuration {
                return fmt.Errorf(
                    "resource %s: startTimeDelay (%v) + endTimeDelay (%v) "+
                    "must be less than period duration (%v)",
                    resource.Name, startDelay, endDelay, periodDuration,
                )
            }
        }
    }
    return nil
}
```

Supprimer `calculateTotalDelay` et appeler `validateResourceDelays` depuis
`ValidatePeriodTimings` à la place de l'appel actuel à `calculateTotalDelay`.

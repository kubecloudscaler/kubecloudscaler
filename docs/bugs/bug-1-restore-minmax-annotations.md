# Bug 1 — RestoreMinMaxAnnotations retourne `isRestored=true` sur annotation corrompue

## Statut
- [ ] À corriger

## Sévérité
**Haute** — restauration silencieusement ignorée sur les ressources HPA et ARS

## Fichier concerné
`pkg/k8s/utils/annotation_manager.go` — lignes 113 et 123

## Description

Lorsqu'une annotation de valeur originale existe mais est corrompue (impossible à parser),
`RestoreMinMaxAnnotations` retourne `isRestored=true` sur les deux chemins d'erreur :

```go
// ligne 113 — annotation min-original-value corrompue
return true, nil, 0, annot, fmt.Errorf("error parsing min value: %w", err)
//     ^^^^  FAUX : la restauration n'a pas eu lieu

// ligne 123 — annotation max-original-value corrompue
return true, nil, 0, annot, fmt.Errorf("error parsing max value: %w", err)
//     ^^^^  FAUX : même problème
```

L'appelant dans `strategies.go` vérifie l'erreur en premier, donc il n'y a pas de crash.
Mais la valeur de retour est sémantiquement incorrecte et peut induire en erreur tout futur
appelant qui ne vérifierait pas l'erreur avant `isRestored`.

## Comparaison avec la correction déjà appliquée

`RestoreIntAnnotations` (même fichier, ligne 199) avait le même bug et a été corrigé :

```go
// APRÈS correction dans RestoreIntAnnotations :
return false, nil, annot, fmt.Errorf("error parsing int value: %w", err)
//     ^^^^^  correct
```

`RestoreMinMaxAnnotations` n'a pas encore reçu la même correction.

## Ressources affectées

Les ressources utilisant `MinMaxReplicasStrategy` :
- HPA (HorizontalPodAutoscaler)
- ARS (AutoscalingRunnerSet — GitHub Actions)

## Fix attendu

Remplacer `return true` par `return false` aux deux chemins d'erreur (lignes 113 et 123),
et mettre à jour le test correspondant dans `annotation_manager_test.go` (même correction
que celle appliquée pour `RestoreIntAnnotations`).

```go
// ligne 113 — AVANT
return true, nil, 0, annot, fmt.Errorf("error parsing min value: %w", err)
// ligne 113 — APRÈS
return false, nil, 0, annot, fmt.Errorf("error parsing min value: %w", err)

// ligne 123 — AVANT
return true, nil, 0, annot, fmt.Errorf("error parsing max value: %w", err)
// ligne 123 — APRÈS
return false, nil, 0, annot, fmt.Errorf("error parsing max value: %w", err)
```

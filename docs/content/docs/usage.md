---
title: Usage
weight: 2
---

## Periods

Each *scaler definition* may include one or more *period definitions*. These *periods* are evaluated **sequentially**, with the **first matching period** taking precedence. A *period* can also be set to "*reversed*" with the `reverse` field, meaning it is considered **inactive** between its **startTime** and **endTime**, but **active** outside of that range (*before* the *startTime* and *after* the *endTime*).

{{< callout type="info" >}}
  *endTime* is inclusive, meaning that the period will be active until the last second before the specified endTime. This means that if you set an *endTime* of `00:00`, the period will be active until `23:59:59`.
{{< /callout >}}

Any Period with the `once` field set to **true** will be applied only when entering or living the given time range. If the resources are scaled manually, cloudscaler will not change their state.

### Recurring periods

The `startTime` and `endTime` must be formated as `12:34`.

### Fixed periods

The `startTime` and `endTime` must be formated as `2006-01-02 15:04:05`.

### Exemples

{{< tabs items="Simple, Multiple, Fixed" >}}

  {{< tab >}}
```yaml
periods:
  - time:
      recurring:
        days:
          - all
        startTime: "01:00"
        endTime: "22:50"
        timezone: "Europe/Paris"
    minReplicas: 0
    maxReplicas: 10
    type: "down"
```
{{< callout type="info" >}}
  The resources will be scaled down to 0 everyday form 1h to 22h50, Paris time.
{{< /callout >}}
  {{< /tab >}}

  {{< tab >}}
```yaml
periods:
  - time:
      recurring:
        days:
          - all
        startTime: "01:00"
        endTime: "07:00"
        timezone: "Europe/Paris"
    minReplicas: 0
    maxReplicas: 10
    type: "down"
  - time:
      recurring:
        days:
          - all
        startTime: "12:00"
        endTime: "20:00"
        timezone: "Europe/Paris"
    minReplicas: 0
    maxReplicas: 10
    type: "up"
```
{{< callout type="info" >}}
  The resources will be scaled down to 0 repliacs everyday form 1h to 7h and scaled up to 10 replicas from 12h to 20h, Paris time.
{{< /callout >}}
  {{< /tab >}}

  {{< tab >}}
```yaml
periods:
  - time:
      fixed:
        startTime: "2024-11-15 20:00:00"
        endTime: "2024-11-17 08:00:00"
        timezone: "Europe/Paris"
    minReplicas: 0
    maxReplicas: 10
    type: "down"
```
{{< callout type="info" >}}
  The resources will be scaled down to 0 repliacs between the November, 15th 20h00 to November, 17th 08h00, Paris time.
{{< /callout >}}
  {{< /tab >}}
{{< /tabs >}}

## Resources

### Kubernetes

The Kubernetes resource can be selected with various parameters, by default, all namespaces are selected excluding the `kube-system` namespace. Using an `excludeNamespaces` list will only exclude the specified namespaces. You can use the `forceExcludeSystemNamespaces` to enforce excluding `kube-system` namespace.

If no resources are specified, all **deployments** will be selected. Allowed resources are:
- deployments
- statefulsets
- conjobs
- horizontalPodAutoscalers

Any resource type can be filtered out by using a [labelSelector](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/label-selector/#LabelSelector) parameter.

{{< callout type="warning" >}}
  Deployments and HorizontalPodAutoscalers are mutually exclusive
{{< /callout >}}

#### SelfHealing

If you use [Argo-CD](https://argo-cd.readthedocs.io/en/stable/user-guide/diffing/) to maintain the state of your cluster, you may want to ignore the differences in the `managedFields` of the resources to avoid out-of-sync issues. This can be done by adding the following to your ArgoCD configuration:

```yaml
resource.customizations.ignoreDifferences.all: |
  managedFieldsManagers:
    - kubecloudscaler
```

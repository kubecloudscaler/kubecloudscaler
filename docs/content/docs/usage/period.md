---
title: Period
weight: 2
---

## Understanding Periods

Periods define when and how your resources should be scaled. Each scaler definition can contain multiple period definitions that control scaling behavior based on time patterns.

### Key Concepts

- **Sequential Evaluation**: Periods are evaluated in order, with the first matching period taking precedence
- **Reverse Mode**: Use the `reverse` field to invert period logic - making it inactive during the specified time range and active outside of it
- **One-time Scaling**: Set `once: true` to apply scaling only when entering or leaving a time range, preventing interference with manual scaling
- **Inclusive End Time**: The `endTime` is inclusive, meaning a period remains active until the last second before the specified end time (e.g., `endTime: "00:00"` stays active until `23:59:59`)

> [!NOTE]
> When `once` is enabled, KubeCloudScaler will only scale resources when transitioning into or out of the specified time range. Manual scaling operations will not be overridden.

## Period Types

### Recurring Periods

Recurring periods repeat on a daily basis according to specified days and times.

**Time Format**: `HH:MM` (24-hour format)

**Example**: `"08:30"` represents 8:30 AM

### Fixed Periods

Fixed periods occur at specific dates and times, useful for one-time events or maintenance windows.

**Time Format**: `YYYY-MM-DD HH:MM:SS`

**Example**: `"2024-12-25 09:00:00"` represents December 25th, 2024 at 9:00 AM

## Configuration Examples

{{< tabs items="Basic Scaling,Multiple Periods,Scheduled Maintenance" >}}

  {{< tab >}}
**Scenario**: Scale down resources during off-hours

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
> [!NOTE]
> Resources are scaled down to 0 replicas daily from 1:00 AM to 10:50 PM (Paris time).

  {{< /tab >}}

  {{< tab >}}
**Scenario**: Different scaling rules for different times of day

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
> [!NOTE]
> Resources are scaled down to 0 replicas from 1:00-7:00 AM and scaled up to 10 replicas from 12:00-8:00 PM (Paris time).

  {{< /tab >}}

  {{< tab >}}
**Scenario**: Planned maintenance window

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
> [!NOTE]
> Resources are scaled down to 0 replicas during a specific maintenance window from November 15th 8:00 PM to November 17th 8:00 AM (Paris time).

  {{< /tab >}}
{{< /tabs >}}

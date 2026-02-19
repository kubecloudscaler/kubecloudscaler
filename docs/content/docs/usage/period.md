---
title: Period
weight: 2
---

## Understanding Periods

Periods define when and how your resources should be scaled. Each scaler (K8s, Gcp, or Flow) can contain multiple period definitions that control scaling behavior based on time patterns.

### Key Concepts

- **Sequential Evaluation**: Periods are evaluated in order, with the first matching period taking precedence
- **Named Periods**: Use the optional `name` field to identify periods, especially when referencing them in Flow resources
- **Reverse Mode**: Use the `reverse` field to invert period logic -- making it inactive during the specified time range and active outside of it
- **One-time Scaling**: Set `once: true` to apply scaling only when entering or leaving a time range, preventing interference with manual scaling
- **Inclusive End Time**: The `endTime` is inclusive, meaning a period remains active until the last second before the specified end time (e.g., `endTime: "00:00"` stays active until `23:59:59`)

> [!NOTE]
> When `once` is enabled, KubeCloudScaler will only scale resources when transitioning into or out of the specified time range. Manual scaling operations will not be overridden.

## Period Structure

```yaml
periods:
  - type: "down"              # Required: "down" or "up"
    name: "my-period"         # Optional: period name (alphanumeric, hyphens, underscores)
    minReplicas: 0            # Optional: minimum replica count
    maxReplicas: 10           # Optional: maximum replica count
    time:
      recurring: { ... }      # Use recurring OR fixed, not both
      fixed: { ... }
```

## Period Types

### Recurring Periods

Recurring periods repeat on a daily basis according to specified days and times.

**Time Format**: `HH:MM` (24-hour format)

**Available days**: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`, `all`

**Fields**:
| Field | Required | Description |
|-------|----------|-------------|
| `days` | Yes | List of day names or `all` |
| `startTime` | Yes | Start time in `HH:MM` format |
| `endTime` | Yes | End time in `HH:MM` format |
| `timezone` | No | IANA timezone (e.g., `Europe/Paris`) |
| `once` | No | Only scale on period transition |
| `reverse` | No | Invert the period (active outside the time range) |
| `gracePeriod` | No | Duration before scaling (e.g., `60s`) |

### Fixed Periods

Fixed periods occur at specific dates and times, useful for one-time events or maintenance windows.

**Time Format**: `YYYY-MM-DD HH:MM:SS`

**Fields**:
| Field | Required | Description |
|-------|----------|-------------|
| `startTime` | Yes | Start date/time in `YYYY-MM-DD HH:MM:SS` format |
| `endTime` | Yes | End date/time in `YYYY-MM-DD HH:MM:SS` format |
| `timezone` | No | IANA timezone (e.g., `Europe/Paris`) |
| `once` | No | Only scale on period transition |
| `reverse` | No | Invert the period |
| `gracePeriod` | No | Duration before scaling (e.g., `120s`) |

## Configuration Examples

{{< tabs items="Basic Scaling,Multiple Periods,Scheduled Maintenance,Reverse Mode" >}}

  {{< tab >}}
**Scenario**: Scale down resources during off-hours

```yaml
periods:
  - type: "down"
    name: "off-hours"
    minReplicas: 0
    maxReplicas: 10
    time:
      recurring:
        days:
          - all
        startTime: "01:00"
        endTime: "22:50"
        timezone: "Europe/Paris"
```
> [!NOTE]
> Resources are scaled down to 0 replicas daily from 1:00 AM to 10:50 PM (Paris time).

  {{< /tab >}}

  {{< tab >}}
**Scenario**: Different scaling rules for different times of day

```yaml
periods:
  - type: "down"
    name: "night"
    minReplicas: 0
    maxReplicas: 10
    time:
      recurring:
        days:
          - all
        startTime: "01:00"
        endTime: "07:00"
        timezone: "Europe/Paris"
  - type: "up"
    name: "peak-hours"
    minReplicas: 0
    maxReplicas: 10
    time:
      recurring:
        days:
          - all
        startTime: "12:00"
        endTime: "20:00"
        timezone: "Europe/Paris"
```
> [!NOTE]
> Resources are scaled down from 1:00-7:00 AM and scaled up from 12:00-8:00 PM (Paris time). Between 7:00-12:00 and 20:00-1:00, no period matches and the default behavior applies.

  {{< /tab >}}

  {{< tab >}}
**Scenario**: Planned maintenance window

```yaml
periods:
  - type: "down"
    name: "maintenance"
    minReplicas: 0
    maxReplicas: 10
    time:
      fixed:
        startTime: "2026-03-15 20:00:00"
        endTime: "2026-03-17 08:00:00"
        timezone: "Europe/Paris"
        gracePeriod: "120s"
```
> [!NOTE]
> Resources are scaled down during a specific maintenance window from March 15th 8:00 PM to March 17th 8:00 AM (Paris time), with a 2-minute grace period.

  {{< /tab >}}

  {{< tab >}}
**Scenario**: Keep resources up during business hours only

```yaml
periods:
  - type: "down"
    name: "outside-business-hours"
    minReplicas: 0
    time:
      recurring:
        days:
          - monday
          - tuesday
          - wednesday
          - thursday
          - friday
        startTime: "08:00"
        endTime: "18:00"
        timezone: "Europe/Paris"
        reverse: true
        gracePeriod: "60s"
```
> [!NOTE]
> With `reverse: true`, the period is **active outside** the specified time range. Resources are scaled down outside 8:00 AM - 6:00 PM on weekdays, and all day on weekends (since weekends are not listed in `days`).

  {{< /tab >}}
{{< /tabs >}}

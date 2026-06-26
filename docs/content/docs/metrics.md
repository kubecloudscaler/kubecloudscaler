---
title: 'Metrics'
weight: 1
---

# Prometheus Metrics

The kubecloudscaler operator exposes custom Prometheus metrics on the same endpoint as controller-runtime metrics. All metrics use the `kubecloudscaler_` namespace prefix.

## Exposed Metrics

### `kubecloudscaler_reconcile_total`

Counter of reconciliation runs by controller and result.

| Label        | Possible values | Description                          |
|-------------|------------------|--------------------------------------|
| `controller` | `k8s_scaler`, `gcp_scaler`, `flow` | Controller that ran the reconciliation |
| `result`     | `success`, `critical_error`, `recoverable_error` | Reconciliation outcome |

**Example queries:**
- Error rate by controller:  
  `rate(kubecloudscaler_reconcile_total{result=~"critical_error|recoverable_error"}[5m]) / rate(kubecloudscaler_reconcile_total[5m])`
- Successful K8s reconciliations:  
  `rate(kubecloudscaler_reconcile_total{controller="k8s_scaler", result="success"}[5m])`

---

### `kubecloudscaler_reconcile_duration_seconds`

Histogram of reconciliation duration in seconds, by controller.

| Label        | Possible values | Description  |
|-------------|------------------|--------------|
| `controller` | `k8s_scaler`, `gcp_scaler`, `flow` | Controller |

Buckets: 0.001, 0.0025, 0.00625, … up to ~14.5s (exponential ×2.5).

**Example queries:**
- K8s reconciliation duration P95:  
  `histogram_quantile(0.95, rate(kubecloudscaler_reconcile_duration_seconds_bucket{controller="k8s_scaler"}[5m]))`
- Average duration by controller:  
  `rate(kubecloudscaler_reconcile_duration_seconds_sum[5m]) / rate(kubecloudscaler_reconcile_duration_seconds_count[5m])`

---

### `kubecloudscaler_scaling_operations_total`

Counter of scaling operations (success or failure) by controller, resource kind, and result.

| Label           | Possible values | Description                                      |
|-----------------|-----------------|--------------------------------------------------|
| `controller`    | `k8s_scaler`, `gcp_scaler` | Controller (Flow does not scale directly)       |
| `resource_kind` | e.g. `deployments`, `statefulsets`, `cronjobs`, `hpa`, `github-ars`, `vm-instances` | Target resource type |
| `result`        | `success`, `failed` | Scaling operation result                        |

**Example queries:**
- Scaling failure rate by kind:  
  `rate(kubecloudscaler_scaling_operations_total{result="failed"}[5m]) / (rate(kubecloudscaler_scaling_operations_total[5m]) or vector(0))`
- Successful scaling volume by kind:  
  `sum by (resource_kind) (rate(kubecloudscaler_scaling_operations_total{result="success"}[5m]))`

---

### `kubecloudscaler_period_activations_total`

Counter of times a period was considered active, by controller and period type.

| Label        | Possible values | Description                                  |
|-------------|-----------------|----------------------------------------------|
| `controller` | `k8s_scaler`, `gcp_scaler` | Controller (Flow does not evaluate periods)  |
| `period_type`| `up`, `down`, `noaction` | Active period type (scale up, scale down, or none) |

**Example queries:**
- Reconciliations with “up” period:  
  `rate(kubecloudscaler_period_activations_total{period_type="up"}[5m])`
- Breakdown by up/down/noaction:  
  `sum by (period_type) (rate(kubecloudscaler_period_activations_total[5m]))`

---

## Registration and Disabling

- Metrics are registered with the default Prometheus registry (`prometheus.DefaultRegisterer`) at startup via `metrics.Init()` in `cmd/main.go`. No extra configuration is required.
- In tests, `DefaultRecorder` remains a no-op until `Init()` is called; controllers use `metrics.GetRecorder()`, so real metrics are not recorded when tests do not call `Init()`.

## Disabling Authentication (dev)

For local access to the metrics endpoint without auth (e.g. `curl http://localhost:8080/metrics`), start the manager with:

```bash
go run ./cmd/main.go --webhook-cert-path=./tmp/k8s-webhook-server/serving-certs --metrics-disable-auth --metrics-bind-address=:8080 --metrics-secure=false
```

Or after `make build`: `./bin/kubecloudscaler --metrics-disable-auth --metrics-bind-address=:8080 --metrics-secure=false` (plus any other required flags).

**Do not use `--metrics-disable-auth` in production**: the endpoint would be accessible without authentication or authorization.

## Grafana Dashboard

A ready-to-use Grafana dashboard is provided in [`grafana/kubecloudscaler-dashboard.json`](https://github.com/kubecloudscaler/kubecloudscaler/blob/main/grafana/kubecloudscaler-dashboard.json). Import it into your Grafana instance to visualize reconciliations, scaling operations, and period activations.

## Best Practices

- **Cardinality**: Labels are limited to known values (controller, result, period_type, resource_kind). Do not add resource or CR names to avoid series explosion.
- **Alerting**: Use rates (`rate()`) and quantiles (`histogram_quantile()`) over 5–15 minute windows to reduce noise.

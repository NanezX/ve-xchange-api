# Monitoring (Prometheus + Grafana + Tailscale)

Este documento registra la configuracion operativa para correr monitoring sin chocar con la app principal y exponer Grafana por Tailscale.

## Objetivo

- Mantener Optikt en puerto 3000.
- Correr Grafana en puerto 3001 para evitar conflicto.
- Mantener Prometheus interno (sin publicarlo al host).
- Exponer Grafana en subpath por Tailscale para acceso HTTPS solo dentro de la tailnet.

## Mapeo de puertos

- Optikt: host 3000 -> contenedor 3000
- Grafana: host 127.0.0.1:3001 -> contenedor 3000
- Prometheus: sin puerto publicado al host
- API ve-xchange: 8080 (scrape target de Prometheus dentro de Docker)

## Levantar monitoring

Desde la raiz del repo:

```bash
docker compose -f docker-compose.monitoring.yml up -d
```

## Exponer Grafana por Tailscale (subpath)

Registrar subpath de Grafana:

```bash
tailscale serve --bg --set-path /grafana/ http://localhost:3001
```

Si quieres que quede en segundo plano (persistente en reinicios), puedes usar:

```bash
tailscale serve --bg https:443 /grafana/ http://localhost:3001
```

## URL de acceso

Usa el MagicDNS del nodo:

```text
https://<tu-hostname>.ts.net/grafana/
```

Ejemplo real del equipo usado durante esta implementacion:

```text
https://nanezx-elitebook.taild8f0b9.ts.net/grafana/
```

## Verificar rutas publicadas en Tailscale

```bash
tailscale serve status
```

Se espera algo como:

```text
https://<tu-hostname>.ts.net (tailnet only)
|-- /         http://localhost:3000    # optikt
|-- /grafana/ http://localhost:3001    # grafana
```

## Config de Grafana relevante

En docker-compose.monitoring.yml, Grafana usa:

- GF_SERVER_ROOT_URL=https://<tu-hostname>.ts.net/grafana/

Nota: si cambias hostname de Tailscale, actualiza GF_SERVER_ROOT_URL.

## Data source en Grafana

Dentro de Grafana, Prometheus debe configurarse con URL interna de Docker:

```text
http://prometheus:9090
```

## Que metricas exponemos hoy

La API publica estas metricas en el endpoint /metrics:

- provider_fetch_total{provider, status} (counter)
- provider_consecutive_failures{provider} (gauge)
- rate_value{currency} (gauge)
- http_request_duration_seconds{method, path, status} (histogram)

Valores de labels usados actualmente:

- provider: DolarAPI, USDT
- status (provider_fetch_total): success, failure
- currency: usd_bcv, eur_bcv, usdt_binance

## Queries utiles para empezar (copy/paste)

### 1) Tasa actual por moneda (Stat o Time series)

```promql
rate_value
```

Si quieres una moneda especifica:

```promql
rate_value{currency="usdt_binance"}
```

### 2) Exitos por proveedor en 1h (Stat/Bar gauge)

```promql
sum by (provider) (increase(provider_fetch_total{status="success"}[1h]))
```

### 3) Fallos por proveedor en 1h (Stat/Bar gauge)

```promql
sum by (provider) (increase(provider_fetch_total{status="failure"}[1h]))
```

### 4) Porcentaje de error por proveedor en 1h

```promql
100 * sum by (provider) (increase(provider_fetch_total{status="failure"}[1h]))
/ clamp_min(sum by (provider) (increase(provider_fetch_total[1h])), 1)
```

### 5) Racha actual de fallos consecutivos (alerta operativa)

```promql
provider_consecutive_failures
```

Ejemplo para alertar cuando sea mayor a 3:

```promql
provider_consecutive_failures > 3
```

### 6) Latencia p95 HTTP por ruta (5m)

```promql
histogram_quantile(0.95,
	sum by (le, method, path) (
		rate(http_request_duration_seconds_bucket[5m])
	)
)
```

### 7) RPS por ruta (5m)

```promql
sum by (method, path) (rate(http_request_duration_seconds_count[5m]))
```

### 8) Errores HTTP (4xx/5xx) por ruta en 15m

```promql
sum by (method, path, status) (
	increase(http_request_duration_seconds_count{status=~"4..|5.."}[15m])
)
```

## Dashboard minimo recomendado

Si estas empezando, crea 1 dashboard con 6 paneles:

- Rate actual (rate_value)
- Exitos por proveedor (1h)
- Fallos por proveedor (1h)
- Consecutive failures actual
- Latencia p95 por ruta
- RPS por ruta

Con eso ya puedes responder: "esta vivo", "esta lento", "esta fallando", y "que proveedor esta mal".

## Seguridad

- Prometheus no se expone al host por defecto.
- Grafana se publica via Tailscale, por lo que solo dispositivos de la tailnet pueden acceder.
- Si necesitas abrir Prometheus temporalmente, usa bind local (127.0.0.1) y no 0.0.0.0.

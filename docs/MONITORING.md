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
- GF_SERVER_SERVE_FROM_SUB_PATH=true

Nota: si cambias hostname de Tailscale, actualiza GF_SERVER_ROOT_URL.

## Data source en Grafana

Dentro de Grafana, Prometheus debe configurarse con URL interna de Docker:

```text
http://prometheus:9090
```

## Seguridad

- Prometheus no se expone al host por defecto.
- Grafana se publica via Tailscale, por lo que solo dispositivos de la tailnet pueden acceder.
- Si necesitas abrir Prometheus temporalmente, usa bind local (127.0.0.1) y no 0.0.0.0.

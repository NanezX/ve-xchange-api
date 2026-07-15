# Cambios: Pipeline BCV

## ¿Por qué?

Reemplazar `ve.dolarapi.com` como fuente de USD/EUR por scraping directo del BCV. DolarAPI era inestable (tasas desactualizadas los fines de semana) y no teníamos control sobre la calidad del dato.

## Arquitectura

- **Provider:** `internal/provider/bcvProvider.go` — scraper con goquery que extrae USD, EUR y Fecha Valor del HTML de bcv.org.ve. Normaliza números localizados (coma decimal, separador de miles).
- **Worker:** `internal/worker/worker.go` — `BusinessWindow` ejecuta el fetch programado de lun–vie 17:00–19:00 UTC-4 con reintentos cada 30 min si el BCV aún no publicó la tasa nueva.
- **Validación:** se rechazan respuestas cuya Fecha Valor no sea el próximo día hábil (ej: viernes se espera lunes).
- **Persistencia:** los handlers `/rates` y `/rates/{currency}` leen desde PostgreSQL (`GetLatestRates`), no desde la memoria del proceso. Si la DB no responde, devuelven 503.

## Cambios principales

- `rates.PriceResponse` pasó de `map[string]float64` a struct con `Values map[string]PriceResult` + `EffectiveDate` opcional.
- Se eliminó DolarAPI del registro de workers en `main.go`.
- Se agregó `BusinessWindow` (ventana horaria con reintentos) al worker.
- `validateBCVPublication` y `nextBusinessDate` se movieron de `main.go` a `internal/provider/bcv.go` para ser testeables.
- `NewBCVProvider` configura TLS `InsecureSkipVerify` porque el servidor del BCV envía una cadena de certificados incompleta. El cliente de Binance no se ve afectado.
- Se actualizó `api/openapi.yaml` con respuestas 503.
- Se eliminó `cmd/server/main_test.go` (tests duplicados).

## Tests agregados

- `nextBusinessDate` (5 casos: lunes→martes, viernes→lunes, sábado→lunes, domingo→lunes, martes→miércoles).
- `ValidateBCVPublication` (5 casos: sin fecha, fecha correcta, fecha vieja, viernes→lunes, fecha equivocada).
- `parseEffectiveDate` (18 casos: todos los meses en español, variantes setiembre/septiembre, fecha con prefijo, errores).
- Context cancellation durante `BusinessWindow`.
- Initial fetch inmediato con `BusinessWindow`.

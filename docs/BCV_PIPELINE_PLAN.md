# Plan de Revision: Pipeline de Tasas BCV

## Objetivo

Reemplazar la fuente actual de USD y EUR por una extraccion automatizada del
sitio oficial del Banco Central de Venezuela (BCV), conservar un historico en
PostgreSQL y servir las tasas actuales desde la base de datos, sin llamadas
externas durante las solicitudes HTTP de clientes.

Este documento registra el diseno revisado. Su implementacion esta completa:

- `internal/provider/bcvProvider.go` extrae y valida USD, EUR y `Fecha Valor`
  del sitio oficial del BCV.
- `internal/worker/worker.go` programa la ventana laboral y sus reintentos
  acotados.
- `cmd/server/main.go` valida el siguiente dia habil antes de persistir.
- `internal/handler/handlers.go` sirve las tasas actuales desde PostgreSQL.

## Decisiones Acordadas

- El proveedor consultara `https://bcv.org.ve` para obtener USD y EUR.
- El proceso se ejecutara fuera del servidor HTTP, en una goroutine de fondo.
- Habra una ejecucion inicial al arrancar, sin restricciones de horario ni de
  fin de semana.
- En ejecuciones programadas, se intentara a las 17:00 UTC-4 de lunes a
  viernes.
- Si la consulta programada falla o el BCV aun muestra una tasa anterior, se
  reintentara cada 30 minutos, solo antes de las 19:00 UTC-4.
- No se consultara el BCV en fines de semana ni despues de la hora limite como
  parte de la agenda programada.
- Una respuesta valida cuya `Fecha Valor` no corresponda al siguiente dia habil
  se considerara reintentable; no actualizara estado ni persistira filas.
- Las rutas actuales `/rates` y `/rates/{currency}` consultaran PostgreSQL para
  todas las monedas, incluido USDT.
- No se requiere una migracion nueva: `prices_history` ya soporta el historico
  de observaciones.

## Estado Actual Relevante

El proyecto ya tiene las piezas principales:

- `worker.StartPriceWorker` ejecuta proveedores en segundo plano y realiza una
  consulta inicial.
- `prices_history` almacena moneda, valor y momento de observacion.
- `db.Store.GetLatestRates` devuelve la ultima fila por moneda.
- Los proveedores ya manejan reintentos de red y el cliente HTTP tiene timeout.
- Las rutas actuales sirven valores desde `state.State`; se cambiaran para usar
  la base de datos como fuente de valores y timestamps.

## Diseno Propuesto

### 1. Resultado enriquecido del proveedor

Actualmente, `rates.PriceResponse` es un mapa de tasas. Se convertira en una
estructura que incluya:

- `Values`: las tasas por clave interna, por ejemplo `USD_BCV` y `EUR_BCV`.
- `EffectiveDate`: fecha de vigencia publicada por el BCV, opcional para
  proveedores que no la publiquen, como Binance.

Esto permite que la agenda BCV decida si la pagina ya contiene la nueva tasa sin
introducir reglas de calendario dentro del parser HTML.

### 2. Proveedor oficial BCV

Se agregara `internal/provider/bcvProvider.go` y se retirara el uso del
proveedor DolarAPI para USD/EUR.

Responsabilidades del proveedor:

1. Solicitar `https://bcv.org.ve` con el cliente HTTP actual.
2. Reutilizar la politica existente de tres intentos con backoff exponencial.
3. Analizar el DOM con `goquery`, buscando el bloque del tipo de cambio de
   referencia y las etiquetas USD, EUR y `Fecha Valor`.
4. Normalizar importes localizados: espacios, separadores de miles y coma
   decimal.
5. Convertir los importes a `float64` y validar que sean positivos y finitos.
6. Analizar la fecha localizada en espanol publicada por el BCV.
7. Rechazar respuestas incompletas, HTML inesperado, importes invalidos y
   errores de red o HTTP.

Las claves internas existentes `USD_BCV` y `EUR_BCV` se conservan para no
alterar el contrato de estado ni de persistencia.

### 3. Agenda laboral BCV

Se extendera `worker.ProviderJob` con una modalidad de ventana horaria,
mutuamente exclusiva con `Every` y `DailyAt`.

Configuracion del trabajo BCV:

| Propiedad | Valor |
| --- | --- |
| Zona horaria | UTC-4 |
| Dias programados | Lunes a viernes |
| Primer intento | 17:00 |
| Reintento | Cada 30 minutos, solo tras fallo |
| Limite | Ningun intento a las 19:00 ni despues |
| Ejecucion inicial | Siempre al iniciar la aplicacion |

Una ejecucion programada terminara al obtener una nueva tasa valida. Si no lo
logra antes de las 19:00, el siguiente intento programado sera el siguiente dia
habil a las 17:00.

La ejecucion inicial no abre una cadena de reintentos fuera de la ventana; es
solo una consulta inmediata para poblar el servicio tras un reinicio.

### 4. Regla de novedad

En la ventana programada, una respuesta BCV se aceptara unicamente si
`EffectiveDate` corresponde al siguiente dia habil respecto a la hora actual en
UTC-4. Ejemplos:

- Lunes: se espera la fecha del martes.
- Viernes: se espera la fecha del lunes.

Si el BCV aun devuelve la fecha vigente anterior, el resultado se tratara como
fallo reintentable. Esto evita registrar repetidamente la tasa previa como si
fuera una publicacion nueva.

### 5. Persistencia y estado

Al aceptar una respuesta BCV, la aplicacion mantendra el flujo actual:

1. Actualizar `state.State` para las senales de salud y metricas en proceso.
2. Actualizar las metricas de USD y EUR.
3. Insertar una observacion para cada moneda en `prices_history` con el momento
   de obtencion.

Se conservaran las banderas `ProviderFailing` de `state.State` para que
`/health` pueda informar degradacion en tiempo real. Binance y su consolidacion
nocturna no forman parte de este cambio funcional.

### 6. Lectura de tasas actuales desde PostgreSQL

`GET /rates` y `GET /rates/{currency}` dejaran de usar los valores almacenados
en memoria. En cada solicitud consultaran `Store.GetLatestRates` y construiran
la respuesta a partir de los valores y timestamps retornados por PostgreSQL.

Comportamiento esperado:

- No se realiza ninguna llamada al BCV, Binance u otro proveedor durante la
  solicitud HTTP.
- Los umbrales actuales de antiguedad se calculan con `recorded_at`.
- Las banderas de proveedor degradado de `state.State` continuan marcando una
  tasa como stale aunque exista una fila reciente en la base de datos.
- Si PostgreSQL no esta configurado o la consulta falla, las rutas devolveran
  `503` y el cuerpo `api.Error`.
- Si una moneda no tiene observaciones, se representara como tasa ausente y
  stale, manteniendo el formato actual de `RateEntry`.

## Archivos Afectados

| Archivo | Cambio planificado |
| --- | --- |
| `internal/rates/rates.go` | Estructurar la respuesta del proveedor con fecha efectiva opcional. |
| `internal/provider/bcvProvider.go` | Nuevo scraper y validacion de la pagina BCV. |
| `internal/provider/dolarApiProvider.go` | Retirar su uso como fuente BCV; evaluar su eliminacion posterior. |
| `internal/provider/binanceProvider.go` | Adaptar al nuevo resultado de proveedor sin fecha efectiva. |
| `internal/worker/worker.go` | Agregar agenda laboral con ventana y reintentos condicionados. |
| `cmd/server/main.go` | Registrar el proveedor BCV, su regla de novedad, persistencia y metricas. |
| `internal/handler/handlers.go` | Leer las tasas actuales desde `GetLatestRates`. |
| `internal/handler/handlers_test.go` | Verificar que las rutas usan PostgreSQL y no dependen de valores en memoria. |
| `api/openapi.yaml` | Documentar respuestas `503` de las rutas de tasas actuales. |
| `internal/api/server.gen.go` | Regenerar tras actualizar OpenAPI. |
| `README.md` y `docs/PLAN.md` | Actualizar fuente, agenda y arquitectura documentada. |

## Pruebas Requeridas

### Parser BCV

- USD y EUR con coma decimal.
- Importes con espacios y separadores de miles.
- Fecha de valor en espanol.
- Falta USD, EUR o fecha.
- HTML con estructura inesperada.
- Valores cero, negativos, `NaN` o infinitos.
- Errores de red y respuestas HTTP no exitosas.

### Agenda

- Ejecucion inicial inmediata, incluso en sabado o fuera de horario.
- Proximo intento laboral desde cada dia de la semana.
- Inicio lunes a viernes a las 17:00 UTC-4.
- Reintentos a las 17:30, 18:00 y 18:30 tras fallo.
- Ausencia de intento a las 19:00 o despues.
- No hay reintento cuando la primera respuesta programada es nueva y valida.
- Una `Fecha Valor` anterior se considera fallo reintentable.

### Handlers y persistencia

- `/rates` usa `GetLatestRates` y devuelve valores persistidos.
- `/rates/{currency}` usa la misma fuente.
- Fallo de store y store no configurado devuelven `503`.
- Una tasa ausente se expone como stale.
- Las banderas de proveedor degradado siguen forzando stale.
- Ninguna prueba de handler usa o necesita un cliente HTTP externo.

## Validacion de Entrega

1. Ejecutar pruebas focalizadas:

   ```bash
   go test ./internal/provider ./internal/worker ./internal/state ./internal/handler
   ```

2. Ejecutar la suite completa y detector de carreras:

   ```bash
   go test ./...
   go test -race ./...
   ```

3. Ejecutar el linter si esta instalado:

   ```bash
   golangci-lint run
   ```

4. Con PostgreSQL local, comprobar que la aplicacion inserta una observacion BCV
   aceptada y que `/rates` y `/rates/usd_bcv` entregan la ultima fila sin
   activar ningun proveedor HTTP.

## Riesgos y Mitigaciones

| Riesgo | Mitigacion |
| --- | --- |
| El HTML del BCV cambia | Parser basado en etiquetas y bloque semantico, validacion estricta y fixtures de estructura inesperada. |
| Latencia o caidas breves del BCV | Timeout HTTP existente, tres intentos con backoff y reintentos de agenda limitados a la ventana. |
| Publicacion tardia de la tasa | Validacion por `Fecha Valor` y reintentos de 30 minutos hasta antes de las 19:00. |
| Saturacion de consultas durante solicitudes API | Fuente de datos en PostgreSQL con una consulta de ultima observacion; sin llamadas externas por request. |
| Confusion entre cache y fuente de verdad | PostgreSQL sera la fuente de valores para rutas de tasas; `State` queda limitado a salud y degradacion. |

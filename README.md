# GoCuotas — cliente Go (`go/`)

Módulo Go **`github.com/humen-io/gocuotas/go`** alineado con el monorepo Java bajo `java/`: **API Client V1** (comercio, liquidaciones, CSV texto plano) y **API Redirect V1** (authenticate, checkout, órdenes, reembolso). Solo dependencias estándar (`net/http`, `encoding/json`, `context`).

## Requisitos

- Go **1.22** o superior
- Desde el directorio `go/`: `go test ./...`, `go build ./...`

## Paquetes

| Paquete | Descripción |
|---------|-------------|
| `github.com/humen-io/gocuotas/go/api` | `APIError` para respuestas HTTP no exitosas (equivalente a `GoCuotasApiException` en Java). |
| `github.com/humen-io/gocuotas/go/clientv1` | **Client V1**: `GET /api_client/v1/client`, `expense_settlements`, `expense_settlements_csvs` y `expense_settlements_csvs/{id}` con `Accept: text/plain` donde corresponde. |
| `github.com/humen-io/gocuotas/go/redirect` | **Redirect V1**: `authentication`, `checkouts`, `orders` (listado con `delivered_*`, detalle GET, reembolso DELETE). `OrderNotFoundError` en **404** sobre una orden por id (misma idea que `OrderNotFoundException` en Java). |

## Equivalencia Java → Go (nombres de API)

### Client V1 (`gocuotas-client` en Java)

| Java (`GoCuotasClientV1`) | Go (`clientv1.Client`) |
|---------------------------|-------------------------|
| `obtenerInformacionComercio(key)` | `GetCommerce(ctx, key)` |
| `obtenerInformacionComercio()` | `GetCommerceFromEnv(ctx)` |
| `listarLiquidaciones(key)` | `ListSettlements(ctx, key)` |
| `listarLiquidaciones()` | `ListSettlementsFromEnv(ctx)` |
| `obtenerInformacionLiquidacion(key, id)` | `GetSettlement(ctx, key, id)` |
| `obtenerInformacionLiquidacion()` | `GetSettlementFromEnvIDs(ctx)` (usa `GOCUOTAS_LIQUIDACION_ID`) |
| `obtenerInformacionLiquidacionesTextoPlano(key)` | `ListSettlementsPlainText(ctx, key)` |
| `obtenerInformacionLiquidacionesTextoPlano()` | `ListSettlementsPlainTextFromEnv(ctx)` |
| `obtenerLiquidacionTextoPlano(key, id)` | `GetSettlementPlainText(ctx, key, id)` |
| `obtenerLiquidacionTextoPlano()` | `GetSettlementPlainTextFromEnvIDs(ctx)` |

Modelos JSON: `CommerceResponse`, `Liquidacion`, `SettlementInfo` (detalle), etc., con etiquetas `json` iguales a los nombres de campo de la API.

### Redirect V1 (`gocuotas-redirect` en Java)

| Java (`GoCuotasRedirectClient`) | Go (`redirect.Client`) |
|--------------------------------|-------------------------|
| `authenticate(email, password)` | `Authenticate(ctx, email, password)` |
| `createCheckout(bearer, req)` | `CreateCheckout(ctx, bearer, req)` |
| `createCheckout(req)` | `CreateCheckoutFromEnv(ctx, req)` (Bearer = `GOCUOTAS_API_KEY`) |
| `listOrders(bearer, start, end)` | `ListOrders(ctx, bearer, start, end)` |
| `listOrders(bearer)` con env de fechas | `ListOrdersFromEnvDelivered(ctx, bearer)` |
| `listOrders()` | `ListOrdersAuto(ctx)` |
| `listarOrdenes(...)` (JsonNode) | `ListOrdersJSON(...)` → `json.RawMessage` |
| `getOrder(...)` | `GetOrder(...)` (string) / `GetOrderJSON(...)` |
| `buscarOrden(id)` | `GetOrderJSONAuto(ctx, id)` o `GetOrderAuto` + parse |
| `refundOrder(...)` | `RefundOrder(...)` / `RefundOrderJSON(...)` |
| `reembolsarOrden(...)` | `RefundOrderJSON(...)` |

La respuesta de authenticate acepta `token`, `access_token`, `accessToken` o `jwt` en JSON (`AuthenticationResponse`).

## Variables de entorno

Mismos nombres que en el README del proyecto Java:

**Client V1**

- `GOCUOTAS_COMMERCE_API_KEY` — API key de comercio (Bearer).
- `GOCUOTAS_LIQUIDACION_ID` — id para detalle JSON y CSV por id cuando usás los métodos `*FromEnvIDs`.
- Opcional: base URL por código (`ClientV1Config.BaseURL`); por defecto `https://www.gocuotas.com`.

**Redirect V1**

- `GOCUOTAS_API_KEY` — panel; Bearer en checkout y contraseña en authenticate para JWT de órdenes.
- `GOCUOTAS_EMAIL` — email del comercio (con `GOCUOTAS_API_KEY` como contraseña) si no definís `GOCUOTAS_JWT`.
- `GOCUOTAS_JWT` — si está definido, se usa como Bearer en órdenes (evita authenticate).
- `GOCUOTAS_DELIVERED_START`, `GOCUOTAS_DELIVERED_END` — rango `YYYY-MM-DD HH:mm` para listados.

## Tests

```bash
cd go
go test ./... -count=1
```

Los tests usan `httptest.Server` (sin red), cubriendo cabeceras, rutas codificadas (`a/b` → `a%2Fb`), CSV `text/plain`, errores HTTP y `OrderNotFoundError` vs `APIError` en listado 404.

## Ejemplos ejecutables (`cmd/`)

Desde `go/`:

```bash
# Client V1 — requiere GOCUOTAS_COMMERCE_API_KEY
go run ./cmd/gocuotas-client-v1/ commerce
go run ./cmd/gocuotas-client-v1/ settlements
GOCUOTAS_LIQUIDACION_ID=9001001 go run ./cmd/gocuotas-client-v1/ settlement
go run ./cmd/gocuotas-client-v1/ settlements-csv
GOCUOTAS_LIQUIDACION_ID=9001001 go run ./cmd/gocuotas-client-v1/ settlement-csv
```

```bash
# Redirect V1
GOCUOTAS_EMAIL=comercio@example.com GOCUOTAS_API_KEY=... go run ./cmd/gocuotas-redirect-v1/ authenticate
GOCUOTAS_API_KEY=... go run ./cmd/gocuotas-redirect-v1/ checkout
GOCUOTAS_JWT=... GOCUOTAS_DELIVERED_START="2021-01-19 10:00" GOCUOTAS_DELIVERED_END="2024-12-19 20:00" go run ./cmd/gocuotas-redirect-v1/ list-orders
GOCUOTAS_JWT=... go run ./cmd/gocuotas-redirect-v1/ get-order 80001001
# Sin JWT: usa email + api key en memoria (una authenticate por proceso)
GOCUOTAS_EMAIL=... GOCUOTAS_API_KEY=... go run ./cmd/gocuotas-redirect-v1/ get-order 80001001
```

Los ids de ejemplo son **ficticios** (misma convención que en `java/README.md`).

## Snippet (Client V1)

```go
ctx := context.Background()
client := clientv1.NewClient()

info, err := client.GetCommerceFromEnv(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println(info.Name, info.Cuit)

csv, err := client.GetSettlementPlainTextFromEnv(ctx, "9001001")
if err != nil {
    log.Fatal(err)
}
fmt.Print(csv)
```

## Snippet (Redirect)

```go
ctx := context.Background()
c := redirect.NewClient()

auth, err := c.Authenticate(ctx, os.Getenv("GOCUOTAS_EMAIL"), os.Getenv("GOCUOTAS_API_KEY"))
if err != nil {
    log.Fatal(err)
}
_ = auth.Token // usar como Bearer en órdenes o definir GOCUOTAS_JWT

raw, err := c.ListOrdersJSON(ctx, auth.Token, "2021-01-19 10:00", "2024-12-19 20:00")
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(raw))
```

## Errores

- Respuestas no 2xx genéricas: `*api.APIError` (`StatusCode`, `ResponseBody`, `Error()`).
- GET/DELETE orden inexistente (404): `*redirect.OrderNotFoundError` con `OrderID` y `ResponseBody`; `ListOrders` con 404 sigue siendo `*api.APIError` (igual que en Java).

## Documentación de API

- [API Redirect V1 (Stoplight)](https://gocuotas-api.stoplight.io/docs/gocuotas/dae8814842a1e-api-redirect-v1)

Para Client V1, misma base host que producción (`https://www.gocuotas.com`) salvo que configures otra `BaseURL` en `ClientV1Config` / `redirect.Config`.

## Licencia

**MIT** — Copyright (c) 2026 humen-io. Mismos términos que el cliente Java del monorepo; el texto legal completo está en [`java/LICENSE`](../java/LICENSE).

# Modos de falla del Catálogo

Cada falla posible del servicio, qué la impide y cómo se verifica. La columna **Hito** indica cuándo queda cubierta la falla: el Hito 1 trabaja
con datos simulados y el Hito 2 con Productos y Stock reales.

Las verificaciones marcadas como _e2e_ están en `scripts/e2e.mjs` y su resultado se
compara con `apps/api/e2e/testdata/base.golden.md`.

## Base del servicio

| #   | Falla                                          | Qué la impide                                                                            | Cómo se verifica                                        | Hito |
| --- | ---------------------------------------------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------- | ---- |
| 1   | PostgreSQL caído o inalcanzable                | `/readyz` responde 503 y la API no se marca como lista                                   | e2e: API apuntando a un puerto cerrado, `/readyz` = 503 | 1    |
| 2   | Falta una variable de entorno obligatoria      | El proceso valida la configuración al arrancar y termina con código 1                    | e2e: arranque sin `DATABASE_URL`, código de salida 1    | 1    |
| 3   | Documentación y contrato expuestos en `prod`   | Con `API_ENV=prod` no se registran `/api/docs` ni `/api/openapi`                         | e2e: ambas rutas responden 404 en `prod`                | 1    |
| 4   | La API no responde y el microcomponente se cae | El cliente `*.api.ts` convierte el error HTTP en un mensaje visible, sin romper la vista | Prueba del cliente web ante HTTP 503                    | 1    |

## Consulta del catálogo

| #   | Falla                                                       | Qué la impide                                                                                                                                                                         | Cómo se verifica                                                         | Hito |
| --- | ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ | ---- |
| 5   | Aparece en el catálogo un producto que no se puede vender   | El service solo devuelve productos con `permitir_venta = true` en Productos y stock mayor a 0 en Stock                                                                                | e2e: todo ítem de `/api/catalogo` trae `disponible = true` y `stock > 0` | 1    |
| 6   | Se pide el detalle de un producto que no existe             | El handler responde 404 con `application/problem+json` (RFC 9457)                                                                                                                     | e2e: `/api/catalogo/{id}` con un id inexistente = 404 problem+json       | 1    |
| 7   | Un filtro sin resultados devuelve `null` o un error         | El service devuelve siempre una lista, vacía si no hay coincidencias                                                                                                                  | e2e: `?categoria=Inexistente` = 200 con `[]`                             | 1    |
| 8   | Un error expone trazas, SQL o rutas internas                | Los errores se responden con un mensaje fijo, sin adjuntar el error original (`huma.Error500InternalServerError("mensaje")`, sin `err`). El detalle técnico va solo al log con `slog` | e2e: el cuerpo del 404 no contiene trazas, rutas `.go` ni SQL            | 1    |
| 9   | Un precio se guarda o se envía con decimales                | Los montos son `int64` en pesos (CLP), nunca `float`                                                                                                                                  | e2e: `precio` es entero en todos los ítems                               | 1    |
| 10  | Se consulta con un identificador mal formado                | Los ids expuestos son UUID (`format: uuid` en el contrato) y Huma rechaza el resto con 422 antes del service                                                                          | e2e: id que no es UUID = 422 problem+json                                | 1    |
| 11  | Una consulta pide el catálogo completo y satura el servicio | Paginación obligatoria con tamaño máximo declarado en el contrato (`maximum`)                                                                                                         | e2e: tamaño mayor al máximo = 422                                        | 2    |

## Dependencias: Productos (grupo 3) y Stock (grupo 8)

| #   | Falla                                                                                         | Qué la impide                                                                                                                                   | Cómo se verifica                                                                       | Hito |
| --- | --------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | ---- |
| 12  | Productos o Stock está caído                                                                  | Timeout de 5 s en cada cliente HTTP. El catálogo responde 503 problem+json con un mensaje claro, sin colgar la request                          | e2e: `PRODUCTOS_URL` o `STOCK_URL` apuntando a un puerto cerrado = 503 en menos de 6 s | 2    |
| 13  | Productos o Stock responde lento                                                              | El mismo timeout de 5 s corta la llamada                                                                                                        | e2e: servidor falso que tarda 10 s = 503 dentro del plazo                              | 2    |
| 14  | Productos y Stock no coinciden (producto borrado con stock, stock de un producto inexistente) | El catálogo solo muestra la intersección. Un producto sin registro de stock se trata como no disponible                                         | e2e con servidores falsos que devuelven datos cruzados                                 | 2    |
| 15  | El stock cambia entre que se muestra el catálogo y se vende                                   | El catálogo es una vista informativa con `actualizado_en`. Quien valida y descuenta stock al vender es Registro de venta (grupo 9)              | Fuera del alcance de este servicio. Queda documentado en el contrato                   | 2    |
| 16  | Productos o Stock publica un cambio incompatible en su contrato                               | Sus contratos se copian en `contracts/externos/` y el cliente se genera desde ahí. Si el contrato cambia, la generación o la compilación fallan | CI: codegen y build                                                                    | 2    |
| 17  | Se reenvía la request sin el JWT ni el `X-Request-Id` del usuario                             | El cliente HTTP los propaga siempre en las llamadas a otros servicios                                                                           | e2e: el servidor falso registra los encabezados recibidos                              | 2    |

## Administración de la publicación

Aplica si el catálogo guarda datos propios de la vitrina (visible, destacado, orden y,
si se decide que es de Catálogo, el precio de venta).

| #   | Falla                                                           | Qué la impide                                                                                           | Cómo se verifica                                                | Hito |
| --- | --------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- | ---- |
| 18  | Un usuario sin permiso modifica la publicación                  | El endpoint exige `catalogo:administrar`, validando el JWT contra `AUTH_JWKS_URL`                       | `acceso_test.go`: matriz de acceso por rol                      | 2    |
| 19  | Dos administradores editan la misma publicación al mismo tiempo | Control de versión optimista: la escritura incluye la versión leída y la base rechaza una versión vieja | e2e: dos `PUT` simultáneos con la misma versión, uno recibe 409 | 2    |
| 20  | Un cambio en la publicación no deja rastro                      | Cada cambio registra quién, qué y cuándo en la tabla `auditoria`                                        | e2e: después del `PUT`, existe el registro de auditoría         | 2    |

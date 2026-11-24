import { execFileSync, spawn } from 'node:child_process'
import { mkdirSync, readFileSync } from 'node:fs'
import { createServer } from 'node:http'
import { resolve } from 'node:path'
import assert from 'node:assert/strict'
const ejecutar = (comando, args, opciones = {}) =>
  execFileSync(comando, args, { stdio: 'pipe', ...opciones })
mkdirSync('.tmp/cobertura', { recursive: true })
const binario = resolve(`.tmp/catalogo${process.platform === 'win32' ? '.exe' : ''}`)
ejecutar('go', ['build', '-cover', '-o', binario, './cmd/api'], { cwd: 'apps/api' })
let contenedor
let proceso
const esperar = (ms) => new Promise((r) => setTimeout(r, ms))
// UUID fijo de un producto simulado (apps/api/internal/service/catalogo.go).
const ID_EXISTENTE = '3f2c9a1e-7b4d-4c8e-9a6f-1d2e3b4c5a60'
// UUID válido que no está en el catálogo.
const ID_INEXISTENTE = '00000000-0000-4000-8000-000000000000'
// Id con el formato antiguo: el contrato lo rechaza antes de llegar al service.
const ID_MAL_FORMADO = 'prod-101'
// Servidor falso del Servicio de Stock (contracts/externos/stock.yaml). Responde
// GET /api/stock/productos/{producto_id} y guarda las credenciales que recibe.
const STOCK = {
  [ID_EXISTENTE]: 15,
  '8a1d4e7b-2c3f-4a5b-8c9d-0e1f2a3b4c5d': 32,
  'c5e8f1a2-9b3c-4d6e-a7f8-2b3c4d5e6f70': 8,
}
const recibidasPorStock = []
const servidorStock = createServer((req, res) => {
  recibidasPorStock.push({
    autorizacion: req.headers.authorization ?? '',
    requestId: req.headers['x-request-id'] ?? '',
  })
  const id = req.url.match(/^\/api\/stock\/productos\/([0-9a-f-]{36})$/)?.[1]
  const stock = id ? STOCK[id] : undefined
  if (stock === undefined) {
    res.writeHead(404, { 'content-type': 'application/problem+json' })
    res.end('{"title":"Not Found","status":404}')
    return
  }
  res.writeHead(200, { 'content-type': 'application/json' })
  res.end(
    JSON.stringify({
      producto_id: id,
      stock_actual: stock,
      actualizado_en: '2026-11-22T12:00:00Z',
    }),
  )
})
await new Promise((r) => servidorStock.listen(0, '127.0.0.1', r))
const STOCK_URL = `http://127.0.0.1:${servidorStock.address().port}`
// Puerto cerrado: simula el Servicio de Stock caído (modo de falla 12).
const STOCK_CAIDO = 'http://127.0.0.1:1'
const entornoApi = (entorno, db, stockUrl = STOCK_URL) => ({
  ...process.env,
  API_ENV: entorno,
  API_PORT: '8080',
  DATABASE_URL: db,
  AUTH_JWKS_URL: 'http://identidad.invalid/jwks',
  FRONTEND_URL: 'http://localhost:5120',
  STOCK_URL: stockUrl,
  TZ: 'America/Santiago',
  GOCOVERDIR: resolve('.tmp/cobertura'),
})
async function arrancar(entorno, db, stockUrl) {
  proceso = spawn(binario, [], {
    env: entornoApi(entorno, db, stockUrl),
    stdio: 'ignore',
  })
  for (let i = 0; i < 100; i++) {
    if (proceso.exitCode !== null) throw new Error('La API terminó al iniciar')
    try {
      if ((await fetch('http://localhost:8080/healthz')).ok) return
    } catch {}
    await esperar(100)
  }
  throw new Error('La API no inició')
}
async function detener() {
  if (!proceso || proceso.exitCode !== null) return
  const fin = new Promise((r) => proceso.once('exit', r))
  proceso.kill('SIGTERM')
  await fin
}
// Arranca la API sin una variable obligatoria y devuelve su código de salida.
async function codigoSalidaSin(variable, db) {
  const env = entornoApi('test', db)
  delete env[variable]
  const hijo = spawn(binario, [], { env, stdio: 'ignore' })
  return Promise.race([
    new Promise((r) => hijo.once('exit', (codigo) => r(codigo))),
    esperar(5000).then(() => {
      hijo.kill('SIGKILL')
      return 'no terminó en 5 s'
    }),
  ])
}
async function pedir(ruta) {
  const res = await fetch(`http://localhost:8080${ruta}`)
  return {
    estado: res.status,
    tipo: res.headers.get('content-type') ?? '',
    cuerpo: await res.json(),
  }
}
// Modos de falla 5 a 11 y 17 de docs/modos-de-falla.md.
async function revisarCatalogo(lineas) {
  const todos = await pedir('/api/catalogo')
  assert.equal(todos.estado, 200)
  assert.ok(Array.isArray(todos.cuerpo) && todos.cuerpo.length > 0, 'Catálogo vacío')
  for (const item of todos.cuerpo) {
    assert.ok(item.disponible && item.stock > 0, `${item.sku} no es vendible`)
    assert.ok(Number.isInteger(item.precio), `${item.sku} tiene un precio no entero`)
  }
  lineas.push(
    `/api/catalogo: ${todos.estado}, ${todos.cuerpo.length} productos vendibles`,
  )

  // Modo de falla 11: paginación, con el total en X-Total-Count.
  const pagina = await fetch('http://localhost:8080/api/catalogo?pagina=2&tamano=2')
  assert.equal(pagina.status, 200)
  assert.equal((await pagina.json()).length, 1)
  assert.equal(pagina.headers.get('x-total-count'), '3')
  lineas.push(`/api/catalogo?pagina=2&tamano=2: ${pagina.status}, 1 de 3 productos`)

  const categoria = 'Periféricos'
  const filtrados = await pedir(
    `/api/catalogo?categoria=${encodeURIComponent(categoria)}`,
  )
  assert.equal(filtrados.estado, 200)
  assert.ok(filtrados.cuerpo.every((item) => item.categoria === categoria))
  lineas.push(
    `/api/catalogo?categoria=${categoria}: ${filtrados.estado}, ${filtrados.cuerpo.length} productos`,
  )

  // categoria_id reemplaza al filtro por nombre, que queda obsoleto.
  const ID_ACCESORIOS = '1e2f3a4b-5c6d-4e7f-8a9b-0c1d2e3f4a5b'
  const porId = await pedir(`/api/catalogo?categoria_id=${ID_ACCESORIOS}`)
  assert.equal(porId.estado, 200)
  assert.ok(porId.cuerpo.length > 0, 'Sin productos en Accesorios')
  assert.ok(porId.cuerpo.every((item) => item.categoria_ids.includes(ID_ACCESORIOS)))
  lineas.push(
    `/api/catalogo?categoria_id={id}: ${porId.estado}, ${porId.cuerpo.length} productos`,
  )

  const sinResultados = await pedir('/api/catalogo?categoria=Inexistente')
  assert.equal(sinResultados.estado, 200)
  assert.deepEqual(sinResultados.cuerpo, [], 'Un filtro sin resultados debe devolver []')
  lineas.push(`/api/catalogo?categoria=Inexistente: ${sinResultados.estado}, lista vacía`)

  const detalle = await pedir(`/api/catalogo/${ID_EXISTENTE}`)
  assert.equal(detalle.estado, 200)
  assert.equal(detalle.cuerpo.id, ID_EXISTENTE)
  lineas.push(`/api/catalogo/{id} existente: ${detalle.estado}`)

  const noExiste = await pedir(`/api/catalogo/${ID_INEXISTENTE}`)
  assert.equal(noExiste.estado, 404)
  assert.ok(noExiste.tipo.startsWith('application/problem+json'), noExiste.tipo)
  assert.doesNotMatch(
    JSON.stringify(noExiste.cuerpo),
    /goroutine|\.go:\d|select |insert |pgx|sqlstate/i,
    'El error expone detalles internos',
  )
  lineas.push(`/api/catalogo/{id} inexistente: ${noExiste.estado} problem+json`)

  // Modo de falla 10: id que no es UUID.
  const malFormado = await pedir(`/api/catalogo/${ID_MAL_FORMADO}`)
  assert.equal(malFormado.estado, 422)
  assert.ok(malFormado.tipo.startsWith('application/problem+json'), malFormado.tipo)
  lineas.push(`/api/catalogo/{id} no UUID: ${malFormado.estado} problem+json`)

  const categorias = await pedir('/api/categorias')
  assert.equal(categorias.estado, 200)
  assert.ok(categorias.cuerpo.length > 0, 'Sin categorías')
  lineas.push(
    `/api/categorias: ${categorias.estado}, ${categorias.cuerpo.length} categorías`,
  )

  // Modo de falla 17: el JWT y el X-Request-Id llegan a Stock.
  recibidasPorStock.length = 0
  const conSesion = await fetch(`http://localhost:8080/api/catalogo/${ID_EXISTENTE}`, {
    headers: { cookie: 'sesion=jwt-e2e', 'x-request-id': 'e2e-req-1' },
  })
  assert.equal(conSesion.status, 200)
  assert.equal(conSesion.headers.get('x-request-id'), 'e2e-req-1')
  assert.deepEqual(recibidasPorStock, [
    { autorizacion: 'Bearer jwt-e2e', requestId: 'e2e-req-1' },
  ])
  lineas.push('Reenvío a Stock: Bearer y X-Request-Id propagados')
}
try {
  let db = process.env.TEST_DATABASE_URL
  if (!db) {
    contenedor = ejecutar('docker', [
      'run',
      '-d',
      '--rm',
      '-e',
      'POSTGRES_PASSWORD=prueba-local-efimera',
      '-p',
      '127.0.0.1::5432',
      'postgres:17.11@sha256:ff87138949ad9a46f32ff23bd6852b52ce024a13dffe9b4087be98292cdd3b4f',
    ])
      .toString()
      .trim()
    const puerto = ejecutar('docker', ['port', contenedor, '5432'])
      .toString()
      .trim()
      .split(':')
      .at(-1)
    db = `postgres://postgres:prueba-local-efimera@localhost:${puerto}/postgres?sslmode=disable`
    let listo = false
    for (let i = 0; i < 60; i++) {
      try {
        ejecutar('docker', [
          'exec',
          contenedor,
          'pg_isready',
          '-h',
          '127.0.0.1',
          '-U',
          'postgres',
        ])
        listo = true
        break
      } catch {}
      await esperar(500)
    }
    assert.ok(listo, 'PostgreSQL no inició')
  }
  await arrancar('test', db)
  const lineas = []
  for (const ruta of ['/healthz', '/readyz', '/api/estado', '/api/docs']) {
    const res = await fetch(`http://localhost:8080${ruta}`)
    assert.equal(res.status, 200)
    lineas.push(`${ruta}: ${res.status}`)
    if (ruta === '/api/estado') assert.equal((await res.json()).servicio, 'catalogo')
  }
  await revisarCatalogo(lineas)
  await detener()
  await arrancar('prod', db)
  for (const ruta of ['/api/docs', '/api/openapi.yaml']) {
    const res = await fetch(`http://localhost:8080${ruta}`)
    assert.equal(res.status, 404)
    lineas.push(`prod ${ruta}: ${res.status}`)
  }
  await detener()
  // Modo de falla 2: configuración incompleta.
  assert.equal(await codigoSalidaSin('DATABASE_URL', db), 1)
  lineas.push('Configuración incompleta (sin DATABASE_URL): código 1')
  // Modo de falla 12: Stock caído responde 503 rápido, sin colgar la request.
  await arrancar('test', db, STOCK_CAIDO)
  const inicio = Date.now()
  const sinStock = await pedir('/api/catalogo')
  assert.equal(sinStock.estado, 503)
  assert.ok(sinStock.tipo.startsWith('application/problem+json'), sinStock.tipo)
  assert.ok(Date.now() - inicio < 6000, 'La request quedó colgada')
  lineas.push(`Stock no disponible: ${sinStock.estado} problem+json`)
  await detener()
  await arrancar(
    'test',
    'postgres://postgres:prueba@127.0.0.1:1/postgres?sslmode=disable',
  )
  assert.equal((await fetch('http://localhost:8080/readyz')).status, 503)
  lineas.push('PostgreSQL no disponible: 503')
  assert.equal(
    lineas.join('\n') + '\n',
    readFileSync('apps/api/e2e/testdata/base.golden.md', 'utf8'),
  )
} finally {
  await detener()
  servidorStock.close()
  if (contenedor) ejecutar('docker', ['stop', contenedor])
}
ejecutar('go', [
  'tool',
  'covdata',
  'textfmt',
  '-i',
  resolve('.tmp/cobertura'),
  '-o',
  resolve('coverage.out'),
])

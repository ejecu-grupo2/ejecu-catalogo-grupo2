import { execFileSync, spawn } from 'node:child_process'
import { mkdirSync, readFileSync } from 'node:fs'
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
const entornoApi = (entorno, db) => ({
  ...process.env,
  API_ENV: entorno,
  API_PORT: '8080',
  DATABASE_URL: db,
  AUTH_JWKS_URL: 'http://identidad.invalid/jwks',
  FRONTEND_URL: 'http://localhost:5120',
  TZ: 'America/Santiago',
  GOCOVERDIR: resolve('.tmp/cobertura'),
})
async function arrancar(entorno, db) {
  proceso = spawn(binario, [], { env: entornoApi(entorno, db), stdio: 'ignore' })
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
// Modos de falla 5 a 10 de docs/modos-de-falla.md.
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

  const categoria = 'Periféricos'
  const filtrados = await pedir(
    `/api/catalogo?categoria=${encodeURIComponent(categoria)}`,
  )
  assert.equal(filtrados.estado, 200)
  assert.ok(filtrados.cuerpo.every((item) => item.categoria === categoria))
  lineas.push(
    `/api/catalogo?categoria=${categoria}: ${filtrados.estado}, ${filtrados.cuerpo.length} productos`,
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

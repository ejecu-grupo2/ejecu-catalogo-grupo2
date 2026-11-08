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
async function arrancar(entorno, db) {
  proceso = spawn(binario, [], {
    env: {
      ...process.env,
      API_ENV: entorno,
      API_PORT: '8080',
      DATABASE_URL: db,
      AUTH_JWKS_URL: 'http://identidad.invalid/jwks',
      FRONTEND_URL: 'http://localhost:5120',
      TZ: 'America/Santiago',
      GOCOVERDIR: resolve('.tmp/cobertura'),
    },
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
  await detener()
  await arrancar('prod', db)
  for (const ruta of ['/api/docs', '/api/openapi.yaml']) {
    const res = await fetch(`http://localhost:8080${ruta}`)
    assert.equal(res.status, 404)
    lineas.push(`prod ${ruta}: ${res.status}`)
  }
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

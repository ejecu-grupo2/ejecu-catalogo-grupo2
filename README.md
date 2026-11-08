# Catálogo · Grupo 2

Base de trabajo construida desde las reglas del DAS. No es una plantilla oficial entregada por el curso ni implementa todavía el dominio del catálogo.

## Requisitos

Go 1.26.10, Node 22.22.0, pnpm 10.13.1, Docker Desktop iniciado y Gitleaks 8.30.1. Las versiones se fijan exactamente; los parches de seguridad respecto al DAS original se registran en [actualizaciones de dependencias](docs/actualizaciones-dependencias.md).

## Preparación

```sh
pnpm install --frozen-lockfile
pnpm exec lefthook install
cd apps/api
go mod download
cd ../..
```

Copiar `.env.example` a `.env`, reemplazar usuario, contraseña, base y URLs. Los valores de DATABASE_URL deben coincidir con POSTGRES_USER, POSTGRES_PASSWORD y POSTGRES_DB. No subir `.env` al repositorio.

```sh
docker compose up -d --build
```

Web: http://localhost:5120. API: http://localhost:5121. Documentación: http://localhost:5121/api/docs. Los puertos 5120/5121 son provisionales: confirmar el número N del servicio con el profesor.

## Validación

```sh
pnpm verify
pnpm codegen
pnpm test:e2e:api
```

El E2E usa un contenedor PostgreSQL efímero o `TEST_DATABASE_URL`, inicia el binario real y compara un reporte golden. Es una prueba de infraestructura; los escenarios de negocio se agregarán cuando se conozca el contrato. La cobertura Go se obtiene del binario compilado con `-cover` y la del frontend de Vitest.

`pnpm codegen` genera `contracts/openapi.yaml` y `apps/web/src/api/schema.d.ts`; ambos se commitean. `make generate` en `apps/api` genera el repositorio SQL con sqlc. Handler → Service → Store es la estructura de las futuras funciones de negocio. `/api/estado` solo informa el estado de esta base; no expone datos del catálogo.

## GitHub y SonarQube

Para conectar SonarQube local con los runners de GitHub mediante el túnel HTTPS del ejemplo, seguir [esta guía](docs/sonarqube-docker.md).

La CI exige `calidad`, `seguridad` e `imagen`. No se omite SonarQube si faltan sus secrets: calidad quedará bloqueado hasta configurar el proyecto. Ver `docs/solicitud-sonarqube.md`.

Después del primer push, configurar protección de `main`: PR obligatorio, al menos una aprobación de otra persona y los tres checks requeridos. No habilitar bypass para integrar PR con fallas.

## Pendientes del Hito 0

- Confirmar número del servicio, alcance y dependencias con el documento de componentes.
- Crear SonarQube y cargar SONAR_TOKEN y SONAR_HOST_URL directamente en GitHub Actions Secrets.
- Confirmar la clave SonarQube propuesta y definir New Code y Quality Gate.
- Agregar seed cuando exista esquema de catálogo y app de desarrollo con mocks MSW.
- Completar autenticación JWT/JWKS, permisos y logs por request antes de agregar endpoints de negocio.
- Publicación del paquete UI, SBOM y firma de imágenes pendiente.
- Ejecutar Docker, escáneres y pipeline; guardar los enlaces de evidencia y proteger main.

Reglas completas: [DAS](docs/DAS.md). Modos de falla: [documento](docs/modos-de-falla.md).

# Catálogo · Grupo 2

Repositorio del servicio de catálogo para el curso DAS. Está preparado el entorno
con Go, React y PostgreSQL, junto con Docker y el pipeline de GitHub Actions.
Las funciones propias del catálogo todavía están pendientes.

## Requisitos

- Go 1.26.10
- Node.js 22.22.0 y pnpm 10.13.1
- Docker Desktop con contenedores Linux
- Gitleaks 8.30.1 para el control de secretos en los commits

Las versiones están fijadas en el proyecto. Los cambios de seguridad respecto al
DAS original se registran en [actualizaciones de dependencias](docs/actualizaciones-dependencias.md).

## Levantar el proyecto

Desde la carpeta del repositorio:

```sh
pnpm install --frozen-lockfile
pnpm exec lefthook install
cd apps/api
go mod download
cd ../..
```

Copia `.env.example` a `.env` y completa los valores. En PowerShell:

```powershell
Copy-Item .env.example .env
```

El usuario, la contraseña y el nombre de la base en `DATABASE_URL` deben coincidir
con `POSTGRES_USER`, `POSTGRES_PASSWORD` y `POSTGRES_DB`. Dentro de Docker, el host
de la base es `postgres`. El archivo `.env` no se sube al repositorio.

Con Docker Desktop abierto:

```sh
docker compose up -d --build
```

| Servicio                | Dirección local                |
| ----------------------- | ------------------------------ |
| Frontend                | http://localhost:5120          |
| API                     | http://localhost:5121          |
| Documentación de la API | http://localhost:5121/api/docs |

Los puertos 5120 y 5121 son los actuales. Falta confirmar el número de servicio
asignado al grupo para dejar el rango definitivo.

## Organización del código

- `apps/api`: backend Go, separado en handlers, servicios y acceso a PostgreSQL.
- `apps/web`: frontend React y componente que consulta el estado de la API.
- `contracts/openapi.yaml`: contrato generado desde los handlers de Huma.
- `docs`: reglas del curso, modos de falla e instrucciones de configuración.

La API expone `/healthz` para comprobar que el proceso está activo, `/readyz` para
comprobar PostgreSQL y `/api/estado` para consultar el estado del servicio. Aún no
hay endpoints de negocio del catálogo.

## Comprobaciones antes de subir cambios

```sh
pnpm verify
pnpm codegen
pnpm test:e2e:api
```

`pnpm verify` revisa formato, lint, tipos, pruebas y compilación. El E2E necesita
Docker: levanta PostgreSQL, ejecuta la API real y compara el resultado con el
reporte golden. También puede usar una base indicada en `TEST_DATABASE_URL`.

Después de cambiar el contrato, `pnpm codegen` actualiza el OpenAPI y los tipos de
`apps/web/src/api/schema.d.ts`. Ambos archivos se incluyen en el commit.
Las consultas SQL se generan con `make generate` desde `apps/api`.

## CI y SonarQube

Cada PR y cada push a `main` ejecuta tres jobs:

- `calidad`: validaciones del código, contrato, E2E y Quality Gate de SonarQube.
- `seguridad`: Gitleaks, auditoría de dependencias, govulncheck, Hadolint y Trivy.
- `imagen`: construcción y revisión de las imágenes Docker.

`main` está protegida. Los cambios entran por PR, con al menos una aprobación de
otra persona del grupo y los tres checks en verde.

SonarQube corre en Docker y GitHub accede mediante un túnel HTTPS de Cloudflare.
La clave del proyecto es `ejecu-catalogo-grupo2`. El token y la URL del túnel se
guardan como `SONAR_TOKEN` y `SONAR_HOST_URL` en Actions Secrets.
Para iniciarlo o recuperar la conexión después de apagar el equipo, sigue la
[guía de SonarQube](docs/sonarqube-docker.md).

La [validación de main del 8 de noviembre de 2026](https://github.com/ejecu-grupo2/ejecu-catalogo-grupo2/actions/runs/42519179046)
terminó con los tres jobs aprobados. El primer análisis de SonarQube obtuvo un
Quality Gate aprobado, 67,5% de cobertura, 0% de duplicación y cero issues.

## Trabajo pendiente

- Definir el modelo, los endpoints y las dependencias del catálogo.
- Incorporar autenticación JWT/JWKS y permisos en los endpoints de negocio.
- Preparar datos de demostración y mocks MSW.
- Completar la publicación del paquete de componentes, el SBOM y la firma de imágenes.
- Acordar el análisis nativo de PR exigido por el DAS, que Community Build no incluye.

## Documentación

- [Lineamientos DAS](docs/DAS.md)
- [Modos de falla](docs/modos-de-falla.md)
- [SonarQube con Docker y Cloudflare](docs/sonarqube-docker.md)
- [Actualizaciones de dependencias](docs/actualizaciones-dependencias.md)

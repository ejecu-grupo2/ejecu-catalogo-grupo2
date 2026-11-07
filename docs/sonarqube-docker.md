# SonarQube con Docker

Esta configuración levanta una instancia local independiente de la base del catálogo.
Usa SonarQube Community Build y su propia PostgreSQL, con volúmenes persistentes.
El token de análisis se genera después de iniciar SonarQube; la contraseña de
PostgreSQL no es ese token.

## 1. Iniciar Docker Desktop

Seleccionar contenedores Linux y esperar a que Docker esté funcionando:

```powershell
docker info
```

Si este comando falla, resolver primero el inicio de Docker Desktop.

## 2. Crear la configuración local e iniciar

Desde la raíz del repositorio, en PowerShell:

```powershell
powershell -NoProfile -File scripts/preparar-sonarqube.ps1
docker compose --env-file .env.sonar -f compose.sonarqube.yaml up -d
docker compose --env-file .env.sonar -f compose.sonarqube.yaml logs -f sonarqube
```

El script crea `.env.sonar` con una clave aleatoria. Git ignora este archivo.
No ejecutar `docker compose config` para compartir su salida: contiene la clave
resuelta. La primera descarga y el inicio pueden tardar varios minutos.
Ctrl+C sale de la vista de logs; los contenedores continúan ejecutándose.

Abrir http://localhost:9000. El acceso inicial es `admin` / `admin`; cambiar la
contraseña cuando SonarQube lo solicite. No publicar el servicio en internet con
estas credenciales. La configuración solo escucha en el equipo local.

Si Elasticsearch termina por `vm.max_map_count`, revisar los requisitos Linux
oficiales y ajustar ese parámetro en el host Linux de Docker Desktop/WSL2. No se
desactivan las comprobaciones de Elasticsearch en esta configuración.

## 3. Crear el proyecto

En SonarQube, seleccionar crear proyecto local/manual y completar:

- Nombre: `Catálogo · Grupo 2`.
- Clave: `ejecu-catalogo-grupo2`, igual que `sonar-project.properties`.
- Rama principal: `main`.
- Definición de New Code: acordar y configurar antes del primer análisis.

Crear/asignar el Quality Gate del DAS: cero bugs y vulnerabilidades nuevas,
hotspots revisados al 100%, duplicación menor al 3% y cobertura mínima del 60%.
Los nombres de las métricas pueden variar según el modo y versión de SonarQube.

En la configuración del análisis o en My Account > Security, generar un token
de análisis limitado al proyecto. Copiarlo directamente al almacenamiento de
secretos; no guardarlo en archivos versionados ni enviarlo por chat.

## 4. Conectar con el pipeline

El workflow existente lee `SONAR_TOKEN` y `SONAR_HOST_URL` desde GitHub Actions
Secrets. El archivo `sonar-project.properties` ya existe y no lleva el token.

`http://localhost:9000` sirve para acceder desde este computador. No funciona
como `SONAR_HOST_URL` para los runners alojados por GitHub:
en ellos localhost es otro equipo.

Para usar esta instancia en CI hay que elegir y configurar una de estas opciones:

- Alojar Docker en un servidor al que accedan los runners, con HTTPS y acceso
  controlado. Guardar su URL real como `SONAR_HOST_URL`.
- Usar un runner propio que alcance esta instancia. Requiere preparar el runner
  y adaptar el workflow; no se instala ni cambia automáticamente aquí.

Community Build permite análisis de la rama principal, pero no análisis de PR
nativo. El DAS exige análisis en cada PR; para cumplirlo hay que confirmar con el
profesor una edición con soporte de PR o una alternativa explícitamente aprobada.
Este Compose local no resuelve esa diferencia ni el bloqueo de facturación de
GitHub Actions. No se omite el Quality Gate ni se rebaja la protección de main.

## Operación

```powershell
docker compose --env-file .env.sonar -f compose.sonarqube.yaml ps
docker compose --env-file .env.sonar -f compose.sonarqube.yaml stop
docker compose --env-file .env.sonar -f compose.sonarqube.yaml start
```

Conservar `.env.sonar` y los volúmenes. No usar `down -v`: elimina los datos.

## Referencias

- [Imagen oficial y versiones](https://hub.docker.com/_/sonarqube).
- [Instalación con Docker](https://docs.sonarsource.com/sonarqube-community-build/server-installation/from-docker-image/set-up-and-start-container).
- [Requisitos Linux](https://docs.sonarsource.com/sonarqube-community-build/server-installation/pre-installation/linux).
- [Funciones por edición](https://docs.sonarsource.com/sonarqube-community-build/feature-comparison-table).

## Runner Ubuntu del equipo

El workflow de esta rama usa las etiquetas predeterminadas `[self-hosted, Linux, X64]`,
porque el runner se registró sin etiqueta adicional. Los tres jobs necesitan ese
runner disponible; deben mantenerse Ubuntu, `./run.sh` y Docker Desktop activos.
La configuración inicial comprueba Docker y la URL de SonarQube sin mostrar el token.

Desde una segunda terminal Ubuntu, comprobar `docker info` y
`curl --max-time 30 http://localhost:9000/api/system/status` antes de ejecutar.
Si responde UP, guardar `http://localhost:9000` como `SONAR_HOST_URL`. Esto solo es
válido para este runner local.

El evento `workflow_dispatch` permite ejecutar manualmente CI una vez que el archivo
esté integrado en la rama principal. El análisis de PR sigue requiriendo una edición
compatible; cambiar el runner no añade esa función a Community Build.

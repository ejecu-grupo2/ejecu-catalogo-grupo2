# SonarQube con Docker y GitHub Actions

Configuracion adaptada de `ejemplo-pingeso`: SonarQube corre localmente en Docker;
Cloudflare publica una URL HTTPS temporal; los jobs se ejecutan en `ubuntu-24.04`
de GitHub. No se necesita el runner personal de WSL para este esquema.

## 1. Preparar e iniciar SonarQube

Abrir Docker Desktop. Desde la raiz del proyecto, en PowerShell:

```powershell
powershell -NoProfile -File scripts/preparar-sonarqube.ps1
docker compose --env-file .env.sonar -f compose.sonarqube.yaml up -d sonar-db sonarqube
```

El script conserva `.env.sonar` si ya existe. No borrarlo: contiene la clave de
PostgreSQL, que es distinta del token de analisis. Git ignora este archivo.
Los volumenes conservan la base, proyectos y configuracion de SonarQube.

Abrir http://localhost:9000. En el primer acceso, usar `admin` / `admin` y cambiar
la clave. Si el proyecto ya existe y hay un token guardado, conservarlos.

Crear un proyecto local con clave `ejecu-catalogo-grupo2` y rama principal `main`.
Definir New Code antes del primer analisis y asignar el Quality Gate del DAS:
0 bugs y vulnerabilidades nuevos, 100% de hotspots revisados, duplicacion menor
al 3% y cobertura minima del 60%.

Comprobar que el servicio responde antes de abrir el tunel:

```powershell
curl.exe --max-time 30 http://127.0.0.1:9000/api/system/status
```

Debe devolver `"status":"UP"`. Si falla, revisar:

```powershell
docker compose --env-file .env.sonar -f compose.sonarqube.yaml logs --tail 80 sonarqube
```

## 2. Iniciar el tunel del ejemplo

El perfil `tunnel` es opcional: no publica SonarQube al iniciar los otros servicios.
Al iniciarlo, la interfaz y API de SonarQube seran accesibles mediante una URL
publica temporal. Mantener autenticacion y una clave de administrador propia.

```powershell
docker compose --env-file .env.sonar -f compose.sonarqube.yaml --profile tunnel up -d sonar-tunnel
docker compose --env-file .env.sonar -f compose.sonarqube.yaml logs -f sonar-tunnel
```

Buscar en los logs una URL como `https://nombre-aleatorio.trycloudflare.com`.
Copiar la URL completa. Ctrl+C sale de los logs, pero el tunel sigue activo en
Docker. El contenedor conecta a `http://sonarqube:9000` por la red interna de
Compose; no depende de `--network=host` ni de localhost entre Windows y WSL.

Comprobar la URL publica sustituyendo el dominio del ejemplo:

```powershell
curl.exe --max-time 30 https://nombre-aleatorio.trycloudflare.com/api/system/status
```

Debe responder UP. Los Quick Tunnels son temporales para pruebas; la URL puede
cambiar al recrearlos. Si cambia, actualizar el secret correspondiente.

## 3. Configurar los secrets de GitHub

En el repositorio: Settings > Secrets and variables > Actions.

| Nombre           | Valor                                                           |
| ---------------- | --------------------------------------------------------------- |
| `SONAR_TOKEN`    | Token de analisis del proyecto, ya generado en SonarQube.       |
| `SONAR_HOST_URL` | URL HTTPS publica del tunel; reemplaza `http://localhost:9000`. |

El token nunca se pega en `sonar-project.properties`, comandos compartidos ni
archivos versionados. La clave `ejecu-catalogo-grupo2` ya esta configurada en el
archivo de propiedades. El pipeline normaliza espacios accidentales y enmascara
el token antes de usarlo, siguiendo la idea del ejemplo sin insertarlo directamente
en el codigo del script de shell.

## 4. Pipeline

Los tres jobs usan runners de GitHub, no `self-hosted`:

- `calidad`: formato, lint, tipos, build, pruebas, contrato, E2E con cobertura,
  comprobacion de URL publica, SonarQube Scan y Quality Gate.
- `seguridad`: Gitleaks por Docker, govulncheck, auditoria npm, Hadolint y Trivy fs.
- `imagen`: espera calidad y seguridad, construye API/web/migraciones y analiza
  las imagenes API y web con Trivy y API con Dockle.

Los cambios se revisan en PR; main permanece protegida. Tras integrar el workflow,
Actions > ci permite ejecutar manualmente con Run workflow o se inicia al hacer
push a main y abrir/actualizar PR. Mantener Docker Desktop, SonarQube y el tunel
encendidos durante el analisis; puede detenerse el runner personal con Ctrl+C.

Community Build no ofrece analisis nativo de PR. El ejemplo tambien utiliza
Community y, por tanto, no elimina esa limitacion: para cumplir completamente el
DAS hay que acordar una edicion compatible o una alternativa aprobada. No se
omite SonarQube en PR ni se acepta un Quality Gate fallido.

## 5. Detener y conservar los datos

Para cerrar el acceso publico al terminar:

```powershell
docker compose --env-file .env.sonar -f compose.sonarqube.yaml stop sonar-tunnel
```

Para detener tambien SonarQube y su base:

```powershell
docker compose --env-file .env.sonar -f compose.sonarqube.yaml stop sonarqube sonar-db
```

No usar `down -v`: elimina los datos persistentes. No compartir la salida de
`docker compose config` porque expande la clave de PostgreSQL.

## Referencias

- [Imagen oficial SonarQube](https://hub.docker.com/_/sonarqube).
- [Docker Compose para SonarQube](https://docs.sonarsource.com/sonarqube-community-build/server-installation/from-docker-image/set-up-and-start-container).
- [Cloudflare Quick Tunnels](https://developers.cloudflare.com/tunnel/get-started/quick-tunnels/).
- [Funciones por edicion SonarQube](https://docs.sonarsource.com/sonarqube-community-build/feature-comparison-table).

$ErrorActionPreference = 'Stop'
$raizProyecto = Split-Path -Parent $PSScriptRoot
$archivoEntorno = Join-Path $raizProyecto '.env.sonar'

if (Test-Path -LiteralPath $archivoEntorno) {
  Write-Host '.env.sonar ya existe; se conserva la configuración actual.'
  exit 0
}

$bytesAleatorios = New-Object byte[] 32
$generador = [System.Security.Cryptography.RandomNumberGenerator]::Create()
try {
  $generador.GetBytes($bytesAleatorios)
} finally {
  $generador.Dispose()
}
$claveBase = [Convert]::ToBase64String($bytesAleatorios)
$contenido = "SONAR_DB_PASSWORD=$claveBase`n"
$utf8 = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText($archivoEntorno, $contenido, $utf8)
Write-Host 'Creado .env.sonar con una clave aleatoria para PostgreSQL. No subir este archivo.'

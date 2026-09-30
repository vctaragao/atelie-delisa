@echo off
setlocal enabledelayedexpansion

REM =================================================================
REM  ATELIE DELISA - PRODUCAO
REM
REM  Atualiza a main para a ultima versao publicada e sobe os
REM  containers de producao, reconstruindo as imagens.
REM
REM  Dados: volume "atelie-delisa-prod_atelie-data".
REM  Esses sao os dados REAIS do atelie - nunca apague esse volume.
REM
REM  O atalho precisa apontar para a copia deste arquivo que
REM  esta na pasta de PRODUCAO, a da branch main. A pasta vem da
REM  localizacao do arquivo, entao a copia da pasta de
REM  desenvolvimento subiria producao a partir da development.
REM
REM  Sem acentos de proposito: o cmd.exe erra ao interpretar
REM  caracteres multibyte em arquivos .bat.
REM =================================================================

REM  A pasta do repositorio e descoberta a partir da localizacao deste
REM  arquivo: ele vive em <repo>\scripts\, entao o repo e o diretorio
REM  acima. Assim o script funciona em qualquer conta de usuario, sem
REM  nome de pasta escrito dentro dele. O `for` expande o ".." para o
REM  caminho absoluto, para as mensagens na tela nao mostrarem o "..".
for %%i in ("%~dp0..") do set "REPO=%%~fi"
set "PROJETO=atelie-delisa-prod"
set "URL=http://localhost:8081"

title Atelie Delisa - PRODUCAO

echo.
echo ===============================================
echo   ATELIE DELISA - PRODUCAO
echo ===============================================
echo   Pasta : %REPO%
echo   Acesso: %URL%
echo.

cd /d "%REPO%" 2>nul
if errorlevel 1 (
  echo [ERRO] Pasta nao encontrada: %REPO%
  goto :falhou
)

echo [1/5] Conferindo o Docker...
docker info >nul 2>&1
if errorlevel 1 (
  echo [ERRO] O Docker Desktop nao esta rodando.
  echo        Abra o Docker Desktop, espere ele iniciar e tente de novo.
  goto :falhou
)

echo [2/5] Conferindo a pasta...
for /f "delims=" %%b in ('git branch --show-current') do set "BRANCH=%%b"
if /i not "!BRANCH!"=="main" (
  echo [ERRO] Esta pasta esta na branch "!BRANCH!", e nao na main.
  echo        Producao roda a partir da pasta da main. Confira se o
  echo        atalho aponta para o script que esta nessa pasta.
  goto :falhou
)

echo [3/5] Buscando a ultima versao da main...
git fetch origin main
if errorlevel 1 (
  echo [ERRO] Falha ao buscar do GitHub. Sem internet ou sem credencial?
  goto :falhou
)
git merge --ff-only origin/main
if errorlevel 1 (
  echo [ERRO] A main local divergiu da remota.
  echo        Resolva isso antes de subir producao.
  goto :falhou
)
echo       Versao que vai subir:
git log -1 --oneline

echo [4/5] Construindo as imagens e subindo os containers...
docker compose -p %PROJETO% -f docker-compose.prod.yml up -d --build
if errorlevel 1 (
  echo [ERRO] Falha ao subir os containers. Veja as mensagens acima.
  goto :falhou
)

echo [5/5] Esperando o sistema responder...
set /a tentativas=0
REM  O ping abaixo e a forma de esperar N segundos: o timeout.exe
REM  exige um console e aborta quando o stdin esta redirecionado.
:aguarda
set /a tentativas+=1
"%SystemRoot%\System32\curl.exe" -s -o nul %URL% >nul 2>&1
if not errorlevel 1 goto :pronto
if !tentativas! geq 90 (
  echo [AVISO] O sistema nao respondeu em 3 minutos.
  echo         Veja os logs: docker compose -p %PROJETO% logs
  goto :falhou
)
"%SystemRoot%\System32\ping.exe" -n 3 127.0.0.1 >nul
goto :aguarda

:pronto
echo.
echo ===============================================
echo   PRONTO - sistema no ar em %URL%
echo ===============================================
echo.
docker compose -p %PROJETO% ps
echo.
echo   Os containers ficam rodando em segundo plano e voltam
echo   sozinhos quando o computador reiniciar.
echo.
echo   Para parar:  docker compose -p %PROJETO% down
echo.
start "" %URL%
"%SystemRoot%\System32\ping.exe" -n 9 127.0.0.1 >nul
exit /b 0

:falhou
echo.
echo ===============================================
echo   FALHOU - o sistema no ar nao foi alterado
echo ===============================================
echo.
pause
exit /b 1

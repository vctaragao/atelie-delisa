@echo off
setlocal enabledelayedexpansion

REM =================================================================
REM  ATELIE DELISA - DESENVOLVIMENTO
REM
REM  Sobe os containers de desenvolvimento a partir da worktree da
REM  branch development, com hot reload: editar um arquivo no
REM  Windows recompila / recarrega na hora.
REM
REM  Este script NAO faz pull. O codigo que roda e o que esta no
REM  seu disco agora, inclusive alteracoes ainda nao commitadas.
REM
REM  Dados: volume "atelie-delisa-dev_atelie-data", separado da
REM  producao. Mexer aqui nao afeta os dados reais do atelie.
REM
REM  Sem acentos de proposito: o cmd.exe erra ao interpretar
REM  caracteres multibyte em arquivos .bat.
REM =================================================================

set "REPO=C:\Users\Maria\atelie-delisa-dev"
set "PROJETO=atelie-delisa-dev"
set "URL=http://localhost:5173"
set "API=http://localhost:8080/api/health"

title Atelie Delisa - DESENVOLVIMENTO

echo.
echo ===============================================
echo   ATELIE DELISA - DESENVOLVIMENTO
echo ===============================================
echo   Pasta : %REPO%
echo   Site  : %URL%
echo   API   : %API%
echo.

cd /d "%REPO%" 2>nul
if errorlevel 1 (
  echo [ERRO] Pasta nao encontrada: %REPO%
  goto :falhou
)

echo [1/3] Conferindo o Docker...
docker info >nul 2>&1
if errorlevel 1 (
  echo [ERRO] O Docker Desktop nao esta rodando.
  echo        Abra o Docker Desktop, espere ele iniciar e tente de novo.
  goto :falhou
)

for /f "delims=" %%b in ('git branch --show-current') do set "BRANCH=%%b"
echo       Branch: !BRANCH!
if /i not "!BRANCH!"=="development" (
  echo.
  echo [AVISO] Esta pasta nao esta na development, e sim na "!BRANCH!".
  echo         Se isso nao foi proposital, feche esta janela agora.
  echo.
"%SystemRoot%\System32\ping.exe" -n 6 127.0.0.1 >nul
)

echo [2/3] Construindo as imagens e subindo os containers...
docker compose -p %PROJETO% up -d --build
if errorlevel 1 (
  echo [ERRO] Falha ao subir os containers. Veja as mensagens acima.
  goto :falhou
)

echo [3/3] Esperando o Vite responder...
set /a tentativas=0
REM  O ping abaixo e a forma de esperar N segundos: o timeout.exe
REM  exige um console e aborta quando o stdin esta redirecionado.
:aguarda
set /a tentativas+=1
"%SystemRoot%\System32\curl.exe" -s -o nul %URL% >nul 2>&1
if not errorlevel 1 goto :pronto
if !tentativas! geq 120 (
  echo [AVISO] O site nao respondeu em 4 minutos.
  echo         Veja os logs: docker compose -p %PROJETO% logs
  goto :falhou
)
"%SystemRoot%\System32\ping.exe" -n 3 127.0.0.1 >nul
goto :aguarda

:pronto
echo.
echo ===============================================
echo   PRONTO - ambiente de dev no ar em %URL%
echo ===============================================
echo.
start "" %URL%
echo   Abaixo vem o log ao vivo dos containers.
echo.
echo   Ctrl+C  para de mostrar o log, mas os containers CONTINUAM
echo           rodando e o hot reload segue funcionando.
echo   Para parar de verdade:  docker compose -p %PROJETO% down
echo.
"%SystemRoot%\System32\ping.exe" -n 4 127.0.0.1 >nul
docker compose -p %PROJETO% logs -f
exit /b 0

:falhou
echo.
echo ===============================================
echo   FALHOU
echo ===============================================
echo.
pause
exit /b 1

@echo off
setlocal enabledelayedexpansion

REM =================================================================
REM  ATELIE DELISA - CHAT
REM
REM  Abre o Claude Code ja dentro da pasta de desenvolvimento, para
REM  nao precisar abrir terminal nem navegar ate ela.
REM
REM  A pasta importa: e nela que a branch development esta em
REM  checkout e onde os containers de dev montam o codigo. Aberto em
REM  outro lugar, o Claude editaria os arquivos errados.
REM
REM  Sem acentos de proposito: o cmd.exe erra ao interpretar
REM  caracteres multibyte em arquivos .bat.
REM =================================================================

set "REPO=C:\Users\Maria\atelie-delisa-dev"
set "CLAUDE=C:\Users\Maria\.local\bin\claude.exe"

title Atelie Delisa - Chat

cd /d "%REPO%" 2>nul
if errorlevel 1 (
  echo [ERRO] Pasta nao encontrada: %REPO%
  goto :falhou
)

REM Se a instalacao mudar de lugar, ainda tenta pelo PATH.
if not exist "!CLAUDE!" (
  where claude >nul 2>&1
  if errorlevel 1 (
    echo [ERRO] O Claude Code nao foi encontrado.
    echo        Procurei em !CLAUDE! e tambem no PATH.
    goto :falhou
  )
  set "CLAUDE=claude"
)

echo.
echo ===============================================
echo   ATELIE DELISA - CHAT
echo ===============================================
echo   Pasta: %REPO%
echo.

REM Aviso que nao bloqueia: da para conversar com o ambiente parado,
REM mas as alteracoes nao aparecem no navegador enquanto ele nao subir.
set "ATIVOS="
for /f "delims=" %%c in ('docker ps --filter "name=atelie-delisa-dev-" -q 2^>nul') do set "ATIVOS=1"
if not defined ATIVOS (
  echo   [aviso] O ambiente de desenvolvimento parece estar parado.
  echo           Abra tambem o atalho "atelie-delisa-dev" para ver as
  echo           alteracoes acontecendo no navegador.
  echo.
)

REM  O prompt inicial mora em scripts/boas-vindas.md, e nao aqui: a
REM  saudacao tem acentos, e este arquivo precisa seguir em ASCII puro.
"!CLAUDE!" "Leia o arquivo scripts/boas-vindas.md e siga as instrucoes que ele traz."
set "RC=!errorlevel!"
if not "!RC!"=="0" (
  echo.
  echo O Claude Code encerrou com erro, codigo !RC!.
  pause
)
exit /b !RC!

:falhou
echo.
pause
exit /b 1

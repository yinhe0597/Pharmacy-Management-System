@echo off
chcp 65001 >nul
title 药房管理系统 - 一键部署
cd /d "%~dp0"

echo ============================================
echo   药房管理系统 · Docker 一键部署
echo ============================================

:: 首次运行自动生成 .env（含随机 JWT 密钥）
if not exist .env (
  echo [1/3] 首次运行：生成 .env（随机 JWT 密钥）...
  powershell -NoProfile -Command ^
    "$c=(48..57)+(65..90)+(97..122)|Get-Random -Count 48|ForEach-Object{[char]$_};-join($c)" > "%TEMP%\yf_secret.txt"
  powershell -NoProfile -Command ^
    "$s=(Get-Content '%TEMP%\yf_secret.txt' -Raw).Trim(); 'JWT_SECRET='+$s | Set-Content -Path '.env' -Encoding ASCII; Add-Content -Path '.env' -Value 'DB_PASSWORD=yaofang123'; Add-Content -Path '.env' -Value 'HTTP_PORT=80'"
  del "%TEMP%\yf_secret.txt" >nul 2>&1
) else (
  echo [1/3] .env 已存在，跳过生成
)

echo [2/3] 构建并启动容器（首次构建约需几分钟）...
docker compose -f docker-compose.prod.yml up -d --build
if errorlevel 1 (
  echo 启动失败！请确认 Docker Desktop 已启动。
  pause
  exit /b 1
)

echo [3/3] 等待服务就绪...
powershell -NoProfile -Command "$d=(Get-Date).AddSeconds(90);while((Get-Date) -lt $d){try{$r=Invoke-WebRequest -Uri ('http://localhost:'+$env:HTTP_PORT+'/healthz') -UseBasicParsing -TimeoutSec 2;if($r.StatusCode -eq 200){Write-Host 'ready';exit 0}}catch{};Start-Sleep 3}exit 1"
if errorlevel 1 (
  echo 服务仍在启动中，请稍后手动访问。
) else (
  echo.
  echo ============================================
  echo   部署完成！
  echo   本机访问:   http://localhost
  echo   内网访问:   http://本机IP  （其它设备浏览器直接打开）
  echo   默认账号:   admin / admin123 （登录后请立即改密码！）
  echo   数据库:     Docker 卷 pgdata_prod 持久化
  echo ============================================
)
pause

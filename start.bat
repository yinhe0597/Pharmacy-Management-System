@echo off
chcp 65001 >nul
title 药房管理系统 - 一键部署
cd /d "%~dp0"

echo ============================================
echo   药房管理系统 · Docker 一键部署
echo ============================================

:: 首次运行自动生成 .env：CSPRNG 随机 JWT 密钥（48 字节 → base64 64 字符，满足后端 ≥32 字符校验）
:: + 随机数据库口令（24 字节 → 48 位十六进制）。使用 RandomNumberGenerator（CSPRNG），勿改回 Get-Random。
if not exist .env (
  echo [1/4] 首次运行：生成 .env（随机 JWT 密钥 + 随机数据库口令）...
  powershell -NoProfile -ExecutionPolicy Bypass -Command ^
    "$ErrorActionPreference='Stop';" ^
    "$rng=[System.Security.Cryptography.RandomNumberGenerator]::Create();" ^
    "$jb=New-Object byte[] 48; $rng.GetBytes($jb); $jwt=[Convert]::ToBase64String($jb);" ^
    "$db=New-Object byte[] 24; $rng.GetBytes($db); $dbp=[BitConverter]::ToString($db).Replace('-','');" ^
    "if ($jwt.Length -lt 32) { throw 'JWT 密钥生成失败' };" ^
    "$lines=@('# 由 start.bat 自动生成','# 含密钥与口令，请勿提交版本库、勿外传。','JWT_SECRET=' + $jwt,'DB_PASSWORD=' + $dbp,'HTTP_PORT=80');" ^
    "Set-Content -Path '.env' -Value $lines -Encoding ASCII"
  if errorlevel 1 (
    echo 生成 .env 失败，请检查 PowerShell 可用性。
    pause
    exit /b 1
  )
  echo       已生成 .env：数据库口令为随机值，可在 .env 中查看。
) else (
  echo [1/4] .env 已存在，跳过生成
)

:: 读取 .env 中的 HTTP_PORT 供就绪探测使用
set "HTTP_PORT=80"
for /f "usebackq tokens=2 delims==" %%a in (`findstr /b "HTTP_PORT=" .env`) do set "HTTP_PORT=%%a"
set "HTTP_PORT=%HTTP_PORT:'=%"

echo [2/4] 校验部署配置...
findstr /b "JWT_SECRET=" .env >nul 2>&1
if errorlevel 1 (
  echo 错误：.env 缺少 JWT_SECRET。请删除 .env 重新运行本脚本。
  pause
  exit /b 1
)
findstr /b "DB_PASSWORD=" .env >nul 2>&1
if errorlevel 1 (
  echo 错误：.env 缺少 DB_PASSWORD。请删除 .env 重新运行本脚本。
  pause
  exit /b 1
)

echo [3/4] 构建并启动容器（首次构建约需几分钟）...
docker compose -f docker-compose.prod.yml up -d --build
if errorlevel 1 (
  echo 启动失败！请确认 Docker Desktop 已启动。
  pause
  exit /b 1
)

echo [4/4] 等待服务就绪...
powershell -NoProfile -Command "$d=(Get-Date).AddSeconds(90);while((Get-Date) -lt $d){try{$r=Invoke-WebRequest -Uri ('http://localhost:'+$env:HTTP_PORT+'/healthz') -UseBasicParsing -TimeoutSec 2;if($r.StatusCode -eq 200){Write-Host 'ready';exit 0}}catch{};Start-Sleep 3}exit 1"
if errorlevel 1 (
  echo 服务仍在启动中，请稍后手动访问。
  echo 如持续失败请查看日志：docker compose -f docker-compose.prod.yml logs -f backend
) else (
  echo.
  echo ============================================
  echo   部署完成！
  echo   本机访问:   http://localhost:%HTTP_PORT%
  echo   内网访问:   http://本机IP:%HTTP_PORT%  （其它设备浏览器直接打开）
  echo   默认账号:   admin / admin123 （种子账号，登录后请立即改密码！）
  echo   数据库口令: 随机生成，见 .env（DB_PASSWORD）
  echo   数据库:     Docker 卷 pgdata_prod 持久化
  echo   ⚠ release 模式下若仍使用默认口令 admin123，服务将拒绝启动。
  echo ============================================
)
pause

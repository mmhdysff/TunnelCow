$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "================================" -ForegroundColor Cyan
Write-Host "   TUNNELCOW BUILD SCRIPT" -ForegroundColor Cyan  
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""


$dataPath = "data"
if (-not (Test-Path $dataPath)) { New-Item -ItemType Directory -Force -Path $dataPath | Out-Null }
$buildFile = "$dataPath/.build_info.json"

if (-not (Test-Path $buildFile)) {
    @{ total = 0; patch = 0 } | ConvertTo-Json | Out-File $buildFile -Encoding UTF8
}
$buildInfo = Get-Content $buildFile -Raw | ConvertFrom-Json
$buildInfo.total++
$version = "0.$($buildInfo.total).$($buildInfo.patch)"

Write-Host "[1/6] Build Configuration" -ForegroundColor Magenta
Write-Host "      Version: v$version" -ForegroundColor White
Write-Host "      Build: #$($buildInfo.total)" -ForegroundColor Gray
Write-Host ""


Write-Host "[2/6] Updating App.jsx Version..." -ForegroundColor Magenta
$appPath = "web/src/App.jsx"
$content = Get-Content $appPath -Raw
$newContent = $content -replace "v\d+\.\d+\.\d+", "v$version"
$newContent | Set-Content $appPath -Encoding UTF8
Write-Host "      [OK] Updated to v$version" -ForegroundColor Green
Write-Host ""


Write-Host "[3/6] Building Frontend..." -ForegroundColor Magenta
Push-Location web
$buildStart = Get-Date
npm run build
$buildTime = ((Get-Date) - $buildStart).TotalSeconds
Pop-Location
if ($LASTEXITCODE -ne 0) { 
    Write-Host "      [ERROR] Frontend build failed!" -ForegroundColor Red
    exit 1 
}
Write-Host "      [OK] Build completed in $([math]::Round($buildTime, 2))s" -ForegroundColor Green
Write-Host ""


Write-Host "[4/6] Embedding Static Assets..." -ForegroundColor Magenta
$dest = "cmd/tunnelcow-client/web_dist"
if (Test-Path $dest) { Remove-Item $dest -Recurse -Force }
New-Item -ItemType Directory -Force -Path $dest | Out-Null
Copy-Item "web/dist/*" $dest -Recurse -Force
$fileCount = (Get-ChildItem $dest -Recurse -File).Count
Write-Host "      [OK] Embedded $fileCount files" -ForegroundColor Green
Write-Host ""


Write-Host "[5/6] Compiling Go Binaries..." -ForegroundColor Magenta
$compileStart = Get-Date


Write-Host "      Building Client..." -ForegroundColor Gray
go build -ldflags "-X main.Version=$version -s -w" -o tunnelcow-client.exe ./cmd/tunnelcow-client
if ($LASTEXITCODE -ne 0) { 
    Write-Host "      [ERROR] Client build failed!" -ForegroundColor Red
    exit 1 
}
$clientSize = [math]::Round((Get-Item tunnelcow-client.exe).Length / 1MB, 2)
Write-Host "      [OK] Client: $clientSize MB" -ForegroundColor Green


Write-Host "      Building Server..." -ForegroundColor Gray
go build -ldflags "-X main.Version=$version -s -w" -o tunnelcow-server.exe ./cmd/tunnelcow-server
if ($LASTEXITCODE -ne 0) { 
    Write-Host "      [ERROR] Server build failed!" -ForegroundColor Red
    exit 1 
}
$serverSize = [math]::Round((Get-Item tunnelcow-server.exe).Length / 1MB, 2)
Write-Host "      [OK] Server: $serverSize MB" -ForegroundColor Green


Write-Host "      Building Linux Binaries..." -ForegroundColor Gray
$env:GOOS = "linux"
$env:GOARCH = "amd64"


go build -ldflags "-X main.Version=$version -s -w" -o tunnelcow-client-linux ./cmd/tunnelcow-client
if ($LASTEXITCODE -ne 0) { 
    Write-Host "      [ERROR] Linux Client build failed!" -ForegroundColor Red
    
    $env:GOOS = "windows"; $env:GOARCH = "amd64"
    exit 1 
}


go build -ldflags "-X main.Version=$version -s -w" -o tunnelcow-server-linux ./cmd/tunnelcow-server
if ($LASTEXITCODE -ne 0) { 
    Write-Host "      [ERROR] Linux Server build failed!" -ForegroundColor Red
    $env:GOOS = "windows"; $env:GOARCH = "amd64"
    exit 1 
}


$env:GOOS = "windows"
$env:GOARCH = "amd64"
Write-Host "      [OK] Linux Binaries Built" -ForegroundColor Green

$compileTime = ((Get-Date) - $compileStart).TotalSeconds
Write-Host "      Time: $([math]::Round($compileTime, 2))s" -ForegroundColor Gray
Write-Host ""


Write-Host "[6/6] Saving Build Metadata..." -ForegroundColor Magenta
$buildInfo | ConvertTo-Json | Out-File $buildFile -Encoding UTF8
Write-Host "      [OK] Saved build #$($buildInfo.total)" -ForegroundColor Green
Write-Host ""


Write-Host "================================" -ForegroundColor Green
Write-Host "  BUILD COMPLETE" -ForegroundColor Green
Write-Host "================================" -ForegroundColor Green
Write-Host ""
Write-Host "  Version: v$version" -ForegroundColor White
Write-Host "  Build:   #$($buildInfo.total)" -ForegroundColor Gray
Write-Host ""
Write-Host "  Artifacts:" -ForegroundColor White
Write-Host "    - tunnelcow-client.exe ($clientSize MB)" -ForegroundColor Gray
Write-Host "    - tunnelcow-server.exe ($serverSize MB)" -ForegroundColor Gray
Write-Host "    - tunnelcow-client-linux" -ForegroundColor Gray
Write-Host "    - tunnelcow-server-linux" -ForegroundColor Gray
Write-Host ""
Write-Host "  Ready to deploy!" -ForegroundColor Cyan
Write-Host ""

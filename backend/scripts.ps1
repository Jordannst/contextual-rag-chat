# PowerShell scripts untuk backend Go

param(
    [Parameter(Mandatory=$true)]
    [ValidateSet("dev", "build", "run", "migrate", "create-db")]
    [string]$Command
)

$ErrorActionPreference = "Stop"

switch ($Command) {
    "dev" {
        Write-Host "Starting development server..." -ForegroundColor Green
        
        # Check if port 5000 is in use
        $portInUse = Get-NetTCPConnection -LocalPort 5000 -ErrorAction SilentlyContinue
        if ($portInUse) {
            Write-Host "Warning: Port 5000 is already in use!" -ForegroundColor Yellow
            $processId = $portInUse.OwningProcess
            $process = Get-Process -Id $processId -ErrorAction SilentlyContinue
            if ($process) {
                Write-Host "Found process: $($process.ProcessName) (PID: $processId)" -ForegroundColor Yellow
                $response = Read-Host "Kill this process and continue? (y/n)"
                if ($response -eq "y" -or $response -eq "Y") {
                    Stop-Process -Id $processId -Force
                    Write-Host "Process stopped. Starting server..." -ForegroundColor Green
                    Start-Sleep -Seconds 1
                } else {
                    Write-Host "Aborted. Please stop the process manually." -ForegroundColor Red
                    exit 1
                }
            }
        }
        
        go run main.go
    }
    "build" {
        Write-Host "Building backend..." -ForegroundColor Green
        if (-not (Test-Path "bin")) {
            New-Item -ItemType Directory -Path "bin" | Out-Null
        }
        go build -o bin/backend.exe main.go
        Write-Host "Build completed! Output: bin/backend.exe" -ForegroundColor Green
    }
    "run" {
        Write-Host "Running backend..." -ForegroundColor Green
        if (-not (Test-Path "bin/backend.exe")) {
            Write-Host "Backend not built. Run 'build' first." -ForegroundColor Yellow
            exit 1
        }
        ./bin/backend.exe
    }
    "migrate" {
        Write-Host "Running database migration..." -ForegroundColor Green
        go run cmd/migrate/main.go
    }
    "create-db" {
        Write-Host "Creating database..." -ForegroundColor Green
        go run cmd/create-db/main.go
    }
}


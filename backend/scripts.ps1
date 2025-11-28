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


# GCC Auto-Installation Test Execution Script
# This script runs all GCC-related tests on Windows

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "GCC Auto-Installation Test Suite" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if running on Windows
if ($PSVersionTable.Platform -and $PSVersionTable.Platform -ne "Win32NT") {
    Write-Host "ERROR: This script must be run on Windows" -ForegroundColor Red
    exit 1
}

# Check if Go is installed
try {
    $goVersion = go version
    Write-Host "Go version: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Pre-Test Environment Check" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check GCC availability
Write-Host "Checking GCC availability..." -ForegroundColor Yellow
try {
    $gccVersion = gcc --version 2>&1
    Write-Host "GCC is installed:" -ForegroundColor Green
    Write-Host $gccVersion[0] -ForegroundColor Green
    $gccInstalled = $true
} catch {
    Write-Host "GCC is NOT installed" -ForegroundColor Yellow
    $gccInstalled = $false
}

Write-Host ""

# Check winget availability
Write-Host "Checking winget availability..." -ForegroundColor Yellow
try {
    $wingetVersion = winget --version 2>&1
    Write-Host "winget is installed: $wingetVersion" -ForegroundColor Green
    $wingetInstalled = $true
} catch {
    Write-Host "winget is NOT installed" -ForegroundColor Yellow
    $wingetInstalled = $false
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Execution" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Run all tests
Write-Host "Running all GCC tests..." -ForegroundColor Yellow
Write-Host ""

$testResults = @()

# Test 1: Check GCC in PATH
Write-Host "Test 1: TestCheckGCCInPath" -ForegroundColor Cyan
$result1 = go test -v ./internal/updater -run TestCheckGCCInPath 2>&1
Write-Host $result1
$testResults += @{Name="TestCheckGCCInPath"; Result=$result1}
Write-Host ""

# Test 2: Check GCC in common locations
Write-Host "Test 2: TestCheckGCCInCommonLocations" -ForegroundColor Cyan
$result2 = go test -v ./internal/updater -run TestCheckGCCInCommonLocations 2>&1
Write-Host $result2
$testResults += @{Name="TestCheckGCCInCommonLocations"; Result=$result2}
Write-Host ""

# Test 3: Verify winget available
Write-Host "Test 3: TestVerifyWingetAvailable" -ForegroundColor Cyan
$result3 = go test -v ./internal/updater -run TestVerifyWingetAvailable 2>&1
Write-Host $result3
$testResults += @{Name="TestVerifyWingetAvailable"; Result=$result3}
Write-Host ""

# Test 4: Detect GCC install path
Write-Host "Test 4: TestDetectGCCInstallPath" -ForegroundColor Cyan
$result4 = go test -v ./internal/updater -run TestDetectGCCInstallPath 2>&1
Write-Host $result4
$testResults += @{Name="TestDetectGCCInstallPath"; Result=$result4}
Write-Host ""

# Test 5: Update PATH environment
Write-Host "Test 5: TestUpdatePATHEnvironment" -ForegroundColor Cyan
$result5 = go test -v ./internal/updater -run TestUpdatePATHEnvironment 2>&1
Write-Host $result5
$testResults += @{Name="TestUpdatePATHEnvironment"; Result=$result5}
Write-Host ""

# Test 6: Ensure GCC available (integration test)
Write-Host "Test 6: TestEnsureGCCAvailable (Integration Test)" -ForegroundColor Cyan
Write-Host "WARNING: This test may install GCC if not present and winget is available" -ForegroundColor Yellow
$continue = Read-Host "Continue with integration test? (y/n)"
if ($continue -eq "y") {
    $result6 = go test -v ./internal/updater -run TestEnsureGCCAvailable 2>&1
    Write-Host $result6
    $testResults += @{Name="TestEnsureGCCAvailable"; Result=$result6}
} else {
    Write-Host "Skipping integration test" -ForegroundColor Yellow
    $testResults += @{Name="TestEnsureGCCAvailable"; Result="SKIPPED"}
}
Write-Host ""

# Test 7: GCC installation scenarios
Write-Host "Test 7: TestGCCInstallationScenarios" -ForegroundColor Cyan
$result7 = go test -v ./internal/updater -run TestGCCInstallationScenarios 2>&1
Write-Host $result7
$testResults += @{Name="TestGCCInstallationScenarios"; Result=$result7}
Write-Host ""

# Test 8: GCC version detection
Write-Host "Test 8: TestGCCVersionDetection" -ForegroundColor Cyan
$result8 = go test -v ./internal/updater -run TestGCCVersionDetection 2>&1
Write-Host $result8
$testResults += @{Name="TestGCCVersionDetection"; Result=$result8}
Write-Host ""

# Test 9: PATH update persistence
Write-Host "Test 9: TestPATHUpdatePersistence" -ForegroundColor Cyan
$result9 = go test -v ./internal/updater -run TestPATHUpdatePersistence 2>&1
Write-Host $result9
$testResults += @{Name="TestPATHUpdatePersistence"; Result=$result9}
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$passed = 0
$failed = 0
$skipped = 0

foreach ($test in $testResults) {
    $name = $test.Name
    $result = $test.Result
    
    if ($result -match "PASS" -or $result -match "ok") {
        Write-Host "[PASS] $name" -ForegroundColor Green
        $passed++
    } elseif ($result -eq "SKIPPED" -or $result -match "SKIP") {
        Write-Host "[SKIP] $name" -ForegroundColor Yellow
        $skipped++
    } else {
        Write-Host "[FAIL] $name" -ForegroundColor Red
        $failed++
    }
}

Write-Host ""
Write-Host "Total: $($testResults.Count) tests" -ForegroundColor Cyan
Write-Host "Passed: $passed" -ForegroundColor Green
Write-Host "Failed: $failed" -ForegroundColor $(if ($failed -gt 0) { "Red" } else { "Green" })
Write-Host "Skipped: $skipped" -ForegroundColor Yellow
Write-Host ""

if ($failed -gt 0) {
    Write-Host "Some tests failed. Review the output above for details." -ForegroundColor Red
    exit 1
} else {
    Write-Host "All tests passed or skipped!" -ForegroundColor Green
    exit 0
}

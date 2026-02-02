# SentinelGo Updater Service

A separate, independent service responsible for managing updates to the SentinelGo main agent.

## Overview

The SentinelGo Updater Service runs as an independent system service that:
- Continuously monitors for new versions of the SentinelGo main agent
- Manages the complete update lifecycle (stop, uninstall, download, compile, install, start)
- Handles rollback on update failures
- Maintains database persistence across updates
- Operates independently from the main agent to ensure reliable updates

### Why a Separate Updater Service?

The previous architecture had the main agent spawn an updater process on-demand. This approach had critical issues:
- **Process Management:** On Linux/macOS, systemd/launchd would kill the updater when the parent agent exited
- **Update Reliability:** Updates could fail if the agent crashed during the update process
- **Service Dependencies:** The agent couldn't reliably manage its own lifecycle

The new architecture solves these problems by:
- Running the updater as a completely independent service with its own lifecycle
- Allowing the updater to fully control the main agent's lifecycle (stop, uninstall, install, start)
- Ensuring updates can complete even if the main agent encounters issues
- Providing a clean separation of concerns between monitoring (agent) and maintenance (updater)

## Installation

### Using go install (Recommended)

```bash
# Install the updater binary
go install github.com/BrainStation-23/SentinelGo-Updater/cmd/sentinel-updater@latest

# Install as a system service
sentinel-updater install

# Start the service
sentinel-updater start
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/BrainStation-23/SentinelGo-Updater.git
cd SentinelGo-Updater

# Build with version information
go build -ldflags "-X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$(git rev-parse --short HEAD)" -o sentinel-updater ./cmd/sentinel-updater

# Install as a system service
sudo ./sentinel-updater install

# Start the service
sudo ./sentinel-updater start
```

## Usage

### Service Management Commands

```bash
# Install the updater as a system service
sentinel-updater install

# Uninstall the updater service
sentinel-updater uninstall

# Start the updater service
sentinel-updater start

# Stop the updater service
sentinel-updater stop

# Restart the updater service
sentinel-updater restart

# Show version information
sentinel-updater --version
```

## Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        System Boot                           │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ├──────────────────┬──────────────────────────┐
                 │                  │                          │
                 ▼                  ▼                          ▼
        ┌────────────────┐  ┌──────────────┐      ┌──────────────────┐
        │ Updater Service│  │  Main Agent  │      │ Standard Data    │
        │   (sentinel-   │  │  (sentinel)  │      │   Directory      │
        │    updater)    │  │              │      │                  │
        │                │  │              │      │ - sentinel.db    │
        │ - Checks for   │  │ - Monitoring │      │ - updater.log    │
        │   updates      │  │ - Reporting  │      │ - agent.log      │
        │ - Manages      │  │ - Tasks      │      │                  │
        │   Main Agent   │  │              │      │                  │
        │   lifecycle    │  │              │      │                  │
        └────────┬───────┘  └──────┬───────┘      └──────────────────┘
                 │                  │                       ▲
                 │                  │                       │
                 │                  └───────────────────────┘
                 │                    Reads/Writes DB
                 │
                 │ Controls (stop/uninstall/install/start)
                 │
                 ▼
        ┌────────────────────────────┐
        │   Service Manager          │
        │  (systemd/launchd/SCM)     │
        └────────────────────────────┘
```

### Update Cycle Flow

The updater service follows this process for each update:

1. **Version Check:** Query Go module system for latest version every 30 seconds
2. **Update Detection:** Compare installed version with latest available version
3. **Stop Agent:** Use platform-specific service manager to stop the main agent
4. **Uninstall Service:** Remove the main agent service registration
5. **Cleanup:** Delete old binary and artifacts (preserving database and logs)
6. **Download & Compile:** Use `go install` to build the new version with CGO enabled
7. **Install Binary:** Copy compiled binary to installation directory with correct permissions
8. **Reinstall Service:** Register the new version with the service manager
9. **Start Agent:** Start the updated main agent service
10. **Verify:** Confirm the agent is running with the new version

If any step fails, the updater attempts to rollback to the previous version.

### Service Independence

The updater service:
1. Runs independently from the main agent
2. Checks for updates every 30 seconds (configurable)
3. Uses `go install` to download and compile new versions
4. Manages the main agent service lifecycle through platform-specific service managers:
   - Linux: systemd
   - macOS: launchd
   - Windows: Windows Service Manager

### Platform-Specific Service Management

**Linux (systemd):**
- Service file: `/etc/systemd/system/sentinelgo-updater.service`
- Commands: `systemctl start/stop/status/enable/disable`
- Runs as root with automatic restart on failure

**macOS (launchd):**
- Plist file: `/Library/LaunchDaemons/com.sentinelgo.updater.plist`
- Commands: `launchctl load/unload/start/stop/list`
- Runs as root with KeepAlive enabled

**Windows (Service Control Manager):**
- Service name: `sentinelgo-updater`
- Commands: `sc start/stop/query`, `net start/stop`
- Runs as LocalSystem with automatic startup

## File Locations

### Linux/macOS
- Data Directory: `/var/lib/sentinelgo/`
- Database: `/var/lib/sentinelgo/sentinel.db`
- Updater Log: `/var/lib/sentinelgo/updater.log`
- Binary: `/usr/local/bin/sentinel-updater`

### Windows
- Data Directory: `C:\ProgramData\SentinelGo\`
- Database: `C:\ProgramData\SentinelGo\sentinel.db`
- Updater Log: `C:\ProgramData\SentinelGo\updater.log`
- Binary: `C:\Program Files\SentinelGo\sentinel-updater.exe`

## Requirements

- Go 1.21 or later (for compilation)
- CGO enabled (for SQLite support in main agent)
- Elevated privileges (root/Administrator) for service management
- On Windows: GCC toolchain for CGO compilation

### Platform-Specific Requirements

**Linux:**
- systemd (for service management)
- build-essential or equivalent (gcc, make, etc.)
- Root access via sudo

**macOS:**
- launchd (built-in)
- Xcode Command Line Tools (for gcc)
- Root access via sudo

**Windows:**
- Windows Service Manager (built-in)
- GCC toolchain (TDM-GCC, MinGW-w64, or similar)
- Administrator privileges

## Configuration

The updater service can be configured through environment variables or by modifying the service configuration.

### Environment Variables

- `CHECK_INTERVAL`: Update check interval (default: 30s, recommended production: 5m-15m)
- `MAIN_AGENT_MODULE`: Go module path for main agent (default: github.com/BrainStation-23/SentinelGo)
- `MAIN_AGENT_SERVICE_NAME`: Service name for main agent (default: sentinelgo)
- `LOG_LEVEL`: Logging verbosity (debug, info, warn, error)
- `MAX_LOG_SIZE`: Maximum log file size before rotation (default: 10MB)
- `MAX_LOG_FILES`: Number of rotated log files to keep (default: 5)

### Setting Environment Variables

**Linux (systemd):**

Edit the service file `/etc/systemd/system/sentinelgo-updater.service`:

```ini
[Service]
Environment="CHECK_INTERVAL=5m"
Environment="LOG_LEVEL=info"
```

Then reload and restart:
```bash
sudo systemctl daemon-reload
sudo systemctl restart sentinelgo-updater
```

**macOS (launchd):**

Edit the plist file `/Library/LaunchDaemons/com.sentinelgo.updater.plist`:

```xml
<key>EnvironmentVariables</key>
<dict>
    <key>CHECK_INTERVAL</key>
    <string>5m</string>
    <key>LOG_LEVEL</key>
    <string>info</string>
</dict>
```

Then reload:
```bash
sudo launchctl unload /Library/LaunchDaemons/com.sentinelgo.updater.plist
sudo launchctl load /Library/LaunchDaemons/com.sentinelgo.updater.plist
```

**Windows:**

Set environment variables in the registry or use `sc config`:

```powershell
# Using registry (requires restart)
[Environment]::SetEnvironmentVariable("CHECK_INTERVAL", "5m", "Machine")

# Then restart the service
Restart-Service sentinelgo-updater
```

### Update Check Interval

The default check interval is 30 seconds for development. For production deployments, consider:

- **Low-frequency updates:** 15 minutes to 1 hour
- **Standard updates:** 5-15 minutes
- **High-priority updates:** 1-5 minutes
- **Development/testing:** 30 seconds to 1 minute

Example:
```bash
# Set to 10 minutes
CHECK_INTERVAL=10m
```

## Development

### Running Tests

```bash
go test ./...
```

### Building for Multiple Platforms

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o dist/sentinel-updater-linux-amd64 ./cmd/sentinel-updater

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o dist/sentinel-updater-linux-arm64 ./cmd/sentinel-updater

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o dist/sentinel-updater-darwin-amd64 ./cmd/sentinel-updater

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o dist/sentinel-updater-darwin-arm64 ./cmd/sentinel-updater

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o dist/sentinel-updater-windows-amd64.exe ./cmd/sentinel-updater
```

## Troubleshooting

### Check Service Status

**Linux:**
```bash
sudo systemctl status sentinelgo-updater

# View recent logs
sudo journalctl -u sentinelgo-updater -n 50 -f
```

**macOS:**
```bash
sudo launchctl list | grep sentinelgo-updater

# View logs
sudo tail -f /var/lib/sentinelgo/updater.log
```

**Windows:**
```powershell
sc query sentinelgo-updater

# Or using Get-Service
Get-Service sentinelgo-updater | Format-List *
```

### View Logs

**Linux/macOS:**
```bash
# Tail logs in real-time
sudo tail -f /var/lib/sentinelgo/updater.log

# View last 100 lines
sudo tail -n 100 /var/lib/sentinelgo/updater.log

# Search for errors
sudo grep -i error /var/lib/sentinelgo/updater.log
```

**Windows:**
```powershell
# View logs
Get-Content C:\ProgramData\SentinelGo\updater.log -Tail 100

# Follow logs in real-time
Get-Content C:\ProgramData\SentinelGo\updater.log -Wait

# Search for errors
Select-String -Path C:\ProgramData\SentinelGo\updater.log -Pattern "error" -CaseSensitive:$false
```

### Common Issues and Solutions

#### 1. Service Fails to Start

**Symptoms:**
- Service status shows "failed" or "inactive"
- Error: "Permission denied" or "Access denied"

**Solutions:**
```bash
# Check if running with elevated privileges
# Linux/macOS
sudo systemctl status sentinelgo-updater

# Windows (run as Administrator)
sc query sentinelgo-updater

# Verify binary exists and is executable
# Linux/macOS
ls -l $(which sentinel-updater)
sudo chmod +x $(which sentinel-updater)

# Windows
where sentinel-updater
```

#### 2. Updates Fail to Compile

**Symptoms:**
- Logs show "compilation failed" errors
- Error: "go: command not found"
- Error: "gcc: command not found"

**Solutions:**

**Verify Go installation:**
```bash
go version
go env GOROOT GOPATH
```

**Verify CGO is enabled:**
```bash
go env CGO_ENABLED
# Should output: 1
```

**Linux - Install build tools:**
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install build-essential

# CentOS/RHEL
sudo yum groupinstall "Development Tools"

# Arch Linux
sudo pacman -S base-devel
```

**macOS - Install Xcode Command Line Tools:**
```bash
xcode-select --install
```

**Windows - Install GCC:**
- Download and install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)
- Or install [MinGW-w64](https://www.mingw-w64.org/)
- Add GCC to PATH:
  ```powershell
  $env:PATH += ";C:\TDM-GCC-64\bin"
  [Environment]::SetEnvironmentVariable("PATH", $env:PATH, "Machine")
  ```

#### 3. Permission Denied Errors

**Symptoms:**
- Error: "permission denied" when accessing files
- Error: "operation not permitted"

**Solutions:**

**Check data directory permissions:**
```bash
# Linux/macOS
sudo ls -ld /var/lib/sentinelgo
sudo chmod 755 /var/lib/sentinelgo
sudo chown root:root /var/lib/sentinelgo

# Windows (run as Administrator)
icacls C:\ProgramData\SentinelGo
```

**Ensure service runs with elevated privileges:**
```bash
# Linux - Check service user
sudo systemctl show sentinelgo-updater | grep User

# macOS - Check plist UserName
sudo plutil -p /Library/LaunchDaemons/com.sentinelgo.updater.plist | grep UserName

# Windows - Service should run as LocalSystem
sc qc sentinelgo-updater
```

#### 4. Main Agent Fails to Start After Update

**Symptoms:**
- Update completes but main agent doesn't start
- Logs show "service verification failed"

**Solutions:**

**Check main agent service status:**
```bash
# Linux
sudo systemctl status sentinelgo

# macOS
sudo launchctl list | grep sentinelgo

# Windows
sc query sentinelgo
```

**Check main agent logs:**
```bash
# Linux/macOS
sudo tail -f /var/lib/sentinelgo/agent.log

# Windows
Get-Content C:\ProgramData\SentinelGo\agent.log -Tail 50
```

**Manually start the main agent:**
```bash
# Linux
sudo systemctl start sentinelgo

# macOS
sudo launchctl start com.sentinelgo.agent

# Windows
sc start sentinelgo
```

**Check binary permissions:**
```bash
# Linux/macOS
ls -l /usr/local/bin/sentinel
sudo chmod +x /usr/local/bin/sentinel

# Windows
where sentinel
```

#### 5. Rollback Failures

**Symptoms:**
- Update fails and rollback also fails
- Both old and new versions fail to start

**Solutions:**

**Manual rollback procedure:**

1. Stop both services:
   ```bash
   sudo sentinel-updater stop
   sudo systemctl stop sentinelgo  # or launchctl/sc
   ```

2. Restore from backup (if available):
   ```bash
   # Linux/macOS
   sudo cp /usr/local/bin/sentinel.backup /usr/local/bin/sentinel
   sudo chmod +x /usr/local/bin/sentinel
   
   # Windows
   copy C:\Program Files\SentinelGo\sentinel.exe.backup C:\Program Files\SentinelGo\sentinel.exe
   ```

3. Reinstall the main agent service:
   ```bash
   sudo sentinel install
   ```

4. Start services:
   ```bash
   sudo sentinel-updater start
   sudo systemctl start sentinelgo  # or launchctl/sc
   ```

#### 6. Database Issues

**Symptoms:**
- Error: "database is locked"
- Error: "unable to open database file"

**Solutions:**

**Check database file:**
```bash
# Linux/macOS
sudo ls -l /var/lib/sentinelgo/sentinel.db
sudo chmod 644 /var/lib/sentinelgo/sentinel.db

# Windows
dir C:\ProgramData\SentinelGo\sentinel.db
```

**Check for processes using the database:**
```bash
# Linux
sudo lsof /var/lib/sentinelgo/sentinel.db

# macOS
sudo lsof /var/lib/sentinelgo/sentinel.db

# Windows
handle sentinel.db
```

**Stop services and retry:**
```bash
sudo systemctl stop sentinelgo sentinelgo-updater
sudo systemctl start sentinelgo-updater
sudo systemctl start sentinelgo
```

#### 7. Log Rotation Issues

**Symptoms:**
- Disk space running low
- Log files growing too large

**Solutions:**

**Check log file sizes:**
```bash
# Linux/macOS
du -h /var/lib/sentinelgo/*.log

# Windows
Get-ChildItem C:\ProgramData\SentinelGo\*.log | Select-Object Name, Length
```

**Manually rotate logs:**
```bash
# Linux/macOS
sudo systemctl stop sentinelgo-updater
sudo mv /var/lib/sentinelgo/updater.log /var/lib/sentinelgo/updater.log.old
sudo systemctl start sentinelgo-updater

# Windows
Stop-Service sentinelgo-updater
Move-Item C:\ProgramData\SentinelGo\updater.log C:\ProgramData\SentinelGo\updater.log.old
Start-Service sentinelgo-updater
```

**Configure log rotation (Linux):**
```bash
# Create logrotate config
sudo tee /etc/logrotate.d/sentinelgo-updater <<EOF
/var/lib/sentinelgo/updater.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
    postrotate
        systemctl reload sentinelgo-updater > /dev/null 2>&1 || true
    endscript
}
EOF
```

#### 8. Network/Connectivity Issues

**Symptoms:**
- Error: "failed to check latest version"
- Error: "connection timeout"
- Error: "unable to download module"

**Solutions:**

**Check network connectivity:**
```bash
# Test Go module proxy
curl -I https://proxy.golang.org

# Test GitHub connectivity
curl -I https://github.com

# Check DNS resolution
nslookup proxy.golang.org
```

**Configure Go proxy (if behind firewall):**
```bash
# Set Go proxy environment variable
export GOPROXY=https://proxy.golang.org,direct

# Or use a private proxy
export GOPROXY=https://your-private-proxy.com
```

**Check firewall rules:**
```bash
# Linux (iptables)
sudo iptables -L -n | grep -i drop

# Linux (firewalld)
sudo firewall-cmd --list-all

# Windows
Get-NetFirewallRule | Where-Object {$_.Enabled -eq 'True' -and $_.Direction -eq 'Outbound'}
```

### Debug Mode

Enable debug logging for more detailed information:

**Linux (systemd):**
```bash
sudo systemctl edit sentinelgo-updater
# Add:
# [Service]
# Environment="LOG_LEVEL=debug"

sudo systemctl restart sentinelgo-updater
```

**macOS (launchd):**
```bash
sudo nano /Library/LaunchDaemons/com.sentinelgo.updater.plist
# Add LOG_LEVEL=debug to EnvironmentVariables

sudo launchctl unload /Library/LaunchDaemons/com.sentinelgo.updater.plist
sudo launchctl load /Library/LaunchDaemons/com.sentinelgo.updater.plist
```

**Windows:**
```powershell
[Environment]::SetEnvironmentVariable("LOG_LEVEL", "debug", "Machine")
Restart-Service sentinelgo-updater
```

### Getting Help

If you encounter issues not covered here:

1. **Check the logs** for detailed error messages
2. **Enable debug mode** for more verbose logging
3. **Review the [Troubleshooting Guide](../docs/troubleshooting-guide.md)** for additional solutions
4. **Contact support** with:
   - Log files from `/var/lib/sentinelgo/updater.log` (or Windows equivalent)
   - Service status output
   - System information (OS, Go version, GCC version)
   - Steps to reproduce the issue

## License

[Your License Here]

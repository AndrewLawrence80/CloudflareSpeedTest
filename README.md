# CloudflareSpeedTest

English | [简体中文](README_zh.md)

Simplified version inspired by [XIU2 CloudflareSpeedTest](https://github.com/XIU2/CloudflareSpeedTest)

This implementation does not enumerate cloudflare ipv4 or ipv6 actively, but checks whether a domain is hosted on cloudflare
by resolving the domain ip through DNS. Domain list are imported from [v2fly domain list community](https://github.com/v2fly/domain-list-community)

## Features

- Automated DNS resolution of domains from v2fly domain list community
- IPv4 and IPv6 ICMP ping tests to measure latency
- HTTP download tests to measure bandwidth
- Results stored in SQLite database for analysis
- Export results to CSV files
- Configurable concurrency and rate limiting

## Installation

### Download Pre-built Binaries

You can download pre-built binaries for your platform from the [Releases](https://github.com/AndrewLawrence80/CloudflareSpeedTest/releases) page.

Available platforms:
- **Linux**: amd64, arm64
- **Windows**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)

Each release includes the binary, configuration example, and Cloudflare IP range file.

### Build from Source

#### Clone the Repository

This project uses the [v2fly domain-list-community](https://github.com/v2fly/domain-list-community) as a git submodule for domain lists.

**Option 1: Clone with submodules (recommended)**

```sh
git clone --recurse-submodules https://github.com/AndrewLawrence80/CloudflareSpeedTest.git
cd CloudflareSpeedTest
```

**Option 2: Clone and initialize submodules separately**

If you've already cloned the repository without submodules:

```sh
git clone https://github.com/AndrewLawrence80/CloudflareSpeedTest.git
cd CloudflareSpeedTest
git submodule init
git submodule update
```

**Alternative: Manual clone of domain list**

If you prefer not to use git submodules, you can clone the domain list separately:

```sh
git clone https://github.com/v2fly/domain-list-community.git
```

Then update the `DOMAIN_LIST_PATH` in your `.env` file to point to the correct location.

#### Compile

You need golang to compile the repo

```sh
go build .
```

## Configuration

Create a `.env` file in the same directory as the executable (or copy from `.env.example`):

```bash
# Log file config
LOG_FILE_PATH=./speedtest.log  # path to log file, set empty to output logs to stdout
LOG_LEVEL=info

# v2fly domain list submodule path
DOMAIN_LIST_PATH=./domain-list-community/data

# Speedtest config
ICMP_COUNT=4                          # number of ICMP echo requests to send
ICMP_TIMEOUT=3                        # timeout in seconds for each ICMP echo request
ICMP_INTERVAL=1                       # interval in seconds between ICMP echo requests
ICMP_PACKETLOSS_THRESHOLD=0.25        # packet loss rate threshold (0.25 = 25%)
HTTP_TIMEOUT=20                       # timeout in seconds for HTTP requests

# Concurrency config
NUM_DNS_WORKERS=512                   # number of concurrent workers for DNS resolution
NUM_ICMP_WORKERS=512                  # number of concurrent workers for ICMP pinging
NUM_HTTP_WORKERS=1                    # number of concurrent workers for HTTP testing

# QPM config
QPM_DNS=0                             # queries per minute for DNS resolution, set to 0 for no limit

# Test config
TEST_URL=https://yourdomain.com/test.img  # URL to use for download speed testing
TOP_N_IPS=10                          # number of top IPs to test for bandwidth
CLOUDFLARE_IP_RANGE_FILE_PATH=cloudflare_ip_range.txt  # path to Cloudflare IP ranges file
```

## Usage

### Quick Start - Run Full Pipeline

Run the complete test pipeline (DNS resolution, ICMP ping, and bandwidth tests):

```sh
./CloudflareSpeedTest test-pipeline
```

This command will:
1. Build the DNS database by resolving domains from the v2fly domain list
2. Export DNS records to CSV
3. Perform ICMPv4 ping tests
4. Export ICMPv4 results to CSV
5. Perform bandwidth tests on top IPv4 addresses
6. Export bandwidth results to CSV
7. Automatically detect IPv6 support and run IPv6 tests if available

### Individual Commands

#### 1. Build DNS Database

Resolve domains from the v2fly domain list and populate the database:

```sh
./CloudflareSpeedTest build-db
```

This command loads all possible domains and performs DNS lookups to identify Cloudflare-hosted domains.

#### 2. ICMP Ping Tests

Perform ICMP ping tests on IPv4 addresses:

```sh
./CloudflareSpeedTest icmpv4-ping
```

Perform ICMP ping tests on IPv6 addresses:

```sh
./CloudflareSpeedTest icmpv6-ping
```

These commands measure latency and packet loss for Cloudflare IP addresses.

#### 3. Bandwidth Tests

Test download bandwidth on top IPv4 addresses (by RTT):

```sh
./CloudflareSpeedTest bandwidthv4
```

Test download bandwidth on top IPv6 addresses (by RTT):

```sh
./CloudflareSpeedTest bandwidthv6
```

These commands perform HTTP download tests on the IPs with the lowest latency and packet loss.

#### 4. Export Results

Export DNS records to CSV:

```sh
./CloudflareSpeedTest export-dns
```
Output: `dns_records.csv`

Export ICMPv4 ping summaries to CSV:

```sh
./CloudflareSpeedTest export-icmp
```
Output: `icmp_summaries.csv`

Export ICMPv6 ping summaries to CSV:

```sh
./CloudflareSpeedTest export-icmpv6
```
Output: `icmpv6_summaries.csv`

Export IPv4 bandwidth test results to CSV:

```sh
./CloudflareSpeedTest export-bandwidth
```
Output: `bandwidth_summaries.csv`

Export IPv6 bandwidth test results to CSV:

```sh
./CloudflareSpeedTest export-bandwidthv6
```
Output: `bandwidthv6_summaries.csv`

#### 5. Version

Print version information:

```sh
./CloudflareSpeedTest version
```

## Typical Workflow

1. **Configure**: Copy `.env.example` to `.env` and adjust settings
2. **Build Database**: Run `./CloudflareSpeedTest build-db` to resolve domains
3. **Test Latency**: Run `./CloudflareSpeedTest icmpv4-ping` (and `icmpv6-ping` if IPv6 available)
4. **Test Bandwidth**: Run `./CloudflareSpeedTest bandwidthv4` (and `bandwidthv6` if IPv6 available)
5. **Export Results**: Use export commands to generate CSV reports

Or simply run `./CloudflareSpeedTest test-pipeline` to execute all steps automatically.

## Output Files

- `speedtest.log` - Log file (if configured)
- `speedtest.db` - SQLite database with all test results
- `dns_records.csv` - Exported DNS resolution results
- `icmp_summaries.csv` - Exported ICMPv4 ping test results
- `icmpv6_summaries.csv` - Exported ICMPv6 ping test results
- `bandwidth_summaries.csv` - Exported IPv4 bandwidth test results
- `bandwidthv6_summaries.csv` - Exported IPv6 bandwidth test results

## Notes

- The tool requires root/administrator privileges to perform ICMP ping tests
- Set `NUM_HTTP_WORKERS=1` to avoid overwhelming your network during bandwidth tests
- The `TOP_N_IPS` setting determines how many IPs with the best latency will be tested for bandwidth
- IPs with packet loss above `ICMP_PACKETLOSS_THRESHOLD` will be excluded from bandwidth tests

## Development

### Creating a Release

The project uses GitHub Actions to automatically build and publish releases. To create a new release:

1. Tag your commit with a version number:
   ```sh
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. GitHub Actions will automatically:
   - Build binaries for all supported platforms (Linux, Windows, macOS)
   - Build for both amd64 and arm64 architectures
   - Create source tarball
   - Generate SHA256 checksums
   - Create a GitHub release with all artifacts

The workflow is triggered by pushing tags matching the pattern `v*.*.*` (e.g., `v1.0.0`, `v2.1.3`).

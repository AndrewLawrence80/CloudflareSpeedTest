# CloudflareSpeedTest

[English](README.md) | 简体中文

[XIU2 CloudflareSpeedTest](https://github.com/XIU2/CloudflareSpeedTest) 的简化版本

本实现不主动枚举 Cloudflare 的 IPv4 或 IPv6 地址，而是通过 DNS 解析域名 IP 来检查域名是否托管在 Cloudflare 上。域名列表来自 [v2fly domain list community](https://github.com/v2fly/domain-list-community)

## 功能特性

- 自动从 v2fly 域名列表社区解析域名
- IPv4 和 IPv6 ICMP Ping 测试以测量延迟
- HTTP 下载测试以测量带宽
- 结果存储在 SQLite 数据库中便于分析
- 导出结果为 CSV 文件
- 可配置的并发和速率限制

## 安装

### 下载预编译二进制文件

你可以从 [Releases](https://github.com/AndrewLawrence80/CloudflareSpeedTest/releases) 页面下载适合你平台的预编译二进制文件。

支持的平台：
- **Linux**: amd64, arm64
- **Windows**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)

每个发布版本都包含二进制文件、配置示例和 Cloudflare IP 范围文件。

### 从源码构建

#### 克隆仓库

本项目使用 [v2fly 域名列表社区](https://github.com/v2fly/domain-list-community) 作为 git 子模块提供域名列表。

**选项 1: 连同子模块一起克隆（推荐）**

```sh
git clone --recurse-submodules https://github.com/AndrewLawrence80/CloudflareSpeedTest.git
cd CloudflareSpeedTest
```

**选项 2: 分别克隆和初始化子模块**

如果你已经克隆了仓库但没有包含子模块：

```sh
git clone https://github.com/AndrewLawrence80/CloudflareSpeedTest.git
cd CloudflareSpeedTest
git submodule init
git submodule update
```

**替代方案：手动克隆域名列表**

如果你不想使用 git 子模块，可以单独克隆域名列表：

```sh
git clone https://github.com/v2fly/domain-list-community.git
```

然后在 `.env` 文件中更新 `DOMAIN_LIST_PATH` 指向正确的位置。

#### 编译

你需要安装 golang 来编译本仓库

```sh
go build .
```

## 配置

在可执行文件同一目录下创建 `.env` 文件（或从 `.env.example` 复制）：

```bash
# 日志文件配置
LOG_FILE_PATH=./speedtest.log  # 日志文件路径，设置为空则将日志输出到 stdout
LOG_LEVEL=info

# v2fly 域名列表子模块路径
DOMAIN_LIST_PATH=./domain-list-community/data

# 测速配置
ICMP_COUNT=4                          # 发送的 ICMP 回显请求数量
ICMP_TIMEOUT=3                        # 每个 ICMP 回显请求的超时时间（秒）
ICMP_INTERVAL=1                       # ICMP 回显请求之间的间隔（秒）
ICMP_PACKETLOSS_THRESHOLD=0.25        # 丢包率阈值（0.25 = 25%）
HTTP_TIMEOUT=20                       # HTTP 请求的超时时间（秒）

# 并发配置
NUM_DNS_WORKERS=512                   # DNS 解析的并发工作线程数
NUM_ICMP_WORKERS=512                  # ICMP Ping 的并发工作线程数
NUM_HTTP_WORKERS=1                    # HTTP 测试的并发工作线程数

# QPM 配置
QPM_DNS=0                             # DNS 解析的每分钟查询数，设置为 0 表示无限制

# 测试配置
TEST_URL=https://yourdomain.com/test.img  # 用于下载速度测试的 URL
TOP_N_IPS=10                          # 要测试带宽的前 N 个 IP 数量
CLOUDFLARE_IP_RANGE_FILE_PATH=cloudflare_ip_range.txt  # Cloudflare IP 范围文件路径
```

## 使用方法

### 快速开始 - 运行完整流程

运行完整的测试流程（DNS 解析、ICMP Ping 和带宽测试）：

```sh
./CloudflareSpeedTest test-pipeline
```

该命令将执行：
1. 通过解析 v2fly 域名列表构建 DNS 数据库
2. 导出 DNS 记录到 CSV
3. 执行 ICMPv4 Ping 测试
4. 导出 ICMPv4 结果到 CSV
5. 对延迟最低的 IPv4 地址执行带宽测试
6. 导出带宽测试结果到 CSV
7. 自动检测 IPv6 支持并在可用时运行 IPv6 测试

### 单独命令

#### 1. 构建 DNS 数据库

从 v2fly 域名列表解析域名并填充数据库：

```sh
./CloudflareSpeedTest build-db
```

此命令加载所有可能的域名并执行 DNS 查询以识别托管在 Cloudflare 上的域名。

#### 2. ICMP Ping 测试

对 IPv4 地址执行 ICMP Ping 测试：

```sh
./CloudflareSpeedTest icmpv4-ping
```

对 IPv6 地址执行 ICMP Ping 测试：

```sh
./CloudflareSpeedTest icmpv6-ping
```

这些命令测量 Cloudflare IP 地址的延迟和丢包率。

#### 3. 带宽测试

测试延迟最低的 IPv4 地址的下载带宽：

```sh
./CloudflareSpeedTest bandwidthv4
```

测试延迟最低的 IPv6 地址的下载带宽：

```sh
./CloudflareSpeedTest bandwidthv6
```

这些命令对延迟最低且丢包率低的 IP 执行 HTTP 下载测试。

#### 4. 导出结果

导出 DNS 记录到 CSV：

```sh
./CloudflareSpeedTest export-dns
```
输出：`dns_records.csv`

导出 ICMPv4 Ping 摘要到 CSV：

```sh
./CloudflareSpeedTest export-icmp
```
输出：`icmp_summaries.csv`

导出 ICMPv6 Ping 摘要到 CSV：

```sh
./CloudflareSpeedTest export-icmpv6
```
输出：`icmpv6_summaries.csv`

导出 IPv4 带宽测试结果到 CSV：

```sh
./CloudflareSpeedTest export-bandwidth
```
输出：`bandwidth_summaries.csv`

导出 IPv6 带宽测试结果到 CSV：

```sh
./CloudflareSpeedTest export-bandwidthv6
```
输出：`bandwidthv6_summaries.csv`

#### 5. 版本

打印版本信息：

```sh
./CloudflareSpeedTest version
```

## 典型工作流程

1. **配置**：复制 `.env.example` 到 `.env` 并调整设置
2. **构建数据库**：运行 `./CloudflareSpeedTest build-db` 解析域名
3. **测试延迟**：运行 `./CloudflareSpeedTest icmpv4-ping`（如果支持 IPv6 则运行 `icmpv6-ping`）
4. **测试带宽**：运行 `./CloudflareSpeedTest bandwidthv4`（如果支持 IPv6 则运行 `bandwidthv6`）
5. **导出结果**：使用导出命令生成 CSV 报告

或者简单地运行 `./CloudflareSpeedTest test-pipeline` 自动执行所有步骤。

## 输出文件

- `speedtest.log` - 日志文件（如果配置了）
- `speedtest.db` - 包含所有测试结果的 SQLite 数据库
- `dns_records.csv` - 导出的 DNS 解析结果
- `icmp_summaries.csv` - 导出的 ICMPv4 Ping 测试结果
- `icmpv6_summaries.csv` - 导出的 ICMPv6 Ping 测试结果
- `bandwidth_summaries.csv` - 导出的 IPv4 带宽测试结果
- `bandwidthv6_summaries.csv` - 导出的 IPv6 带宽测试结果

## 注意事项

- 该工具需要 root/管理员权限才能执行 ICMP Ping 测试
- 设置 `NUM_HTTP_WORKERS=1` 以避免在带宽测试期间压垮网络
- `TOP_N_IPS` 设置决定了将要测试带宽的延迟最低的 IP 数量
- 丢包率高于 `ICMP_PACKETLOSS_THRESHOLD` 的 IP 将被排除在带宽测试之外

## 开发

### 创建发布版本

项目使用 GitHub Actions 自动构建和发布版本。要创建新的发布版本：

1. 为你的提交打上版本号标签：
   ```sh
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. GitHub Actions 将自动：
   - 为所有支持的平台（Linux、Windows、macOS）构建二进制文件
   - 为 amd64 和 arm64 架构构建
   - 创建源码压缩包
   - 生成 SHA256 校验和
   - 创建包含所有构建产物的 GitHub 发布版本

工作流由推送符合 `v*.*.*` 模式的标签触发（例如：`v1.0.0`、`v2.1.3`）。

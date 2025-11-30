# AutoCert

🔒 **Let's Encrypt HTTPS 证书一键安装部署工具**

AutoCert 是一个跨平台的 Let's Encrypt HTTPS 证书管理工具，支持一键安装、自动更新、跨机器迁移等功能，简化 SSL/TLS 证书的部署和管理流程。

## 🏗️ 项目架构

### 工作原理

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   CLI 命令   │ ──▶ │  证书管理器  │ ──▶ │  ACME 验证  │ ──▶ │ Web服务器配置│
│  (Cobra)    │     │  (Manager)  │     │ (Challenge) │     │(Configurator)│
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

**核心流程**：
1. **CLI 解析** → `cmd/` 使用 Cobra 处理用户命令
2. **证书申请** → `internal/cert/` 生成 CSR，执行 ACME 验证
3. **验证模式** → Webroot / Standalone / DNS 三种挑战方式
4. **服务器配置** → `internal/webserver/` 自动配置 Nginx/Apache/IIS

### 目录结构

```
autocert/
├── main.go                 # 程序入口
├── cmd/                    # CLI 命令定义
│   ├── root.go            # 根命令和全局配置
│   ├── install.go         # 安装证书命令
│   ├── manage.go          # 续期/状态/定时任务
│   ├── backup.go          # 导出/导入命令
│   └── version.go         # 版本信息
├── internal/              # 内部模块
│   ├── cert/              # 证书管理核心
│   │   └── manager.go     # 统一证书管理器
│   ├── webserver/         # Web 服务器配置器
│   │   └── configurator.go
│   ├── config/            # 配置文件管理
│   ├── logger/            # 日志模块
│   ├── scheduler/         # 定时任务调度
│   └── backup/            # 备份恢复功能
├── scripts/               # 安装和打包脚本
│   ├── install.sh         # Linux/macOS 安装脚本
│   ├── install.ps1        # Windows 安装脚本
│   ├── package.sh         # Linux/macOS 打包脚本
│   └── package.ps1        # Windows 打包脚本
└── docs/                  # 文档目录
```

### 核心依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| `github.com/spf13/cobra` | v1.8.0 | CLI 命令框架 |
| `github.com/spf13/viper` | v1.18.2 | 配置文件管理 |
| `github.com/sirupsen/logrus` | v1.9.3 | 结构化日志 |
| `github.com/go-acme/lego` | v4.x | ACME 协议实现 |

## ✨ 特性

- 🚀 **一键安装** - 只需运行一个命令即可完成证书申请与安装
- 🔄 **自动续期** - 内置定时任务，自动检测并更新证书
- 🌐 **跨平台支持** - 兼容 Linux (Ubuntu, CentOS, Debian, AlmaLinux) 和 Windows
- 🔧 **多服务器支持** - 支持 Nginx、Apache、IIS
- 📦 **无侵入性** - 与现有配置无缝衔接，不覆盖已有设置
- 🔄 **可迁移** - 证书及配置文件可导出/导入，方便机器间快速部署
- 📊 **详细日志** - 完整的操作日志和状态监控
- 🔔 **通知支持** - 邮件通知证书更新结果

## 🚀 快速开始

### 一键安装

**Linux/macOS:**
```bash
curl -sSL https://ftmi.info/install.sh | bash
```

**Windows (PowerShell 管理员模式):**
```powershell
iwr -useb https://ftmi.info/install.ps1 | iex
```

### 基本使用

1. **安装单域名证书**
   ```bash
   # Nginx
   autocert install --domain example.com --email admin@example.com --nginx
   
   # Apache
   autocert install --domain example.com --email admin@example.com --apache
   
   # IIS (Windows)
   autocert install --domain example.com --email admin@example.com --iis
   ```

2. **安装二级域名证书**
   ```bash
   # 二级域名
   autocert install --domain api.example.com --email admin@example.com --nginx
   autocert install --domain www.example.com --email admin@example.com --nginx
   ```

3. **安装多域名证书（SAN证书）**
   ```bash
   # 主域名 + www 子域名
   autocert install --domains "example.com,www.example.com" --email admin@example.com --nginx
   
   # 多个子域名
   autocert install --domains "api.example.com,www.example.com,blog.example.com" --email admin@example.com --nginx
   ```

4. **安装泛域名证书（必须使用 DNS 验证）**
   ```bash
   # 泛域名证书
   autocert install --domain "*.example.com" --email admin@example.com --nginx --dns
   
   # 混合域名（主域名 + 泛域名）
   autocert install --domains "example.com,*.example.com" --email admin@example.com --nginx --dns
   ```

5. **设置自动续期**
   ```bash
   autocert schedule install
   ```

6. **查看证书状态**
   ```bash
   autocert status
   ```

7. **手动续期**
   ```bash
   autocert renew --domain example.com
   ```

## 🌐 域名类型支持

AutoCert 支持多种类型的域名证书申请：

### 📄 单域名证书
为单个域名申请证书：
```bash
autocert install --domain example.com --email admin@example.com --nginx
```

### 🌍 二级域名证书
为子域名申请证书：
```bash
# API 子域名
autocert install --domain api.example.com --email admin@example.com --nginx

# WWW 子域名
autocert install --domain www.example.com --email admin@example.com --nginx

# 博客子域名
autocert install --domain blog.example.com --email admin@example.com --nginx
```

### 📋 多域名证书（SAN 证书）
在一个证书中包含多个域名：
```bash
# 主域名 + www 子域名
autocert install --domains "example.com,www.example.com" --email admin@example.com --nginx

# 多个子域名
autocert install --domains "api.example.com,www.example.com,blog.example.com,admin.example.com" --email admin@example.com --nginx

# 主域名 + 多个子域名
autocert install --domains "example.com,www.example.com,api.example.com" --email admin@example.com --nginx
```

### ✨ 泛域名证书（通配符证书）
使用通配符匹配所有子域名（必须使用 DNS 验证）：
```bash
# 泛域名证书
autocert install --domain "*.example.com" --email admin@example.com --nginx --dns

# 混合证书（主域名 + 泛域名）
autocert install --domains "example.com,*.example.com" --email admin@example.com --nginx --dns

# 多主域名 + 泛域名
autocert install --domains "example.com,www.example.com,*.example.com" --email admin@example.com --nginx --dns
```

> ⚠️ **注意**：泛域名证书只能使用 DNS 验证模式，需要手动在 DNS 服务商中添加 TXT 记录。

### 📊 域名类型对比

| 类型 | 优点 | 适用场景 | 验证模式 |
|------|------|----------|----------|
| 单域名 | 简单、快速 | 单个网站 | Webroot/Standalone |
| 二级域名 | 独立管理 | 子服务、API | Webroot/Standalone |
| 多域名 | 统一管理 | 多个固定域名 | Webroot/Standalone |
| 泛域名 | 灵活扩展 | 动态子域名 | DNS 专用 |

📖 **详细指南**：查看 [docs/wildcard-and-subdomain-guide.md](docs/wildcard-and-subdomain-guide.md) 获取完整的使用指南。

## 📖 详细文档

### 安装方式

#### 方式一：一键安装脚本（推荐）

一键安装脚本会自动检测系统环境，下载合适的二进制文件，并完成基础配置。

#### 方式二：手动下载

1. 从 [Releases](https://github.com/autocert/autocert/releases) 页面下载对应平台的二进制文件
2. 解压到系统 PATH 目录
3. 运行 `autocert --help` 验证安装

#### 方式三：源码编译

```bash
git clone https://github.com/renky1025/autcert.git
cd autocert
make build
sudo make install
```

### 🚀 构建和发布

#### 基本构建
```bash
# 构建单平台二进制文件
make build

# 构建所有平台
make build-all
```

#### 一键打包（标准格式）

**Linux/macOS 环境：**
```bash
# 打包所有平台
make package

# 打包特定平台
make package-linux
make package-windows

# 完整发布流程（清理+测试+打包）
make release

# 直接使用打包脚本
./scripts/package.sh v1.0.0 dist autocert all
```

**Windows 环境：**
```powershell
# PowerShell 打包脚本
.\scripts\package-simple.ps1 -Version "v1.0.0" -Platform "all"
.\scripts\package-simple.ps1 -Version "v1.0.0" -Platform "windows"

# 批处理打包
.\scripts\build-release.bat v1.0.0 all
```

#### 打包输出格式

AutoCert 支持生成标准格式的发布包：

**Linux/macOS 包格式：**
```
autocert_${VERSION}_linux_${ARCH}.tar.gz
autocert_${VERSION}_darwin_${ARCH}.tar.gz
```

**Windows 包格式：**
```
autocert_${VERSION}_windows_${ARCH}.zip
```

**支持的架构：** `amd64` (x86_64), `arm64` (ARM64)

**示例输出：**
```
dist/
├── autocert_v1.0.0_linux_amd64.tar.gz
├── autocert_v1.0.0_linux_arm64.tar.gz
├── autocert_v1.0.0_windows_amd64.zip
├── autocert_v1.0.0_windows_arm64.zip
├── autocert_v1.0.0_darwin_amd64.tar.gz
└── autocert_v1.0.0_darwin_arm64.tar.gz
```

📖 **详细指南：** 查看 [docs/packaging-guide.md](docs/packaging-guide.md) 获取完整的打包说明。

### 命令参考

#### 主要命令

| 命令 | 说明 |
|------|------|
| `install` | 安装和配置 HTTPS 证书 |
| `renew` | 续期证书 |
| `status` | 查看证书状态 |
| `schedule` | 管理定时任务 |
| `export` | 导出证书和配置 |
| `import` | 导入证书和配置 |
| `version` | 显示版本信息 |

#### install 命令详解

```bash
autocert install [flags]

Flags:
  -d, --domain string     要申请证书的单个域名
      --domains string    多个域名，用逗号分隔 (例: example.com,www.example.com,*.example.com)
  -e, --email string      用于 Let's Encrypt 账户的邮箱地址 (必需)
  -w, --webroot string    Webroot 模式的网站根目录路径
      --standalone        使用 Standalone 模式验证
      --dns               使用 DNS 验证模式（泛域名证书必需）
      --nginx             配置 Nginx
      --apache            配置 Apache  
      --iis               配置 IIS
```

**域名类型示例：**
```bash
# 单域名证书
autocert install --domain example.com --email admin@example.com --nginx

# 二级域名证书
autocert install --domain api.example.com --email admin@example.com --nginx

# 多域名证书（SAN证书）
autocert install --domains "example.com,www.example.com,api.example.com" --email admin@example.com --nginx

# 泛域名证书（需要 DNS 验证）
autocert install --domain "*.example.com" --email admin@example.com --nginx --dns

# 混合域名（主域名 + 泛域名）
autocert install --domains "example.com,*.example.com" --email admin@example.com --nginx --dns
```

**验证模式选择：**
- **Webroot 模式**：适用于已有运行的 Web 服务器，不支持泛域名
- **Standalone 模式**：临时启动验证服务器，不支持泛域名
- **DNS 模式**：支持所有类型域名，泛域名必须使用此模式

#### schedule 命令详解

```bash
# 安装定时任务
autocert schedule install --name autocert-renew

# 删除定时任务
autocert schedule remove --name autocert-renew

# 列出定时任务
autocert schedule list
```

#### 导出/导入命令

```bash
# 导出所有证书
autocert export --output certs.tar.gz

# 导出指定域名证书
autocert export --output example-cert.tar.gz --domain example.com

# 导入证书
autocert import certs.tar.gz --restore-schedule
```

### 配置文件

AutoCert 使用 YAML 格式的配置文件：

**Linux:** `/etc/autocert/config.yaml`  
**Windows:** `C:\ProgramData\AutoCert\config.yaml`

```yaml
# 基础配置
log_level: info
config_dir: /etc/autocert
cert_dir: /etc/autocert/certs
log_dir: /var/log

# ACME 配置
acme:
  server: https://acme-v02.api.letsencrypt.org/directory
  key_type: rsa
  key_size: 2048

# Web 服务器配置
webserver:
  type: nginx  # nginx, apache, iis
  config_path: /etc/nginx/nginx.conf
  reload_cmd: systemctl reload nginx

# 通知配置
notification:
  email:
    smtp: smtp.example.com
    port: 587
    username: user@example.com
    password: password
    from: noreply@example.com
    to: admin@example.com
```

### 目录结构

#### Linux
```
/etc/autocert/
├── config.yaml          # 主配置文件
├── certs/               # 证书目录
│   └── example.com/     # 域名证书目录
│       ├── cert.pem     # 证书文件
│       ├── key.pem      # 私钥文件
│       └── chain.pem    # 证书链文件
└── logs/                # 日志目录
```

#### Windows
```
C:\ProgramData\AutoCert\
├── config.yaml          # 主配置文件  
├── certs\               # 证书目录
│   └── example.com\     # 域名证书目录
│       ├── cert.pem     # 证书文件
│       ├── key.pem      # 私钥文件
│       └── chain.pem    # 证书链文件
└── logs\                # 日志目录
```

## 🔧 高级用法

### 批量域名管理

```bash
# 为多个域名安装证书
for domain in example.com www.example.com api.example.com; do
    autocert install --domain $domain --email admin@example.com --nginx
done
```

### 证书迁移

```bash
# 在源服务器导出
autocert export --output backup-$(date +%Y%m%d).tar.gz

# 传输到目标服务器
scp backup-20241201.tar.gz user@newserver:/tmp/

# 在目标服务器导入
autocert import /tmp/backup-20241201.tar.gz
```

### 自定义验证模式

```bash
# DNS 验证（需要配置 DNS API）
autocert install --domain example.com --email admin@example.com --dns cloudflare

# 指定 Webroot 路径
autocert install --domain example.com --email admin@example.com --webroot /var/www/example.com
```

## 🔍 故障排除

### 常见问题

**1. 端口 80/443 被占用**
```bash
# 检查端口占用
netstat -tlnp | grep :80
netstat -tlnp | grep :443

# 使用 webroot 模式而非 standalone 模式
autocert install --domain example.com --email admin@example.com --nginx --webroot /var/www/html
```

**2. DNS 解析问题**
```bash
# 检查域名解析
nslookup example.com
dig example.com

# 确保域名正确指向服务器 IP
```

**3. 权限问题**
```bash
# Linux: 确保以 root 权限运行
sudo autocert install --domain example.com --email admin@example.com --nginx

# Windows: 以管理员身份运行 PowerShell
```

**4. Web 服务器配置问题**
```bash
# 检查 Nginx 配置语法
nginx -t

# 检查 Apache 配置语法
apache2ctl configtest
```

### 日志查看

```bash
# Linux
tail -f /var/log/autocert.log

# Windows
Get-Content "C:\ProgramData\AutoCert\logs\autocert.log" -Wait
```

### 调试模式

```bash
# 启用详细输出
autocert install --domain example.com --email admin@example.com --nginx --verbose

# 查看配置
autocert status --domain example.com
```

## 🤝 贡献

欢迎贡献代码、报告问题或提出建议！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [Let's Encrypt](https://letsencrypt.org/) - 免费的 SSL/TLS 证书
- [acme.sh](https://github.com/acmesh-official/acme.sh) - ACME 协议实现参考
- [Cobra](https://github.com/spf13/cobra) - 强大的 CLI 框架

## 📞 支持

- 🐛 [问题反馈](https://github.com/renky1025/autcert/issues)
- 💬 [讨论](https://github.com/renky1025/autcert/discussions)

---

**⚡ 让 HTTPS 证书管理变得简单！**
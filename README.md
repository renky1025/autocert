# AutoCert

AutoCert 是一个用于申请、安装、续期和迁移 Let's Encrypt 证书的命令行工具，面向 Linux 和 Windows 环境，支持 Nginx、Apache、IIS。

## 项目现在能做什么

- 为单域名申请并安装证书
- 为多域名（SAN）证书申请并安装证书
- 为泛域名申请证书（通过 DNS 验证）
- 保存站点元数据，支持 `status` 和 `renew`
- 为支持无人值守续期的站点创建定时任务
- 导出/导入证书与相关配置，便于迁移

## 使用前先知道

- 泛域名必须使用 `--dns`
- 当前 `--dns` 为人工参与验证流程，不支持无人值守自动续期；安装时需要加 `--skip-schedule`
- `--standalone` 也不适合无人值守自动续期；如需使用它，同样应加 `--skip-schedule`
- 当前只有基于 `--webroot` 托管且保存了有效 `webroot` 的站点支持无人值守自动续期
- 实际使用时，显式传入 Web 服务器类型和 `--webroot` 最稳妥，不要依赖自动探测

## 安装与构建

要求：

- Go `1.24`

从源码构建：

```bash
make build
./build/autocert --help
```

常用构建命令：

```bash
make test
make build
make package
```

仓库内脚本：

- Linux/macOS 安装脚本：`scripts/install.sh`
- Windows 安装脚本：`scripts/install.ps1`
- Linux/macOS 打包脚本：`scripts/package.sh`
- Windows 打包脚本：`scripts/package.ps1`

## 快速开始

单域名，Nginx，Webroot：

```bash
autocert install \
  --domain example.com \
  --email admin@example.com \
  --nginx \
  --webroot /var/www/example.com
```

多域名（SAN）：

```bash
autocert install \
  --domains "example.com,www.example.com,api.example.com" \
  --email admin@example.com \
  --nginx \
  --webroot /var/www/example.com
```

Windows + IIS：

```powershell
autocert install `
  --domain example.com `
  --email admin@example.com `
  --iis `
  --webroot C:\inetpub\wwwroot
```

泛域名：

```bash
autocert install \
  --domain "*.example.com" \
  --email admin@example.com \
  --nginx \
  --dns \
  --skip-schedule
```

查看状态和续期：

```bash
autocert status
autocert renew --domain example.com
autocert renew --all
```

安装或移除自动续期任务：

```bash
autocert schedule install --name autocert-renew
autocert schedule remove --name autocert-renew
autocert schedule list
```

迁移证书：

```bash
autocert export --output autocert-backup.tar.gz
autocert import autocert-backup.tar.gz --restore-schedule
```

## 命令概览

| 命令 | 说明 |
| --- | --- |
| `install` | 申请证书并配置 Web 服务器 |
| `renew` | 续期一个或全部已托管证书 |
| `status` | 查看证书状态 |
| `schedule` | 安装、删除、列出续期任务 |
| `export` | 导出证书和配置 |
| `import` | 导入证书和配置 |
| `version` | 显示版本信息 |

## 配置与数据目录

CLI 配置文件：

- 可通过 `--config /path/to/file.yaml` 显式指定
- 默认会查找 `$HOME/.autocert.yaml` 和当前目录下的 `./.autocert.yaml`

默认数据目录：

- Linux: `/etc/autocert`
- Linux 证书目录: `/etc/autocert/certs`
- Windows: `%ProgramData%\AutoCert`
- Windows 证书目录: `%ProgramData%\AutoCert\certs`

## 开发

```bash
make test
make lint
make package
```

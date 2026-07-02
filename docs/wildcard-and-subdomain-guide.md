# 二级域名和泛域名证书申请指南

本指南详细介绍如何使用 AutoCert 为二级域名和泛域名申请 Let's Encrypt 证书。

## 📚 概念说明

### 二级域名
二级域名是在主域名下创建的子域名，例如：
- `api.example.com`
- `www.example.com`  
- `blog.example.com`
- `admin.example.com`

### 泛域名（通配符域名）
泛域名使用通配符 `*` 来匹配一个域名下的所有子域名，例如：
- `*.example.com` 可以匹配 `api.example.com`、`www.example.com`、`blog.example.com` 等

### SAN 证书（多域名证书）
Subject Alternative Name (SAN) 证书可以在一个证书中包含多个域名，支持：
- 多个不同的域名
- 主域名 + 子域名组合
- 普通域名 + 泛域名组合

## 🚀 使用示例

### 1. 单个二级域名证书

```bash
# 基本用法
autocert install --domain api.example.com --email admin@example.com --nginx

# 使用 webroot 模式
autocert install --domain api.example.com --email admin@example.com --nginx --webroot /var/www/api

# 使用 standalone 模式
autocert install --domain api.example.com --email admin@example.com --nginx --standalone
```

### 2. 泛域名证书（必须使用 DNS 验证）

```bash
# 泛域名证书
autocert install --domain "*.example.com" --email admin@example.com --nginx --dns

# 注意：泛域名证书只能使用 DNS 验证模式
```

### 3. 多域名证书（SAN 证书）

```bash
# 主域名 + www 子域名
autocert install --domains "example.com,www.example.com" --email admin@example.com --nginx

# 多个子域名
autocert install --domains "api.example.com,www.example.com,blog.example.com" --email admin@example.com --nginx

# 主域名 + 多个子域名
autocert install --domains "example.com,www.example.com,api.example.com,admin.example.com" --email admin@example.com --nginx
```

### 4. 混合域名证书（主域名 + 泛域名）

```bash
# 主域名 + 泛域名（需要 DNS 验证）
autocert install --domains "example.com,*.example.com" --email admin@example.com --nginx --dns

# 多个主域名 + 泛域名
autocert install --domains "example.com,www.example.com,*.example.com" --email admin@example.com --nginx --dns
```

## 🔧 验证模式选择

### Webroot 模式
- **适用于**：已有运行的 Web 服务器
- **支持域名**：普通域名、二级域名
- **不支持**：泛域名
- **原理**：在网站根目录下创建验证文件

```bash
autocert install --domain api.example.com --email admin@example.com --nginx --webroot /var/www/api
```

### Standalone 模式  
- **适用于**：临时停止 Web 服务器进行验证
- **支持域名**：普通域名、二级域名
- **不支持**：泛域名
- **原理**：临时启动内置 HTTP 服务器

```bash
autocert install --domain api.example.com --email admin@example.com --nginx --standalone
```

### DNS 模式
- **适用于**：所有类型域名
- **支持域名**：普通域名、二级域名、泛域名
- **必需场景**：泛域名证书
- **原理**：在 DNS 中添加 TXT 记录进行验证

```bash
autocert install --domain "*.example.com" --email admin@example.com --nginx --dns
```

## 📋 DNS 验证步骤

当使用 DNS 验证模式时，需要手动添加 DNS 记录：

### 步骤 1：运行命令
```bash
autocert install --domain "*.example.com" --email admin@example.com --nginx --dns
```

### 步骤 2：添加 DNS 记录
程序会提示需要添加的 DNS TXT 记录：

```
需要为泛域名添加 DNS TXT 记录:
记录名: _acme-challenge.example.com
记录类型: TXT
记录值: [系统生成的验证值]
```

### 步骤 3：等待 DNS 传播
DNS 记录生效通常需要几分钟到几小时，可以使用以下命令检查：

```bash
# 检查 DNS 记录是否生效
nslookup -type=TXT _acme-challenge.example.com
dig TXT _acme-challenge.example.com
```

## 🌐 不同 Web 服务器配置

### Nginx 配置

```bash
# 单域名
autocert install --domain api.example.com --email admin@example.com --nginx

# 多域名
autocert install --domains "example.com,www.example.com,api.example.com" --email admin@example.com --nginx

# 泛域名
autocert install --domain "*.example.com" --email admin@example.com --nginx --dns
```

### Apache 配置

```bash
# 单域名
autocert install --domain api.example.com --email admin@example.com --apache

# 多域名
autocert install --domains "example.com,www.example.com,api.example.com" --email admin@example.com --apache
```

### IIS 配置（Windows）

```powershell
# 单域名
autocert install --domain api.example.com --email admin@example.com --iis

# 多域名
autocert install --domains "example.com,www.example.com,api.example.com" --email admin@example.com --iis
```

## 📁 证书文件组织

### 单域名证书
```
/etc/autocert/certs/
└── api.example.com/
    ├── cert.pem      # 证书文件
    ├── key.pem       # 私钥文件
    └── chain.pem     # 证书链文件
```

### 多域名证书
```
/etc/autocert/certs/
└── example.com_san/  # 主域名_san
    ├── cert.pem      # 多域名证书文件
    ├── key.pem       # 私钥文件
    ├── chain.pem     # 证书链文件
    ├── fullchain.pem # 完整证书链
    ├── cert.json     # 证书元数据
    └── site.json     # 托管站点配置
```

### 泛域名证书
```
/etc/autocert/certs/
└── *.example.com/    # 直接使用泛域名作为目录名
    ├── cert.pem
    ├── key.pem
    └── chain.pem
```

## 🔄 证书续期

只有基于 HTTP-01/webroot 托管的证书支持无人值守自动续期。DNS 验证证书仍可手动续期：

```bash
# 续期特定域名
autocert renew --domain api.example.com

# 续期所有证书
autocert renew

# 强制续期
autocert renew --all
```

## 💡 最佳实践

### 1. 域名选择策略

**推荐做法**：
- 如果有多个固定的子域名，使用多域名证书（SAN）
- 如果子域名数量多且经常变化，使用泛域名证书
- 小型网站建议使用单域名证书

### 2. 验证模式选择

**推荐选择**：
- 生产环境：优先使用 Webroot 模式
- 测试环境：可以使用 Standalone 模式
- 泛域名：必须使用 DNS 模式

### 3. 证书管理

**建议**：
- Webroot 站点保持续期任务开启
- DNS 验证证书预留人工续期窗口
- 定期备份证书文件
- 监控证书过期时间

## ⚠️ 注意事项

### 1. 泛域名限制
- 泛域名证书只能使用 DNS 验证
- 当前 DNS 验证需要人工完成 TXT 记录校验，因此不会加入无人值守自动续期
- 泛域名不包含主域名本身（`*.example.com` 不包含 `example.com`）
- 如需同时支持主域名和子域名，请使用混合证书

### 2. DNS 验证要求
- 需要有 DNS 管理权限
- DNS 记录传播需要时间
- 某些 DNS 服务商可能有延迟

### 3. Rate Limiting
- Let's Encrypt 有速率限制
- 同一域名每周最多申请 20 个证书
- 失败的验证也会计入限制

## 🔗 相关命令

```bash
# 查看证书状态
autocert status --domain api.example.com

# 查看所有证书
autocert status

# 导出证书
autocert export --domain api.example.com --output api-cert.tar.gz

# 导入证书
autocert import api-cert.tar.gz

# 设置定时任务
autocert schedule install
```

## 🆘 故障排除

### 常见问题

1. **DNS 验证失败**
   - 检查 DNS 记录是否正确添加
   - 等待 DNS 传播完成
   - 使用 `dig` 或 `nslookup` 验证记录

2. **泛域名验证失败**
   - 确保使用了 `--dns` 参数
   - 检查 DNS TXT 记录格式

3. **多域名证书问题**
   - 确保所有域名都指向同一服务器
   - 检查防火墙和端口配置

### 调试命令

```bash
# 启用详细日志
autocert install --domain "*.example.com" --email admin@example.com --nginx --dns --verbose

# 检查配置
autocert --help
```

---

通过以上指南，您应该能够成功为各种类型的域名申请和配置 HTTPS 证书。如有问题，请查看日志文件或联系技术支持。

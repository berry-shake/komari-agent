# 安全加固说明

- 自动更新使用独立的 HTTPS Client，强制校验证书。`--ignore-unsafe-cert` 只保留在面板请求和面板 WebSocket 连接中，不再修改全局 Transport，也不会影响 GitHub 更新连接。
- 更新仅接受 HTTPS GitHub 发行下载地址及其受信任的资源域名，拒绝降级重定向；下载上限为 128 MiB，校验文件上限为 16 KiB。校验元数据大小与 SHA-256 后才原子替换目标文件，校验失败保留旧程序。
- 仅支持本项目发布的原始二进制，移除通用自更新 SDK、归档解压及其 xz/OpenPGP 依赖。纯数字版本选择规则保持不变。
- 节点 Token 从 URL query 移至 `X-Client-Token` Header；日志统一脱敏节点 Token、自动发现 Key 和 Cloudflare Access Secret。
- 自动发现配置通过同目录临时文件原子替换，创建权限为 0600；读取旧配置时也会收紧权限。Windows 使用受保护的 DACL，仅允许当前账户、SYSTEM 和本地管理员访问。
- HTTP 响应与解压后的 WebSocket 消息设有大小上限。
- Go 构建基线为 1.26.8；CI 和发布工作流增加安全回归、竞态测试和 govulncheck。

请先部署支持 `X-Client-Token` 的新版服务端，再升级本 Agent。现有节点 UUID、Token、服务参数和纯数字版本体系保持不变。

本地检查：

```sh
go test -short ./...
go test -race -short ./server ./update ./utils ./ws
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
```

使用 `-short` 避免把现有测试中的外网 ping 检查纳入构建结果；安全回归使用本地 HTTP/HTTPS/WebSocket 测试服务和临时文件。

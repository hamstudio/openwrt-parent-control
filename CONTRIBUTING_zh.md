# 贡献指南 (Contributing Guide)

[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING_zh.md)

感谢你关注并支持 ParentControl Guard 项目！我们非常欢迎来自社区的各种贡献：问题反馈、文档完善、新功能提案、DPI 特征库补充以及代码贡献。

---

## 📋 行为准则

在所有 Issue、Pull Request 以及社区交流中，请保持相互尊重、友善且具有建设性的沟通。

---

## 🛠️ 开发环境准备

### 1. 基础依赖
- **Go**: 1.22+
- **Node.js**: 18+ (用于 Cloudflare Worker 开发)
- **Xcode**: 15+ / Swift 5.9+ (用于 iOS 与 Swift Core 开发)
- **Android Studio**: Hedgehog+ / Kotlin 1.9+ (用于 Android 客户端)

### 2. 本地测试验证
```bash
# 运行 Go 后端单元测试
go test -v ./...

# 运行 Swift Core 跨平台单元测试
cd client/ParentControlCore && swift test
```

---

## 🚀 如何参与贡献

### 报告问题 (Bug Report)
1. 在提交前先检索现有 [Issues](../../issues)，确认该问题是否已被提出。
2. 提供详尽的环境信息：OpenWrt 版本、芯片架构、路由器型号、相关运行日志（`logread -e parentcontrold`）。
3. 提供清晰可复现的步骤与预期结果。

### 补充或贡献 DPI 应用特征库
若需为 DPI 引擎新增一款受管应用特征：
1. 分析该应用的网络通信特征（域名特征、TLS SNI、IP 范围或端口特征）。
2. 在 `internal/dpi/dpi.go` 中添加对应的 App 结构体定义与规则映射。
3. 在 `internal/dpi/dpi_test.go` 中编写对应单元测试，确认加载与解析正常。

### 提交拉取请求 (Pull Request)
1. Fork 本仓库并新建分支：`git checkout -b feat/your-feature-name`。
2. 确保本地所有单元测试通过（`go test ./...` 与 `swift test`）。
3. 遵循现有的代码规范，代码内注释统一使用英文。
4. 保持单个 PR 的修改范围聚焦。
5. 提交 PR 至 `main` 分支并详细说明变更内容与验证方式。

---

## 📄 开源许可证
一旦您向 ParentControl Guard 提交贡献，即表示您同意您的代码以项目的 [MIT License](LICENSE) 许可证进行开源。

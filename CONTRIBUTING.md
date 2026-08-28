# Contributing to ParentControl Guard

[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING_zh.md)

Thank you for your interest in contributing to ParentControl Guard! We welcome contributions of all kinds: bug reports, documentation enhancements, feature proposals, new DPI signatures, and code contributions.

---

## 📋 Code of Conduct

Please be respectful, constructive, and collaborative in all issues, pull requests, and discussions.

---

## 🛠️ Development Setup

### 1. Prerequisites
- **Go**: 1.22+
- **Node.js**: 18+ (for Cloudflare Worker development)
- **Xcode**: 15+ / Swift 5.9+ (for iOS and Swift Core)
- **Android Studio**: Hedgehog+ / Kotlin 1.9+ (for Android client)

### 2. Testing Locally
```bash
# Run Go backend tests
go test -v ./...

# Run Swift Core unit tests
cd client/ParentControlCore && swift test
```

---

## 🚀 How to Contribute

### Reporting Bugs
1. Search existing [Issues](../../issues) to check if the bug has already been reported.
2. Provide detailed environment info: OpenWrt version, target architecture, hardware model, router IP/port, logs (`logread -e parentcontrold`).
3. Include clear steps to reproduce the issue.

### Suggesting or Adding DPI Application Signatures
To add a new application signature to the DPI feature database:
1. Identify the application's network characteristics (domain names, TLS SNI, IP ranges, or protocol ports).
2. Open `internal/dpi/dpi.go` and add the signature entry with its ID, category, and matching rules.
3. Add a test case in `internal/dpi/dpi_test.go` verifying that the app can be retrieved, matched, and serialized.

### Pull Requests (PRs)
1. Fork the repository and create a new feature branch: `git checkout -b feat/your-feature-name`.
2. Ensure code passes all tests (`go test ./...` and `swift test`).
3. Write clean, readable code with English comments.
4. Keep PRs focused and single-purposed.
5. Submit your PR against the `main` branch with a clear summary of changes.

---

## 📄 License
By contributing to ParentControl Guard, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).

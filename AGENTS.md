# AGENTS.md — graphify-sdk-go（本機私有，不進 GitHub）

Graphify 官方 Go SDK：Layer 2 外部 SDK，讓 Go 生態的開發者
透過 Stdio/JSON-RPC（MCP 協議）存取 Graphify 的知識圖譜能力。

## 定位與架構

- **屬於 Graphify SDK 家族**：與 `graphify-sdk-php`、`graphify-sdk-python` 平行。
- **Layer 2 外部 SDK**：不是 plugin，不實作 `PluginInterface`；
  透過 Stdio+JSON-RPC 與 `graphify-mcp` 通訊。
- **零外部依賴**：只使用 Go 標準函式庫（os/exec, encoding/json, hash/crc32, bufio）。
- **資料契約**：與 graphify-core 共用，payload 透過 Stdio/JSON-RPC 交換。

## 開發與驗證命令

- **編譯驗證**：`go build ./...`
- **語法檢查**：`go vet ./...`
- **測試**：`go test ./...`
- **依賴審計**：`go mod tidy && go mod verify`

## 目錄結構

```
graphify-sdk-golang/
├── client.go          # 公開 API — 封裝所有 24+ MCP 工具
├── transport.go       # Stdio/JSON-RPC 傳輸層
├── errors.go          # 型別化錯誤階層
├── plugin/
│   └── host.go        # 插件 SDK — JSON-RPC stdio 主機
├── types/
│   └── types.go       # 資料傳輸物件（15+ 類型）
├── docs/
│   ├── design.md      # 架構設計文件
│   ├── api-reference.md  # 完整 API 參考
│   └── plugin-sdk.md  # Plugin SDK 指南
├── AGENTS.md          # 開發者指引（本檔案）
├── go.mod             # 零外部依賴（stdlib only）
├── .gitignore
├── README.md
└── README.zh-TW.md
```

## 開發守則

- **文件優先**：變更前先更新 docs/，實作後更新 README。
- **雙語文檔維護**：英文版 `README.md` 與台灣繁體中文版 `README.zh-TW.md` 永遠保持一致。
- **秘密與敏感資訊防護**：不提交真實金鑰；使用標準環境變數或本地 gitignored 檔案。
- **開源去識別化**：對應 GitHub repo 為公開 repo，
  嚴禁在版本控制檔案中寫入本地網路拓撲、私有主機名、本地 IP、或本機絕對路徑。
- **可抽出性**：本套件未來可無痛獨立為 `graphify-sdk-go` repo。

## SDK 家族對齊

所有 Graphify SDK 必須實現相同的工具方法集（基於 `graphify-mcp` 的所有 tools），
確保跨語言 API 一致性。工具清單與 DTO 定義見 `docs/design.md`。

## 與 PHP SDK 的對應關係

| 概念 | PHP | Go |
|------|-----|-----|
| 套件名 | `graphify/sdk-php` | `github.com/cawa0505/graphify-sdk-go` |
| 命名空間 | `Graphify\Sdk` | `graphify`（package） |
| Client 類 | `GraphifyClient` | `Client` |
| Transport | `Bridge\McpTransport` | `Transport`（同 package） |
| Plugin Host | `Plugin\PluginHost` | `plugin.Host` |
| 錯誤 | `Exception\*Exception` | `*Error`（errors.go） |
| DTOs | `Dto\*` | `types.*` |
| 方法命名 | `camelCase()` | `PascalCase()` |
| 選項參數 | 建構子參數 | Functional options |
| 外部依賴 | 零（PHP built-in） | 零（Go stdlib） |
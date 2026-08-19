# Graphify Go SDK

Graphify 官方 Go SDK — 透過 MCP (Model Context Protocol) 經由 Stdio/JSON-RPC 存取知識圖譜能力。

## 文件

- **[docs/design.md](docs/design.md)** — 架構概覽、傳輸層、DTO 設計、關鍵決策
- **[docs/api-reference.md](docs/api-reference.md)** — 完整 API 參考與型別參考
- **[docs/plugin-sdk.md](docs/plugin-sdk.md)** — Plugin SDK 使用指南

## 系統需求

- Go 1.22+
- `graphify` 二進位檔需在 PATH 環境變數中（或可自訂路徑）

## 安裝

```bash
go get github.com/cawa0505/graphify-sdk-go
```

## 快速開始

```go
package main

import (
	"fmt"
	"log"

	"github.com/cawa0505/graphify-sdk-go"
)

func main() {
	client := graphify.NewClient("/path/to/your/project")
	defer client.Stop()

	// 圖譜摘要
	summary, err := client.GraphSummary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("節點: %d, 關聯: %d\n", summary.TotalNodes, summary.TotalEdges)

	// 語義記憶查詢
	result, err := client.MemoryQuery("尋找使用者認證相關")
	if err != nil {
		log.Fatal(err)
	}
	if result.IsFound() {
		for _, node := range result.Nodes {
			fmt.Printf("%s (%s) 位於 %s\n", node.Label, node.Kind, node.SourceFile)
		}
	}

	// 追蹤相依路徑
	path, err := client.TracePath(
		"src/models/user.go:struct:User",
		"src/http/handlers/auth.go:function:Login",
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("路徑:", path)

	// 查詢節點
	graph, err := client.QueryNode(
		"src/services/auth.go:struct:AuthService",
		2,
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("找到 %d 個相關節點\n", len(graph.Nodes))
}
```

## API 參考

SDK 封裝所有 24+ 個 `graphify` 工具。

### 核心圖譜

| 方法 | 說明 | 回傳 |
|--------|-------------|---------|
| `GraphSummary()` | 拓撲指標 | `*types.GraphSummary` |
| `QueryGraph(question)` | BFS 遍歷 | `*types.GraphOutput` |
| `QueryNode(nodeId, ...depth)` | 節點查詢 | `*types.GraphOutput` |
| `TracePath(from, to)` | 最短路徑 | `[]string` |
| `ReindexFile(filePath)` | 重新索引檔案 | `*types.ReindexResult` |

### 記憶與 Relay

| 方法 | 說明 | 回傳 |
|--------|-------------|---------|
| `MemoryQuery(query, ...limit)` | 語義搜尋 | `*types.MemoryQueryResult` |
| `RelayInit(projectContext, ...kind)` | 初始化交接 | `map[string]any` |
| `RelaySave(params)` | 儲存狀態 | `map[string]any` |
| `RelayClose(repo, next)` | 關閉交接 | `map[string]any` |
| `RelaySwitch(repo, ...kind)` | 切換倉庫 | `map[string]any` |
| `RelayResume(repo, ...kind)` | 恢復 | `map[string]any` |
| `RelayStatus()` | 狀態摘要 | `*types.RelayStatus` |
| `RelayAdd(file, repo)` | 匯入文件 | `map[string]any` |

### OpenDoc

| 方法 | 說明 | 回傳 |
|--------|-------------|---------|
| `OpenDocIndex(...docPaths)` | 索引規格區塊 | `map[string]any` |
| `OpenDocGetContext(symbol)` | 查詢符號文件 | `map[string]any` |
| `OpenDocAuditDrift()` | 審核漂移 | `map[string]any` |

### 審查

| 方法 | 說明 | 回傳 |
|--------|-------------|---------|
| `ReviewIngest(payload)` | 匯入審查 | `map[string]any` |
| `ReviewGetContext(node)` | 查詢審查 | `map[string]any` |
| `ReviewResolve(reviewID, reason)` | 解決審查 | `map[string]any` |
| `ReviewSearchCrg(...base)` | 搜尋 CRG | `map[string]any` |

### 遙測與覆蓋率

| 方法 | 說明 | 回傳 |
|--------|-------------|---------|
| `TelemetryIngest(source, ...path)` | 匯入指標 | `map[string]any` |
| `TelemetryGetContext(node, ...radius)` | 查詢遙測 | `map[string]any` |
| `CoverageIngest(format, data)` | 匯入覆蓋率 | `map[string]any` |
| `CoverageGetContext(node)` | 查詢覆蓋率 | `*types.CoverageResult` |
| `CoverageBlindspots()` | 低覆蓋率列表 | `map[string]any` |

### 插件閘道

| 方法 | 說明 | 回傳 |
|--------|-------------|---------|
| `PluginNotify(kind)` | 廣播更新 | `map[string]any` |

## 架構

```
Go 應用 → GraphifyClient → Transport (Stdio/JSON-RPC) → graphify (Rust)
```

- **零外部依賴**：僅使用 Go 標準函式庫
- **同步 API**：單執行緒、透過 stdio 請求-回應
- **自動工作區金鑰**：從專案路徑推導（對應 Rust crc32 邏輯）
- **延遲啟動**：首次請求時才啟動 `graphify` 子程序

## 專案結構

```
graphify-sdk-golang/
├── client.go         # 公開 API — 封裝所有 MCP 工具
├── transport.go      # Stdio/JSON-RPC 傳輸層
├── errors.go         # 型別化錯誤階層
├── plugin/
│   └── host.go       # 插件 SDK — JSON-RPC stdio 主機
├── types/
│   └── types.go      # 資料傳輸物件
├── go.mod
├── README.md
└── README.zh-TW.md
```

## 插件 SDK

`plugin` 套件提供用於在 Go 中建構 Graphify 插件的 JSON-RPC stdio 主機。詳見 [plugin/host.go](plugin/host.go)。

```go
host := plugin.NewHost()
host.RegisterTool("analyze", plugin.ToolSchema{
    Description: "分析專案結構",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "path": map[string]any{
                "type":        "string",
                "description": "專案路徑",
            },
        },
    },
}, func(args map[string]any) (map[string]any, error) {
    return map[string]any{"status": "ok"}, nil
})
host.Run()
```

## 授權

MIT
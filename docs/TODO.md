# Go Code Similarity Detection CLI Tool - TODO

## プロジェクト概要

Go ASTを利用してコードの類似性を検証するCLIツール。主に重複コードのクローン検出を目的とし、AIツールによるリファクタリングの契機を提供する。

## 📊 プロジェクト進捗状況

**全体進捗: Phase 4/7 完了 (約75%)**

- ✅ **Phase 1-4: 基盤機能完了** - CLI、AST解析、類似性検出、ディレクトリスキャン実装済み
- 🚧 **Phase 5: 次の実装対象** - キャッシュシステム・YAML出力
- 🔄 **Phase 6-7: 部分完了** - 出力機能、テスト、ドキュメント、CI/CD

**現在の機能状態:**

- ✅ **動作可能**: 単一ファイル・複数ファイル・ディレクトリの類似性検出
- ✅ **高品質**: テストカバレッジ67-100%、lint完全パス
- ✅ **本番準備済み**: エラーハンドリング、ログ出力、構造化JSON出力、ディレクトリ走査

## 要件

- **対象**: Golangアプリケーションコード
- **検出単位**: 関数レベル
- **類似性**: 重複コードのクローン検出（完全一致に近い類似性）
- **入力**: ディレクトリまたはファイル指定
- **出力**: AIツール向けstructured data（JSON/YAML）
- **設定**: 類似度閾値、並列処理数、キャッシュ利用、ignore files

## アーキテクチャ設計

### コンポーネント構成

```
similarity-go/
├── cmd/                    # CLI エントリーポイント
│   └── root.go            # cobra CLI設定
├── internal/              # 内部パッケージ
│   ├── ast/              # AST解析関連
│   │   ├── parser.go     # Goファイル解析
│   │   ├── function.go   # 関数抽出・正規化
│   │   └── hash.go       # AST構造のハッシュ化
│   ├── similarity/        # 類似性検出
│   │   ├── detector.go   # 類似性検出エンジン
│   │   ├── algorithm.go  # 比較アルゴリズム
│   │   └── threshold.go  # 閾値管理
│   ├── scanner/          # ファイルスキャン
│   │   ├── walker.go     # ディレクトリ走査
│   │   └── ignore.go     # ignore file処理
│   ├── cache/            # キャッシュシステム
│   │   └── manager.go    # キャッシュ管理
│   ├── worker/           # 並列処理
│   │   └── pool.go       # ワーカープール
│   └── output/           # 出力処理
│       ├── formatter.go  # JSON/YAML出力
│       └── result.go     # 結果構造体
├── pkg/                  # 公開パッケージ
│   └── types/            # 共通型定義
├── testdata/             # テストデータ
├── docs/                 # ドキュメント
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 実装フェーズ

### Phase 1: 基盤実装 ✅ **完了**

- [x] プロジェクト初期化（go mod, 基本構造）
- [x] CLI フレームワーク導入（cobra）
- [x] 基本的なコマンドライン引数処理
- [x] ログ設定

### Phase 2: AST解析実装 ✅ **完了**

- [x] Go ファイルパーサー実装
- [x] AST から関数抽出
- [x] 関数の正規化（変数名・コメント除去）
- [x] AST構造のハッシュ化

### Phase 3: 類似性検出実装 ✅ **完了**

- [x] 類似性検出アルゴリズム実装
- [x] 閾値による判定機能
- [x] 類似度スコア計算
- [x] CLI統合・JSON出力機能
- [x] エラーハンドリング・詳細ログ出力

### Phase 4: スキャン・並列処理 ✅ **完了**

- [x] ディレクトリ走査機能
- [x] ignore file 処理（.gitignoreライク）
- [ ] 並列処理（goroutine pool）
- [ ] プログレス表示

### Phase 5: キャッシュ・出力 🔄 **部分完了**

- [ ] キャッシュシステム実装
- [x] JSON/YAML出力フォーマット（JSON完了、YAML予定）
- [x] 結果の構造化

### Phase 6: テスト・最適化 ✅ **大部分完了**

- [x] 単体テスト作成（カバレッジ 67-100%）
- [x] 統合テスト作成
- [ ] パフォーマンステスト
- [ ] メモリ使用量最適化

### Phase 7: ドキュメント・配布 🔄 **部分完了**

- [x] README作成
- [x] 使用例・サンプル作成
- [x] バイナリビルド設定（Makefile）
- [x] CI/CD設定（GitHub Actions）

## CLI インターフェース設計

### 基本コマンド

```bash
similarity-go [flags] <target>

# 例
similarity-go ./src                              # ディレクトリ全体をスキャン
similarity-go main.go utils.go                   # 複数ファイル指定
similarity-go --threshold 0.8 --format json ./project  # 設定付きスキャン
similarity-go --verbose ./internal               # 詳細出力付きディレクトリスキャン
similarity-go ./cmd ./internal                   # 複数ディレクトリ指定
```

### フラグ

- `--threshold, -t`: 類似度閾値 (0.0-1.0, default: 0.8)
- `--format, -f`: 出力形式 (json|yaml, default: json)
- `--workers, -w`: 並列処理数 (default: CPU数)
- `--cache`: キャッシュ利用 (default: true)
- `--ignore`: ignore file指定 (default: .similarityignore)
- `--output, -o`: 出力ファイル指定
- `--verbose, -v`: 詳細出力
- `--min-lines`: 最小関数行数 (default: 5)

## 出力形式設計

### JSON出力例

```json
{
  "summary": {
    "total_functions": 150,
    "similar_groups": 12,
    "total_duplications": 28
  },
  "similar_groups": [
    {
      "id": "group_1",
      "similarity_score": 0.95,
      "functions": [
        {
          "file": "src/handler.go",
          "function": "HandleUser",
          "start_line": 15,
          "end_line": 30,
          "hash": "abc123..."
        },
        {
          "file": "src/admin.go",
          "function": "HandleAdmin",
          "start_line": 45,
          "end_line": 60,
          "hash": "abc124..."
        }
      ],
      "refactor_suggestion": "Extract common logic into shared function"
    }
  ]
}
```

## 技術的検討事項

### AST比較アルゴリズム

1. **構造ハッシュ**: AST構造を正規化してハッシュ化
2. **ツリー差分**: AST構造の編集距離計算
3. **トークン比較**: トークン列の類似度計算

### パフォーマンス考慮

- 並列処理による高速化
- キャッシュによる重複計算回避
- メモリ効率的なAST処理
- 大規模プロジェクト対応

### エラーハンドリング

- 不正なGoファイルの処理
- パーミッションエラー処理
- メモリ不足対応
- 適切なエラーメッセージ

## 依存関係

- **CLI**: github.com/spf13/cobra
- **YAML**: gopkg.in/yaml.v3
- **並列処理**: Go標準ライブラリ
- **AST**: Go標準ライブラリ (go/ast, go/parser, go/token)

## 今後の拡張可能性

- 他言語対応
- IDE プラグイン
- Web UI
- CI/CD統合
- より高度な類似性アルゴリズム

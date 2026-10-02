# Architecture Decision Records

このディレクトリには、このプロジェクトのアーキテクチャ上の重要な決定を記録した Architecture Decision Record (ADR) を置く。

## ADR を書く対象

次のいずれかに当てはまる決定を記録する。

- システムの構造に影響する（リポジトリ構成、レイヤー分割、データの持ち方など）
- 品質特性に影響する（安全性、保守性、性能、テスト容易性など）
- 後から変更するのが難しい（言語、フレームワーク、永続化方式、API 契約など）
- `docs/rules/` の開発ルールに例外を設ける、または変更する

実装の詳細や手順書は ADR に書かない。必要であれば別のドキュメントにまとめ、ADR からリンクする。

## 運用ルール

- ADR は追記専用のログとして扱う。承認済み（Accepted）の ADR の決定内容は編集しない。
- 決定を変更する場合は、新しい ADR を作成する。新しい ADR の Notes に `Supersedes` を書き、古い ADR の Status を `Superseded by ADR-NNNN` に更新する。
- 誤字やリンク切れなど、決定内容に影響しない修正は承認後でも行ってよい。
- 1 つの ADR には 1 つの決定だけを書く。段階的に進める決定は、段階ごとに別の ADR にする。
- 番号は連番で振り、欠番を再利用しない。

## ステータス

| Status | 意味 |
|---|---|
| Proposed | 提案中。まだ決定していない |
| Accepted | 決定済み。現在有効 |
| Deprecated | 有効ではなくなったが、置き換える決定はない |
| Superseded | 別の ADR に置き換えられた |

## 新しい ADR の作り方

1. `0000-template.md` をコピーし、`NNNN-kebab-case-title.md` という名前で保存する（例: `0002-repository-structure.md`）。
2. Status を `Proposed` にして内容を書き、Pull Request を作成する。
3. 決定したら Status を `Accepted` にしてマージし、下の一覧に追記する。

## 一覧

| No. | Title | Status | Date |
|---|---|---|---|
| [0001](0001-record-architecture-decisions.md) | 設計判断を ADR として記録する | Accepted | 2026-10-01 |
| [0002](0002-adopt-effect-4.md) | フロントエンドで Effect 4 を採用する | Accepted | 2026-10-02 |
| [0003](0003-double-entry-ledger.md) | お金の動きを複式の台帳として記録する | Accepted | 2026-10-02 |

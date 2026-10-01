# ADR-0001: 設計判断を ADR として記録する

## Status

Proposed

## Date

2026-10-01

## Context

このプロジェクトは、家庭内の購入申請と家計の流れを管理するサービスを題材に、関数型プログラミング（FP）の考え方を導入することでバグを少なく運用できるかを検証する実験である。結果によっては業務への導入を検討するため、何をどういう理由で決めたかを後から説明できる必要がある。

開発を進めると、リポジトリ構成、API 契約、金額の表現、お金の記録方式など、後から変更しにくい判断が数多く発生する。これらの判断の理由がコードやコミットメッセージにしか残らないと、次の問題が起きる。

- 時間が経つと、なぜその設計にしたのかを思い出せなくなり、同じ議論を繰り返す。
- 前提が変わったときに、その判断がまだ妥当かどうかを評価できない。
- 実験の振り返りで、FP の導入によって得られた効果と、個々の設計判断による効果を切り分けられない。

また、このプロジェクトでは実装は手書きで行い、AI はコードレビューにのみ使う。レビュー時に AI が設計判断の意図を把握できるよう、判断の記録はリポジトリ内のファイルとして参照できる必要がある。

## Options Considered

### Option 1: 記録しない（コードとコミットメッセージに任せる）

- 概要: 専用の記録は作らず、判断の理由はコミットメッセージや Pull Request の説明に書く。
- 利点: 追加の手間がかからない。
- 欠点: 判断が複数のコミットや PR に散らばり、一覧性がない。どの判断が現在も有効かが分からない。

### Option 2: リポジトリ外のドキュメントツールに記録する（GitHub Wiki、Notion など）

- 概要: 設計判断をリポジトリとは別の場所にまとめる。
- 利点: 編集しやすく、リッチな表現ができる。
- 欠点: コードの変更と記録の変更を同じ Pull Request でレビューできない。コードとの対応が崩れやすい。AI レビュー時に参照させるには別途取り込みが必要になる。

### Option 3: リポジトリ内に ADR として Markdown で記録する

- 概要: `docs/adr/` に 1 判断 1 ファイルの ADR を置き、Git で管理する。
- 利点: コードと同じ Pull Request でレビューでき、履歴が Git に残る。どの判断が有効かを Status で管理できる。AI レビュー時にもファイルとして参照させられる。
- 欠点: 書く手間がかかる。どの判断を ADR にするかの基準を決めて守る必要がある。

## Decision

Option 3 を採用し、アーキテクチャ上の重要な判断を `docs/adr/` に ADR として記録する。

テンプレートは、AWS Prescriptive Guidance と Microsoft Azure Well-Architected Framework が推奨する構成要素を組み合わせた `docs/adr/0000-template.md` を使う。ADR を書く対象、運用ルール、ステータスの定義は `docs/adr/README.md` に定める。

### Tradeoffs

判断の理由を後から追跡できることと、AI レビューで設計意図を参照できることを得る代わりに、判断のたびに ADR を書く手間を受け入れる。手間を抑えるため、ADR の対象は `docs/adr/README.md` の基準に当てはまるものに限定する。

### Confidence

High

ADR はリポジトリ内で軽量に運用でき、合わなければ記録をやめるだけで済むため、判断を誤った場合のコストが小さい。

## Consequences

### Positive

- 設計判断とその理由が一か所にまとまり、時間が経っても参照できる。
- 判断の変更が Supersede の関係として残り、方針がいつ・なぜ変わったかを追跡できる。
- 実験の振り返りで、各判断の確信度と結果を照らし合わせられる。
- AI レビュー時に、ADR を参照させることで設計意図に沿ったレビューを受けられる。

### Negative

- 判断のたびに ADR を書く手間がかかり、開発速度が落ちる場合がある。
- ADR を書く基準が曖昧だと、記録の漏れや過剰な記録が起きる。
- 承認済みの ADR を編集しない運用のため、ファイル数が増え続ける。

## Compliance

- Pull Request のレビュー時に、`docs/adr/README.md` の「ADR を書く対象」に当てはまる変更が含まれていないかを確認する。当てはまる場合は、対応する ADR が同じ PR か先行する PR に含まれていることを確認する。
- AI にレビューさせる際は、`docs/adr/` と `docs/rules/` を参照させる。
- 実験の振り返り時に、次の点を確認してこの判断の妥当性を評価する。
  - ADR が実際に参照され、議論の繰り返しを防げたか。
  - ADR がないまま行われた判断によって、後から混乱が起きなかったか。

## Notes

- Author: kajiya-i
- Related:
  - [Documenting Architecture Decisions - Michael Nygard](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
  - [ADR process - AWS Prescriptive Guidance](https://docs.aws.amazon.com/prescriptive-guidance/latest/architectural-decision-records/adr-process.html)
  - [Maintain an architecture decision record (ADR) - Microsoft Azure Well-Architected Framework](https://learn.microsoft.com/en-us/azure/well-architected/architect-role/architecture-decision-record)
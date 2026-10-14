# ADR-0009: API 契約を OpenAPI で書き、Go と TypeScript のコードを生成する

## Status

Accepted

## Date

2026-10-14

## Context

バックエンドは Go、フロントエンドは TypeScript（Effect 4）で書くため、型を直接共有できない。`docs/rules/backend.md` §12 と `docs/rules/frontend.md` §18 では、次の方針を定めている。

- 言語に依存しない契約を、単一の情報源にする。
- 契約の型とドメインの型は分け、境界で変換する。
- 想定されたエラーは、機械が読めるエラーのコードで返し、フロントエンドでタグ付きのエラーに変換する。
- 契約が変わったら、コンパイルかデコードの段階で壊れ、実行時に黙って誤った値にならない。

この方針を実現するために、契約をどこにどの形式で書き、Go と TypeScript のコードをどう作るかを決める必要がある。

判断の時点で、次の状況にある。

- Go の oapi-codegen が OpenAPI 3.1 に対応し、標準ライブラリの `net/http` 向けに、リクエストとレスポンスを構造体に対応づける strict server のコードを生成できる。
- Effect 4.0 と同時に、公式の `@effect/openapi-generator` が公開され、OpenAPI から Effect 4 の Schema と HTTP のクライアントを生成できる。

## Options Considered

### Option 1: OpenAPI を手で書き、Go と TypeScript の両方を生成する

- 概要: リポジトリに OpenAPI の文書を置き、Go のサーバーと TypeScript のクライアントの両方を生成する。
- 利点: 契約がバックエンドにもフロントエンドにも従属しない、言語に依存しない単一の情報源になる。Go は oapi-codegen、TypeScript は Effect の公式のツールで生成でき、どちらも広く使われている。
- 欠点: OpenAPI の YAML を手で書く手間がかかる。

### Option 2: Go のコードから OpenAPI を生成する

- 概要: Go のハンドラーや構造体の注釈から OpenAPI を生成し、そこから TypeScript を生成する。
- 利点: Go 側で書く量が少ない。
- 欠点: 契約がバックエンドの実装に従属し、言語に依存しない単一の情報源という方針とずれる。

### Option 3: Effect の HttpApi で契約を書く

- 概要: TypeScript の HttpApi で契約を書き、そこから OpenAPI を生成し、Go のサーバーを生成する。
- 利点: フロントエンドで、型安全なクライアントをそのまま使える。
- 欠点: 契約がフロントエンドに従属する。HttpApi は Effect の中で unstable なモジュールであり、契約の基盤にするには変更のリスクが大きい。

### Option 4: Protocol Buffers と Connect

- 概要: `.proto` で契約を書き、Go と TypeScript を生成する。
- 利点: 型の表現力が高く、生成の仕組みが成熟している。
- 欠点: REST ではなくなる。Effect の Schema との連携を自分で作る必要がある。

## Decision

Option 1 を採用し、API 契約を OpenAPI で書き、Go と TypeScript のコードを生成する。

- 契約は、リポジトリのルートの `api/openapi.yaml` に、OpenAPI 3.1 で書く。バックエンドとフロントエンドのどちらにも属さない共有の契約として扱う。
- Go は、oapi-codegen を `go.mod` の `tool` ディレクティブで導入し、strict server のモードで `net/http` 向けのコードを `backend/internal/infrastructure/` 以下に生成する。生成された型はドメイン層に持ち込まず、ドメインの型に変換して使う。
- TypeScript は、`@effect/openapi-generator` で Schema と HTTP のクライアントを `frontend/src/api/` 以下に生成する。生成されたクライアントが使う unstable な `effect/http` は、`tsconfig.json` で unstable な API を許可している `src/api/` の中に収める。生成された型は、必要に応じてドメインの型に変換して使う。
- 想定されたエラーは、RFC 9457 の Problem Details に、エラーのコード（`code`）を加えた形で返す。エラーのコードは契約の中で列挙し、フロントエンドでタグ付きのエラーに変換する。
- 生成したコードはコミットし、CI で生成し直して差分がないことを確かめる。

### Tradeoffs

契約を言語に依存しない単一の情報源にでき、Go と TypeScript の両方で型の食い違いを生成の段階で防げることを得る代わりに、OpenAPI の YAML を手で書く手間と、2 つの生成ツールを管理する手間を受け入れる。

### Confidence

Medium

どちらの生成ツールも主要な機能はそろっているが、`@effect/openapi-generator` は正式版が出たばかりであり、oapi-codegen の OpenAPI 3.1 への対応も新しい。最初の API（相談の一覧と作成など）を実装した時点で、生成されたコードの扱いやすさと、契約を書く手間を振り返り、この判断を見直す。

## Consequences

### Positive

- 契約の変更が、Go と TypeScript の両方のコードの生成で反映され、片方だけが古いまま残ることを防げる。
- エラーのコードが契約に列挙されるため、バックエンドとフロントエンドで、扱うエラーの種類を揃えられる。
- フロントエンドの API クライアントを、Effect の Schema と HTTP のクライアントとして手書きせずに得られる。

### Negative

- OpenAPI の YAML を手で書く必要がある。
- oapi-codegen と `@effect/openapi-generator` の 2 つの生成ツールのバージョンと設定を管理する必要がある。`@effect/openapi-generator` は、ADR-0002 に従い、`effect` と同じバージョンに揃える。
- 生成されたコードの型と、ドメインの型の変換を、バックエンドとフロントエンドの両方で書く必要がある。

## Compliance

- `api/openapi.yaml` を変更する PR では、Go と TypeScript の生成したコードも同じ PR で更新する。
- CI で、Go と TypeScript のコードを生成し直し、差分がないことを確かめる。
- ドメイン層が、生成された Go のパッケージを import していないことを、`depguard` で確かめる。
- 最初の API を実装した時点で、この ADR の Confidence を見直す。

## Notes

- Author: kajiya-i
- Related:
  - [ADR-0002](0002-adopt-effect-4.md)
  - [ADR-0006](0006-persistence-stack.md)
  - [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)
  - [@effect/openapi-generator](https://www.npmjs.com/package/@effect/openapi-generator)
  - [RFC 9457: Problem Details for HTTP APIs](https://www.rfc-editor.org/rfc/rfc9457)
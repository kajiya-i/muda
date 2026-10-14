# ADR-0008: 開発ツールを Nix の devShell で揃える

## Status

Accepted

## Date

2026-10-14

## Context

このプロジェクトの開発には、Go、Bun、golangci-lint、C コンパイラ（sqlc のビルドに必要）、PostgreSQL のクライアントなどのツールが必要である。

これまでは、Go のツール（goose、sqlc）は `go.mod` の `tool` ディレクティブで、TypeScript のツールは `package.json` で固定してきた。一方で、次のツールは開発者がそれぞれ手動でインストールしており、バージョンが揃っている保証がない。

- Go 本体と Bun 本体
- golangci-lint。CI では v2.14 に固定しているが、手元のバージョンが異なると、lint の結果が手元と CI で食い違う。
- C コンパイラ。sqlc のビルドに必要になった。

チームで開発する場合、これらを開発者がそれぞれインストールする手間と、バージョンの食い違いによる問題を減らしたい。

プロジェクトの開始時には、Nix の導入を検討したが、言語の仕組みで固定できるツールが大半であり、FP の検証とは関係のない学習のコストを増やすとして見送った。その後、言語の仕組みでは固定できないツールが増えたため、改めて判断する。

## Options Considered

### Option 1: 開発者がそれぞれインストールし、必要なバージョンを README に書く

- 概要: 今のまま、各ツールを手動でインストールする。
- 利点: 追加の仕組みが要らない。
- 欠点: インストールの手間が開発者ごとにかかる。README のバージョンと実際のバージョンが食い違っても気づけない。

### Option 2: mise などのバージョン管理ツール

- 概要: `mise.toml` などに、ツールとバージョンを書く。
- 利点: 学習のコストが低い。Go や Bun の切り替えが簡単である。
- 欠点: C コンパイラなど、言語のランタイム以外のツールは扱いにくい。

### Option 3: Dev Containers

- 概要: 開発環境そのものをコンテナとして定義し、VSCode からコンテナの中で開発する。
- 利点: Docker さえあれば環境を再現でき、VSCode との統合が強い。
- 欠点: すべての作業がコンテナの中になり、ファイルアクセスの遅さなどの問題が起きることがある。

### Option 4: Nix flakes の devShell

- 概要: `flake.nix` に開発ツールを書き、`flake.lock` で nixpkgs のリビジョンを固定する。`nix develop`（または direnv）で、すべてのツールがそろった環境に入る。
- 利点: 言語のランタイム、lint、C コンパイラ、データベースのクライアントなど、あらゆるツールを 1 か所で固定できる。ホストの上で直接動くので、エディタとの連携が素直である。同じ定義を CI でも使える。
- 欠点: Nix 自体の学習のコストがかかる。開発者のマシンに Nix のインストールが必要になる。

## Decision

Option 4 を採用し、開発ツールを Nix flakes の devShell で揃える。

- リポジトリのルートに `flake.nix` と `flake.lock` を置き、Go、Bun、golangci-lint、PostgreSQL のクライアントを devShell に含める。C コンパイラは、devShell に標準で含まれるものを使う。
- `flake.lock` をコミットし、nixpkgs のリビジョンを固定する。ツールのバージョンを上げるときは、`nix flake update` で `flake.lock` を更新する。
- Go のツールチェーンの自動ダウンロード（`GOTOOLCHAIN`）は無効にし、devShell の Go を使う。
- goose と sqlc は、引き続き `go.mod` の `tool` ディレクティブで固定する。
- PostgreSQL のサーバーは、引き続き Docker Compose で起動する。
- direnv を使えば、ディレクトリに移動するだけで devShell に入れるようにする。direnv の利用は任意とする。
- CI も、別の PR で devShell を使う形に移行し、手元と CI のツールのバージョンを揃える。

### Tradeoffs

開発者がツールを個別にインストールする手間をなくし、手元と CI のツールのバージョンを揃えられることを得る代わりに、Nix 自体の学習と、開発者のマシンへの Nix のインストールを受け入れる。

### Confidence

Medium

Nix の devShell は確立した方法であり、このプロジェクトで必要なツールはすべて nixpkgs にある。一方で、このプロジェクトの開発者が Nix に慣れていないため、日々の開発や CI への組み込みで想定外の手間が生じる可能性がある。CI への移行を終えた時点で、この判断を見直す。

## Consequences

### Positive

- リポジトリを clone し、Nix をインストールすれば、`nix develop` だけで開発に必要なツールがそろう。
- ツールのバージョンが `flake.lock` で固定され、開発者ごとの食い違いがなくなる。
- CI でも同じ devShell を使えば、手元と CI で lint などの結果が食い違わない。

### Negative

- 開発者のマシンに Nix のインストールが必要になる。
- Nix の書き方や、`flake.lock` の更新の仕方を学ぶ必要がある。
- nixpkgs のパッケージのバージョンは nixpkgs のリビジョンで決まるため、特定のバージョン（golangci-lint の特定のマイナーバージョンなど）を細かく選ぶのは難しい。
- PostgreSQL のサーバーは Docker、ツールは Nix という 2 つの仕組みを使うことになる。

## Compliance

- README の開発環境の手順を、Nix を使う形に書き換える。
- CI を devShell を使う形に移行し、手元と CI で同じバージョンの golangci-lint を使う。
- CI への移行を終えた時点で、開発者の手間やつまずいた点を振り返り、この ADR の Confidence を見直す。

## Notes

- Author: kajiya-i
- Related:
  - [ADR-0006](0006-persistence-stack.md)
  - [Nix flakes](https://nixos.wiki/wiki/Flakes)
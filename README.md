# Muda

家族が何かを買う前に家族に相談し、いいねをもらってから買うことで、無駄遣いを減らし、おうちのお金の流れを家族みんなで見えるようにするアプリケーション。

## コンセプト

- **無駄遣いを、仕組みで減らす。** 買う前に家族に相談し、いいねをもらうという流れを通すことで、衝動的な買い物を構造的に減らす。迷ったときは、無駄遣いが減る方を選ぶ。
- **家族がお互いを理解する。** 厳しさは、罰するためではなく、家族がお互いのお金の使い方を知り、話し合うきっかけにするためのものである。
- **しくみは厳しく、見た目はかわいらしく。** ルールは厳しくても、家族が続けて使えるよう、画面と言葉はやわらかくする。

## 目的

- FP を導入しバグを少なく運用できるかを検証する

## 開発環境

開発に必要なツール（Go、Bun、golangci-lint、PostgreSQL のクライアント）は、Nix の devShell でそろえる。[Nix](https://nixos.org/download/) をインストールし、flakes を有効にしたうえで、リポジトリのルートで次を実行する。

```bash
nix develop
```

[direnv](https://direnv.net/) を使う場合は、初回に `direnv allow` を実行すると、ディレクトリに移動するだけで devShell に入り、`.env` も読み込まれる。VSCode では direnv 拡張（`mkhl.direnv`）を入れると、エディタも devShell のツールを使う。

データベース（PostgreSQL）のサーバーは Docker Compose で起動する。

```bash
cp .env.example .env
docker compose up -d
```

止めるときは `docker compose down`、データも消すときは `docker compose down -v` を実行する。

データベースのマイグレーションは、`backend/` で goose を使って実行する。

```bash
cd backend
go tool goose -env ../.env up       # 適用する
go tool goose -env ../.env status   # 状態を確かめる
go tool goose -env ../.env down     # 1 つ取り消す
go tool goose -env ../.env -s create <名前> sql   # 新しいマイグレーションを作る
```

データベースにアクセスする Go のコードは、`backend/db/queries/` の SQL から sqlc で生成する。生成したコードは手で編集せず、コミットする。

```bash
cd backend
go tool sqlc generate   # コードを生成する
go tool sqlc diff       # 生成済みのコードと差分がないかを確かめる
```

## ドキュメント

- ドメインの仕様: [docs/domain](docs/domain)
- 設計判断の記録: [docs/adr](docs/adr)
- 開発ルール: [docs/rules](docs/rules)
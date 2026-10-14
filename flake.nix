{
  description = "Muda の開発環境";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go_1_27
            pkgs.bun
            pkgs.golangci-lint
            pkgs.postgresql_18 # psql などのクライアントを使う。サーバーは Docker Compose で起動する
          ];

          # go.mod の go ディレクティブによるツールチェーンの自動ダウンロードを無効にし、
          # devShell の Go を使う
          GOTOOLCHAIN = "local";
        };
      });
}
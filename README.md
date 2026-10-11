# Mecalo

[![CI](https://img.shields.io/github/actions/workflow/status/chan-mai/Mecalo/ci.yaml?branch=develop&style=flat-square&logo=github&logoColor=white&label=CI&color=B9DEFF)](https://github.com/chan-mai/Mecalo/actions/workflows/ci.yaml)
[![License](https://img.shields.io/badge/license-AGPL--3.0-B9DEFF?style=flat-square)](./LICENSE)

![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white)
![CockroachDB](https://img.shields.io/badge/CockroachDB-6933FF?style=flat-square&logo=cockroachlabs&logoColor=white)
![Nuxt](https://img.shields.io/badge/Nuxt-00DC82?style=flat-square&logo=nuxt&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?style=flat-square&logo=typescript&logoColor=white)
![TypeSpec](https://img.shields.io/badge/TypeSpec-0078D4?style=flat-square)
![Three.js](https://img.shields.io/badge/Three.js-000000?style=flat-square&logo=threedotjs&logoColor=white)

セルフホスト型の食事記録・栄養管理アプリケーション.
Mecalo = meal(memory) + calorie

- 朝食・昼食・夕食・間食ごとの記録, 過去の食事のコピー, お気に入り, 検索履歴・直近の記録からの登録
- 漢字・かなの表記揺れに対応した食品検索
- オフライン時動作をサポート
- 自作食品・レシピ
- 日本人の食事摂取基準に基づく目標量の算出と、不足・過剰の表示
- CSV / JSONエクスポートとJSON取込
- ローカルアカウントとOIDCによるログイン, 利用者ごとのデータ分離
- 広告・トラッキングなし。外部サービスへの接続は既定ですべて無効

## 構成

| ディレクトリ | 内容 |
|---|---|
| `api/` | TypeSpecによるAPI定義 |
| `backend/` | Go APIサーバ/管理コマンド |
| `frontend/` | Nuxt製フロント |
| `deploy/` | Docker Compose定義 |
| `docs/` | 設計資料/ADR等 |

設計(方針)の詳細は [docs/architecture.md](docs/architecture.md) を参照


## 設定


| 変数 | 既定値 | 説明 |
|---|---|---|
| `MECALO_DATABASE_URL` | `data/mecalo.db` | SQLiteのファイルパス, または `postgresql://` で始まる CockroachDB の接続文字列 |
| `MECALO_LISTEN` | `:8080` | 待受アドレス |
| `MECALO_TIMEZONE` | `Asia/Tokyo` | タイムゾーン |
| `MECALO_AUTH_PROVIDERS` | `local` | `local`, `oidc` をカンマ区切りで指定 |
| `MECALO_REGISTRATION` | `closed` | アカウントの作成可否 `open`で誰でもアカウント作成可能 |
| `MECALO_SECURE_COOKIE` | `true` | HTTPS環境用設定 |
| `MECALO_CORS_ORIGINS` | なし | 許可オリジン |
| `MECALO_OIDC_ISSUER` | なし | OIDCプロバイダ Issuer URI |
| `MECALO_OIDC_CLIENT_ID` | なし | Client ID(public client) |
| `MECALO_OIDC_CLIENT_SECRET` | なし | confidential clientの場合のみ |
| `MECALO_OIDC_REDIRECT_URL` | なし | `https://<host>/api/auth/oidc/callback` |
| `MECALO_OIDC_NAME` | `OpenID Connect` | ログイン画面の表示名 |
| `MECALO_OIDC_AUTO_PROVISION` | `false` | 未登録のOIDC利用者を自動作成するか |
| `MECALO_OPENFOODFACTS_LOOKUP` | `false` | 未登録バーコードをOpen Food Factsへ照会するか |


### 管理コマンド

```sh
mecalo serve
mecalo import mext [-file本表.xlsx]
mecalo import off -file openfoodfacts-products.jsonl.gz [-country japan]
MECALO_NEW_PASSWORD=... mecalo user create -login alice -admin
mecalo backup -out backup.db
mecalo export -login alice > alice.json
```

## 開発

devcontainer(`.devcontainer/`)にGo, Node.js, pnpmとCockroachDBを用意しています

```sh
make gen            # TypeSpec → OpenAPI → Go / TypeScriptの生成
make lint           # go vet, golangci-lint, ESLint(@misskey-dev/eslint-plugin), 型検査
make test           # バックエンドのテスト(SQLite)
make test-cockroach # CockroachDBでも契約テストを実行
make dev-backend    # APIサーバ
make dev-frontend   # Nuxt開発サーバ
make models         # assets/blender/foods.blendから装飾用glTFを書き出し(要Blender)
```


## データの出典

- 文部科学省「日本食品標準成分表(八訂)増補2023年」 https://www.mext.go.jp/a_menu/syokuhinseibun/mext_00001.html
  - レポジトリには同梱せず取込コマンド実行時に取得
- Open Food Facts https://world.openfoodfacts.org/ (ODbL 1.0 / DbCL 1.0)
- 厚生労働省「日本人の食事摂取基準(2025年版)」 https://www.mhlw.go.jp/stf/newpage_44138.html
  - 1歳以上の値を`backend/data/dri/2025.yaml`に転記

## ライセンス

[AGPL-3.0](LICENSE)

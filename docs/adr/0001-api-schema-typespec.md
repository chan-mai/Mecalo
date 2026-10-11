# 0001: APIスキーマをTypeSpecで管理する

## 決定

APIの定義は`api/`のTypeSpecを正とし、OpenAPI 3.0を生成する。Goはoapi-codegen(strict server), TypeScriptはopenapi-typescriptで型を生成する。

## 理由

- OpenAPIを直接書くより記述量が少なく、差分を人間が読みやすい
- 生成器の対応状況から出力はOpenAPI 3.0とする
- サーバとクライアントの型が同じ定義から作られ、手書きの型ずれが起きない

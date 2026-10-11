# 0002: DBアクセスは共通SQLと方言差分で実装する

## 決定

SQLiteとCockroachDBの両方に対応するため、`database/sql`と手書きSQLによる実装を1つ持ち、差分(プレースホルダ, 時刻の書込値, トランザクション再試行)を`sqlstore/dialect.go`に分離する。マイグレーションは方言ごとに`backend/migrations/{sqlite,postgres}`に置く。

## 理由

- sqlcは方言ごとに生成コードとadapterが2系統必要になり、同じ変更を2か所へ入れる保守負担が大きい
- ORMを使わずSQLを直接読める形にする
- 型安全性の不足は、両DBで同一テストを実行する契約テストで補う

## 補足

- 数値は精度を保つため`shopspring/decimal`で扱い、SQLiteではTEXT, CockroachDBではNUMERICで保存する
- 日本語検索は正規化キーへの部分一致で行う。CockroachDBではtrigram転置インデックスを使う

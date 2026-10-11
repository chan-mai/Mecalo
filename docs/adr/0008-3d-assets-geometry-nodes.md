# 0008: 装飾用3Dモデルはジオメトリノードで管理する

## 決定

装飾用の食材モデルは`assets/blender/foods.blend`のジオメトリノード(`GN_*`)で作成し、`make models`で`frontend/public/models/foods.glb`へ書き出す。

## 理由

- 形状をノードの数値で調整でき、生成スクリプトを読まずにBlender上で変更できる
- 書き出し結果は静的ファイルで、フロントエンドはBlenderに依存しない
- glTFが読めない場合はThree.jsの基本形状で代替表示する

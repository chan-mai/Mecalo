namespace Mecalo;

enum FoodKind {
  @doc("日本食品標準成分表")
  standard,

  @doc("バーコード付き市販品")
  product,

  custom,
  recipe,
}

model FoodSource {
  datasetId: string;

  @doc("食品番号, バーコード等")
  reference: string;

  @doc("取得元ページ, 空は取込元全体")
  url: string;

  @doc("取得日時")
  retrievedAt?: utcDateTime;
}

@doc("栄養値の基準量, 省略は可食部100g")
model Basis {
  @doc("1個, 1食等")
  @minLength(1)
  @maxLength(30)
  unit: string;

  @doc("1単位の重量(g), 省略は不明")
  @minValueExclusive(0)
  grams?: float64;
}

model Dataset {
  id: string;
  kind: string;
  name: string;
  version: string;
  license: string;
  url: string;
  attribution: string;
  importedAt: utcDateTime;
}

model FoodRevision {
  id: string;
  number: int32;
  name: string;
  brand: string;

  @doc("検索対象の別名")
  aliases: string[];

  @doc("可食部100gあたり, basis指定時はその1単位あたり")
  nutrients: NutrientProfile;

  basis?: Basis;
  portions: Portion[];
  source: FoodSource;
  createdAt: utcDateTime;
}

model Food {
  id: string;
  kind: FoodKind;
  barcode: string;

  @doc("所有し編集可能")
  editable: boolean;

  favorite: boolean;
  deleted: boolean;

  @doc("取得元の一覧から消えた食品, 検索対象外")
  discontinued: boolean;

  revision: FoodRevision;
}

model FoodInput {
  @minLength(1)
  @maxLength(200)
  name: string;

  brand?: string;

  @doc("検索対象の別名")
  @maxItems(20)
  aliases?: string[];

  barcode?: string;
  nutrients: Record<NutrientInput>;
  basis?: Basis;
  portions?: Portion[];
}

model FoodSelection {
  @doc("食品を選ぶ前に入力した検索語")
  @minLength(1)
  @maxLength(100)
  query: string;
}

model IngredientInput {
  foodId: string;
  quantity: Quantity;
}

model RecipeInput {
  @minLength(1)
  @maxLength(200)
  name: string;

  @minItems(1)
  ingredients: IngredientInput[];

  @doc("完成重量(g), 省略時は材料重量の合計")
  cookedGrams?: float64;

  @doc("何人前, 省略時は人前単位なし")
  servings?: float64;
}

model RecipeLine {
  foodId: string;
  revisionId: string;
  name: string;
  quantity: Quantity;
  grams: float64;
}

model Recipe {
  food: Food;
  lines: RecipeLine[];
  rawGrams: float64;
  cookedGrams: float64;
  servings: float64;
}

@doc("外部取得元の検索結果1件, 未保存")
model ExternalFood {
  datasetId: string;

  @doc("取得元の表示名")
  datasetName: string;

  reference: string;
  name: string;
  brand: string;
  nutrients: NutrientProfile;
  basis?: Basis;
  portions: Portion[];

  @doc("保存済みの場合の食品ID")
  savedId?: string;
}

model ExternalSearchResult {
  foods: ExternalFood[];

  @doc("取得できなかった取得元")
  failed: string[];

  @doc("結果に含まれる取得元の帰属表示")
  attributions: string[];
}

model ExternalFoodSelection {
  datasetId: string;
  reference: string;
}

enum SourceKind {
  @doc("バーコード照会")
  lookup,

  @doc("名前での外部検索")
  finder,

  @doc("商品一覧の定期同期")
  catalog,
}

model Source {
  dataset: Dataset;
  kind: SourceKind;
  syncedAt?: utcDateTime;

  @doc("直近の同期の失敗内容, 空は成功")
  error: string;

  @doc("直近の同期で確認した商品数")
  items: int32;

  @doc("同期中")
  running: boolean;
}

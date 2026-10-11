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

  @doc("可食部100gあたり")
  nutrients: NutrientProfile;

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

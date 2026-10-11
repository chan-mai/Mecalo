namespace Mecalo;

enum Sex {
  male,
  female,
}

enum ActivityLevel {
  low,
  moderate,
  high,
}

enum Goal {
  lose,
  maintain,
  gain,
}

model Profile {
  sex: Sex;
  birthDate: plainDate;

  @minValueExclusive(0)
  heightCm: float64;

  activity: ActivityLevel;
  goal: Goal;

  @doc("目的に応じたエネルギー増減(kcal/日)")
  energyAdjustKcal: float64;

  @doc("鉄の基準値選択用")
  menstruating: boolean;
}

model TargetOverride {
  code: string;
  min?: float64;
  max?: float64;
}

model Settings {
  effectiveFrom: plainDate;
  profile: Profile;
  overrides: TargetOverride[];
}

model SettingsInput {
  @doc("省略時は当日")
  effectiveFrom?: plainDate;

  profile: Profile;
  overrides?: TargetOverride[];
}

model Weight {
  date: plainDate;
  kg: float64;
  bodyFatPct?: float64;
}

model WeightInput {
  @minValueExclusive(0)
  kg: float64;

  @minValue(0)
  @maxValue(100)
  bodyFatPct?: float64;
}

model ReferenceInfo {
  edition: string;
  title: string;
  url: string;
}

model ImportResult {
  foods: int32;
  meals: int32;
  weights: int32;
  settings: int32;
  skipped: int32;
}

@doc("レシピの材料構成")
model RecipeComposition {
  lines: RecipeLine[];
  rawGrams: float64;
  cookedGrams: float64;
  servings: float64;
}

model ExportFood {
  id: string;
  kind: FoodKind;
  barcode: string;
  createdAt: utcDateTime;
  deletedAt?: utcDateTime;

  @doc("番号順の全リビジョン")
  revisions: FoodRevision[];

  @doc("recipe種別のみ, キーはリビジョンID")
  recipes?: Record<RecipeComposition>;
}

@doc("1利用者分の全データ")
model ExportBundle {
  format: "mecalo-export";
  version: 1;
  exportedAt: utcDateTime;
  user: User;
  settings: Settings[];
  weights: Weight[];
  foods: ExportFood[];
  favorites: string[];
  meals: MealEntry[];
}

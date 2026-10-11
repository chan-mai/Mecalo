namespace Mecalo;

enum MealSlot {
  breakfast,
  lunch,
  dinner,
  snack,
}

@doc("記録時点の食品情報と栄養値を保持")
model MealEntry {
  id: string;
  date: plainDate;
  slot: MealSlot;
  foodId: string;
  foodRevisionId: string;
  foodName: string;
  brand: string;
  source: FoodSource;
  quantity: Quantity;
  grams: float64;

  @doc("grams分の栄養値")
  nutrients: NutrientProfile;

  clientRef: string;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

model AddMealEntry {
  date: plainDate;
  slot: MealSlot;
  foodId: string;
  quantity: Quantity;

  @doc("再送時の重複防止キー")
  @maxLength(64)
  clientRef?: string;
}

model UpdateMealEntry {
  date?: plainDate;
  slot?: MealSlot;
  quantity?: Quantity;
}

model CopyMeals {
  fromDate: plainDate;

  @doc("省略時は全区分")
  fromSlot?: MealSlot;

  toDate: plainDate;

  @doc("省略時は元と同じ区分")
  toSlot?: MealSlot;
}

model QuickPick {
  food: Food;
  quantity: Quantity;
}

model QuickPicks {
  recent: QuickPick[];
  frequent: QuickPick[];
  favorites: QuickPick[];
  searches: string[];
}

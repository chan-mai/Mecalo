namespace Mecalo;

enum Level {
  deficient,
  low,
  adequate,
  high,
  excessive,

  @doc("未計測・充足率不足")
  `unknown`,

  @doc("基準値なし")
  `none`,
}

model Indicators {
  ear?: float64;
  rda?: float64;
  ai?: float64;
  ul?: float64;
  dgMin?: float64;
  dgMax?: float64;
}

model NutrientTarget {
  code: string;
  ...Indicators;

  @doc("エネルギーの目標値")
  goal?: float64;

  customMin?: float64;
  customMax?: float64;
}

model Targets {
  edition: string;
  ageBand: string;
  energyKcal: float64;
  nutrients: NutrientTarget[];
}

model Evaluation {
  code: string;
  level: Level;
  value: NutrientValue;
  lower?: float64;
  upper?: float64;
}

model DaySummary {
  date: plainDate;
  entryCount: int32;
  total: NutrientProfile;
  bySlot: Record<NutrientProfile>;
}

model DailyReport {
  summary: DaySummary;

  @doc("身体情報・体重が未登録なら省略")
  targets?: Targets;

  evaluations: Evaluation[];
}

@doc("ホーム画面用の1日分")
model DayView {
  date: plainDate;
  entries: MealEntry[];
  report: DailyReport;

  @doc("当日以前で最新の体重")
  latestWeight?: Weight;
}

model RangeReport {
  from: plainDate;
  to: plainDate;
  days: DailyReport[];
  recordedDays: int32;
  average: NutrientProfile;
  averageEvaluations: Evaluation[];
}

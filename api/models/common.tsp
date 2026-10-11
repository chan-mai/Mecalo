import "@typespec/http";

using Http;

namespace Mecalo;

@doc("RFC 9457 Problem Details")
model Problem {
  type?: string;
  title: string;
  status: int32;
  detail?: string;
}

model ProblemResponse<Status> {
  @statusCode statusCode: Status;
  @header contentType: "application/problem+json";
  @body problem: Problem;
}

@error model BadRequest is ProblemResponse<400>;
@error model Unauthorized is ProblemResponse<401>;
@error model Forbidden is ProblemResponse<403>;
@error model NotFound is ProblemResponse<404>;
@error model Conflict is ProblemResponse<409>;

alias CommonErrors = BadRequest | Unauthorized | Forbidden | NotFound;

@doc("栄養値の状態")
enum NutrientStatus {
  measured,
  estimated,

  @doc("微量, 集計上は0")
  trace,

  @doc("未計測, 0と区別")
  missing,

  @doc("複数の値の合算")
  calculated,
}

model NutrientValue {
  amount: float64;
  status: NutrientStatus;

  @doc("元の重量のうち値が判明している割合(0〜1)")
  coverage: float64;
}

@doc("キーは栄養素コード, キーなしは未計測")
model NutrientProfile is Record<NutrientValue>;

@doc("入力用の栄養値, amount省略は未計測")
model NutrientInput {
  amount?: float64;
  status?: "measured" | "estimated" | "trace" | "missing";
}

model Quantity {
  @minValueExclusive(0)
  amount: float64;

  @doc("\"g\" または食品の単位名")
  unit: string;
}

model Portion {
  unit: string;

  @minValueExclusive(0)
  grams: float64;
}

enum NutrientUnit {
  kcal,
  g,
  mg,
  ug,
}

enum NutrientGroup {
  energy,
  macro,
  mineral,
  vitamin,
}

model NutrientDefinition {
  code: string;
  unit: NutrientUnit;
  group: NutrientGroup;
}

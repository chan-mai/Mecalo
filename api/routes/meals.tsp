import "@typespec/http";

using Http;

namespace Mecalo;

@route("/days/{date}")
@tag("meals")
namespace DaysApi {
  @doc("記録・集計・目標評価・体重をまとめてレスポンス")
  @get
  op get(@path date: plainDate): DayView | CommonErrors;
}

@route("/meals")
@tag("meals")
namespace MealsApi {
  @doc("clientRefが既存なら既存の記録をレスポンス")
  @post
  op add(@body body: AddMealEntry): MealEntry | CommonErrors;

  @doc("記録画面の候補")
  @get
  @route("/quick-picks")
  op quickPicks(): QuickPicks | CommonErrors;

  @doc("食品の現行データで再計算, 削除済み食品は元の値を複製")
  @post
  @route("/copy")
  op copy(@body body: CopyMeals): MealEntry[] | CommonErrors;

  @doc("数量変更時は記録時の食品データで再計算")
  @patch(#{ implicitOptionality: false })
  @route("/{id}")
  op update(@path id: string, @body body: UpdateMealEntry): MealEntry | CommonErrors;

  @delete
  @route("/{id}")
  op delete(@path id: string): NoContentResponse | CommonErrors;

  @doc("食品の現行データで栄養値を再計算")
  @post
  @route("/{id}/recalculate")
  op recalculate(@path id: string): MealEntry | CommonErrors;
}

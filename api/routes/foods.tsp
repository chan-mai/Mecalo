import "@typespec/http";

using Http;

namespace Mecalo;

@route("/foods")
@tag("foods")
namespace FoodsApi {
  @doc("検索履歴・記録頻度・お気に入りを加味した順でレスポンス")
  @get
  op search(
    @query q: string,
    @query kind?: FoodKind[],
    @query limit?: int32 = 30,
    @query offset?: int32 = 0,
  ): Food[] | CommonErrors;

  @post op create(@body body: FoodInput): Food | CommonErrors;

  @doc("自分が登録した食品")
  @get
  @route("/mine")
  op listMine(): Food[] | CommonErrors;

  @doc("外部取得元の名前検索, 利用者の操作時のみ")
  @get
  @route("/external")
  op searchExternal(@query q: string): ExternalSearchResult | CommonErrors;

  @doc("外部検索の結果をシステム食品として保存")
  @post
  @route("/external")
  op saveExternal(@body body: ExternalFoodSelection): Food | CommonErrors;

  @doc("ローカルDB優先, 未登録時は外部照会が有効な場合のみ照会")
  @get
  @route("/barcode")
  op getByBarcode(@query code: string): Food | CommonErrors;

  @get
  @route("/{id}")
  op get(@path id: string): Food | CommonErrors;

  @doc("新リビジョンとして保存, 過去の記録は不変")
  @put
  @route("/{id}")
  op update(@path id: string, @body body: FoodInput): Food | CommonErrors;

  @delete
  @route("/{id}")
  op delete(@path id: string): NoContentResponse | CommonErrors;

  @get
  @route("/{id}/revisions")
  op listRevisions(@path id: string): FoodRevision[] | CommonErrors;

  @doc("検索結果から選んだ食品を検索語と対応付け, 次回から上位に表示")
  @post
  @route("/{id}/selections")
  op recordSelection(@path id: string, @body body: FoodSelection): NoContentResponse | CommonErrors;

  @put
  @route("/{id}/favorite")
  op addFavorite(@path id: string): NoContentResponse | CommonErrors;

  @delete
  @route("/{id}/favorite")
  op removeFavorite(@path id: string): NoContentResponse | CommonErrors;
}

@route("/favorites")
@tag("foods")
namespace FavoritesApi {
  @get op list(): Food[] | CommonErrors;
}

@route("/search-history")
@tag("foods")
namespace SearchHistoryApi {
  @get op list(): string[] | CommonErrors;
  @delete op clear(): NoContentResponse | CommonErrors;
}

@route("/recipes")
@tag("recipes")
namespace RecipesApi {
  @post op create(@body body: RecipeInput): Recipe | CommonErrors;

  @get
  @route("/{id}")
  op get(@path id: string): Recipe | CommonErrors;

  @doc("材料の現行データで再計算し新リビジョンとして保存")
  @put
  @route("/{id}")
  op update(@path id: string, @body body: RecipeInput): Recipe | CommonErrors;
}

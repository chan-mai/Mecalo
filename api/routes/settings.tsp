import "@typespec/http";

using Http;

namespace Mecalo;

@route("/settings")
@tag("settings")
namespace SettingsApi {
  @doc("当日時点の設定")
  @get
  op get(): Settings | CommonErrors;

  @doc("同一の適用開始日は上書き")
  @put
  op save(@body body: SettingsInput): Settings | CommonErrors;

  @get
  @route("/history")
  op history(): Settings[] | CommonErrors;
}

@route("/targets/{date}")
@tag("settings")
namespace TargetsApi {
  @get op get(@path date: plainDate): Targets | CommonErrors;
}

@route("/weights")
@tag("weights")
namespace WeightsApi {
  @get op list(@query from: plainDate, @query to: plainDate): Weight[] | CommonErrors;

  @put
  @route("/{date}")
  op save(@path date: plainDate, @body body: WeightInput): Weight | CommonErrors;

  @delete
  @route("/{date}")
  op delete(@path date: plainDate): NoContentResponse | CommonErrors;
}

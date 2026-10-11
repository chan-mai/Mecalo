import "@typespec/http";

using Http;

namespace Mecalo;

@route("/reports")
@tag("reports")
namespace ReportsApi {
  @get
  @route("/daily/{date}")
  op daily(@path date: plainDate): DailyReport | CommonErrors;

  @doc("最大366日")
  @get
  @route("/range")
  op range(@query from: plainDate, @query to: plainDate): RangeReport | CommonErrors;
}

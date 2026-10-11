import "@typespec/http";

using Http;

namespace Mecalo;

@route("/reference")
@tag("reference")
namespace ReferenceApi {
  @get
  @route("/nutrients")
  op nutrients(): NutrientDefinition[] | CommonErrors;

  @get
  @route("/datasets")
  op datasets(): Dataset[] | CommonErrors;

  @doc("目標算出に使う食事摂取基準")
  @get
  @route("/dri")
  op dietaryReference(): ReferenceInfo | CommonErrors;
}

@route("/export")
@tag("data")
namespace ExportApi {
  @doc("全データ, 取込にそのまま利用可能")
  @get
  @route("/json")
  op json(): {
    @header contentDisposition: string;
    @body body: ExportBundle;
  } | CommonErrors;

  @get
  @route("/meals.csv")
  op mealsCsv(): {
    @header contentType: "text/csv";
    @header contentDisposition: string;
    @body body: bytes;
  } | CommonErrors;

  @get
  @route("/weights.csv")
  op weightsCsv(): {
    @header contentType: "text/csv";
    @header contentDisposition: string;
    @body body: bytes;
  } | CommonErrors;
}

@route("/import")
@tag("data")
namespace ImportApi {
  @doc("exportJsonの出力を取込, 既存IDは読み飛ばし")
  @post
  op json(@body body: ExportBundle): ImportResult | CommonErrors;
}

@route("/health")
@tag("system")
namespace HealthApi {
  @get
  @useAuth(NoAuth)
  op get(): {
    status: "ok";
  };
}

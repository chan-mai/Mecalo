import "@typespec/http";
import "@typespec/openapi";
import "./models/common.tsp";
import "./models/auth.tsp";
import "./models/foods.tsp";
import "./models/meals.tsp";
import "./models/reports.tsp";
import "./models/settings.tsp";
import "./routes/auth.tsp";
import "./routes/foods.tsp";
import "./routes/meals.tsp";
import "./routes/reports.tsp";
import "./routes/settings.tsp";
import "./routes/data.tsp";

using Http;
using OpenAPI;

@service(#{ title: "Mecalo API" })
@info(#{ version: "0.1.0", license: #{ name: "AGPL-3.0-only" } })
@server("/api", "Same origin")
@useAuth(SessionCookie)
namespace Mecalo;

@doc("ログイン時に発行するセッションCookie")
model SessionCookie is ApiKeyAuth<ApiKeyLocation.cookie, "mecalo_session">;

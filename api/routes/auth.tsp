import "@typespec/http";

using Http;

namespace Mecalo;

@route("/auth")
@tag("auth")
namespace AuthApi {
  @get
  @route("/config")
  @useAuth(NoAuth)
  op config(): AuthConfig;

  @doc("利用者0件時のみ, セットアップトークン一致時に管理者を作成しログイン")
  @post
  @route("/setup")
  @useAuth(NoAuth)
  op setup(@body body: SetupRequest): User | CommonErrors | Conflict;

  @post
  @route("/signup")
  @useAuth(NoAuth)
  op signUp(@body body: SignUpRequest): User | CommonErrors | Conflict;

  @post
  @route("/login")
  @useAuth(NoAuth)
  op login(@body body: Credentials): User | CommonErrors;

  @post
  @route("/logout")
  op logout(): NoContentResponse;

  @doc("OIDC認可リクエストへリダイレクト")
  @get
  @route("/oidc/start")
  @useAuth(NoAuth)
  op oidcStart(@query returnTo?: string): {
    @statusCode statusCode: 302;
    @header location: string;
  } | NotFound;

  @get
  @route("/oidc/callback")
  @useAuth(NoAuth)
  op oidcCallback(@query code?: string, @query state?: string, @query error?: string): {
    @statusCode statusCode: 302;
    @header location: string;
  } | BadRequest;
}

@route("/me")
@tag("auth")
namespace MeApi {
  @get op get(): User | Unauthorized;

  @doc("表示名・ログイン名の変更")
  @patch(#{ implicitOptionality: false })
  op update(@body body: ProfileUpdate): User | CommonErrors | Conflict;

  @put
  @route("/password")
  op changePassword(@body body: PasswordChange): NoContentResponse | CommonErrors;
}

@route("/admin/users")
@tag("admin")
namespace AdminApi {
  @get op list(): User[] | CommonErrors;
  @post op create(@body body: CreateUserRequest): User | CommonErrors | Conflict;
  @get
  @route("/{id}")
  op get(@path id: string): User | CommonErrors;

  @doc("自身の管理者権限は外せない")
  @patch(#{ implicitOptionality: false })
  @route("/{id}")
  op update(@path id: string, @body body: UserUpdate): User | CommonErrors | Conflict;

  @delete op delete(@path id: string): NoContentResponse | CommonErrors;
}

@route("/admin/sources")
@tag("admin")
namespace AdminSourcesApi {
  @doc("有効な外部取得元と同期状態")
  @get
  op list(): Source[] | CommonErrors;

  @doc("同期中でない全取得元の商品一覧の同期を順に開始")
  @post
  @route("/sync")
  op syncAll(): AcceptedResponse | CommonErrors | Conflict;

  @doc("商品一覧の同期を開始")
  @post
  @route("/{id}/sync")
  op sync(@path id: string): AcceptedResponse | CommonErrors | Conflict;
}

@route("/admin/settings")
@tag("admin")
namespace AdminSettingsApi {
  @doc("設定の一覧と値の出どころ")
  @get
  op list(): InstanceSetting[] | CommonErrors;

  @doc("設定の変更, 環境変数・既定値と同じ値はWebUIでの設定を解除")
  @put
  op update(@body body: InstanceSettingsUpdate): InstanceSetting[] | CommonErrors;

  @doc("全項目を環境変数・既定値へ戻す")
  @delete
  op resetAll(): InstanceSetting[] | CommonErrors;

  @doc("1項目を環境変数・既定値へ戻す")
  @delete
  @route("/{key}")
  op reset(@path key: string): InstanceSetting[] | CommonErrors;
}

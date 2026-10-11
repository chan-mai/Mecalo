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

  @doc("利用者0件時のみ管理者を作成しログイン")
  @post
  @route("/setup")
  @useAuth(NoAuth)
  op setup(@body body: SignUpRequest): User | CommonErrors | Conflict;

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

  @put
  @route("/password")
  op changePassword(@body body: PasswordChange): NoContentResponse | CommonErrors;
}

@route("/admin/users")
@tag("admin")
namespace AdminApi {
  @get op list(): User[] | CommonErrors;
  @post op create(@body body: CreateUserRequest): User | CommonErrors | Conflict;
  @delete op delete(@path id: string): NoContentResponse | CommonErrors;
}

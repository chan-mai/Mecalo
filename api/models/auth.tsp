namespace Mecalo;

enum Role {
  admin,
  member,
}

model User {
  id: string;
  displayName: string;
  role: Role;
  createdAt: utcDateTime;
}

model AuthConfig {
  @doc("利用者0件で初回セットアップが必要")
  setupRequired: boolean;

  registrationOpen: boolean;
  localEnabled: boolean;
  oidcEnabled: boolean;

  @doc("ログイン画面に表示するOIDCプロバイダ名")
  oidcName?: string;
}

model Credentials {
  @pattern("^[a-zA-Z0-9_.-]{3,32}$")
  loginName: string;

  @minLength(8)
  password: string;
}

model SignUpRequest {
  ...Credentials;
  displayName?: string;
}

model SetupRequest {
  ...SignUpRequest;

  @doc("MECALO_SETUP_TOKENの値, 未指定時は起動時のログに出力")
  setupToken: string;
}

model PasswordChange {
  current: string;

  @minLength(8)
  next: string;
}

model CreateUserRequest {
  ...SignUpRequest;
  role: Role;
}

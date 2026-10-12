namespace Mecalo;

enum Role {
  admin,
  member,
}

model User {
  id: string;
  displayName: string;

  @doc("ローカルのログイン名, OIDCのみの利用者は省略")
  loginName?: string;

  role: Role;
  createdAt: utcDateTime;

  @doc("初回セットアップ完了済み")
  onboarded: boolean;
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

@doc("省略した項目は変更なし")
model ProfileUpdate {
  @maxLength(64)
  displayName?: string;

  @pattern("^[a-zA-Z0-9_.-]{3,32}$")
  loginName?: string;
}

@doc("省略した項目は変更なし")
model UserUpdate {
  ...ProfileUpdate;
  role?: Role;

  @doc("現在のパスワードなしで再設定, 対象のセッションは破棄")
  @minLength(8)
  password?: string;
}

model CreateUserRequest {
  ...SignUpRequest;
  role: Role;
}

enum SettingKind {
  text,
  secret,
  bool,
  duration,
  @doc("optionsから1つ")
  choice,
  @doc("optionsから複数, カンマ区切り")
  choices,
  @doc("任意の値のカンマ区切り")
  list,
}

enum SettingOrigin {
  default,
  env,
  ui,
}

@doc("WebUIで変更できる設定")
model InstanceSetting {
  @doc("環境変数名")
  key: string;

  group: string;
  kind: SettingKind;
  options: string[];

  @doc("現在の値, 秘密の値は空")
  value: string;

  defaultValue: string;

  @doc("環境変数の値, 秘密の値は空")
  envValue: string;

  hasEnv: boolean;

  @doc("値の有無")
  configured: boolean;

  origin: SettingOrigin;
}

model InstanceSettingsUpdate {
  @doc("変更する項目のみ, キーは環境変数名")
  values: Record<string>;
}

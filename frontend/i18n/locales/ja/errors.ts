// エラーと入力検証
export default {
	generic: 'エラーが発生しました',
	auth: {
		failed: 'ログイン名またはパスワードが違います',
		oidc: '外部ログインに失敗しました',
		notRegistered: 'このアカウントは登録されていません',
		setupToken: '認証に失敗しました。セットアップトークンが一致しません',
		loginNameTaken: 'このログイン名は使用されています',
		setupDone: '初期設定は完了しています。ログインしてください',
	},
	validation: {
		required: '入力してください',
		loginName: '英数字と _ . - の3〜32文字で入力してください',
		passwordLength: '8文字以上で入力してください',
		positive: '0より大きい値を入力してください',
		number: '数値を入力してください',
		date: '日付を入力してください',
		range: '下限は上限以下にしてください',
		maxLength: '長すぎます',
	},
	offlineUnavailable: 'この操作はオンライン時のみ利用できます',
};

// ログイン・アカウント作成
import type ja from '../../ja/pages/auth';

export default {
	auth: {
		login: 'Log in',
		signup: 'Create account',
		setup: 'Initial setup',
		setupHint: 'Create the first administrator account',
		setupToken: 'Setup token',
		setupTokenHint: 'Enter the value of the MECALO_SETUP_TOKEN environment variable. (*If it is not set, it is printed in the startup log)',
		loginWith: 'Log in with {name}',
		noAccount: 'Don\'t have an account?',
	},
} satisfies typeof ja;

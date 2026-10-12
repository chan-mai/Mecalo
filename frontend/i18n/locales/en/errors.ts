// エラーと入力検証
import type ja from '../ja/errors';

export default {
	generic: 'Something went wrong',
	auth: {
		failed: 'Incorrect login name or password',
		oidc: 'External login failed',
		notRegistered: 'This account is not registered',
		setupToken: 'Authentication failed. The setup token does not match',
		setupDone: 'Initial setup is already complete. Log in instead',
	},
	validation: {
		required: 'Required',
		loginName: 'Use 3-32 characters of letters, digits, _ . -',
		passwordLength: 'Use at least 8 characters',
		positive: 'Enter a value greater than 0',
		number: 'Enter a number',
		date: 'Enter a date',
		range: 'Min must not exceed max',
		maxLength: 'Too long',
	},
	offlineUnavailable: 'This action is only available online',
} satisfies typeof ja;

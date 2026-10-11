import * as v from 'valibot';
import type { Schemas } from '~/composables/api/client';

export const credentialsSchema = v.object({
	loginName: v.pipe(v.string(), v.regex(/^[a-zA-Z0-9_.-]{3,32}$/, 'errors.validation.loginName')),
	password: v.pipe(v.string(), v.minLength(8, 'errors.validation.passwordLength')),
}) satisfies v.GenericSchema<unknown, Schemas['Credentials']>;

export const signUpSchema = v.object({
	...credentialsSchema.entries,
	displayName: v.optional(v.pipe(v.string(), v.trim(), v.maxLength(64, 'errors.validation.maxLength'))),
}) satisfies v.GenericSchema<unknown, Schemas['SignUpRequest']>;

export const passwordChangeSchema = v.object({
	current: v.pipe(v.string(), v.minLength(1, 'errors.validation.required')),
	next: v.pipe(v.string(), v.minLength(8, 'errors.validation.passwordLength')),
}) satisfies v.GenericSchema<unknown, Schemas['PasswordChange']>;

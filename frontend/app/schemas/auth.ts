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

export const setupSchema = v.object({
	...signUpSchema.entries,
	setupToken: v.pipe(v.string(), v.trim(), v.minLength(1, 'errors.validation.required')),
}) satisfies v.GenericSchema<unknown, Schemas['SetupRequest']>;

const loginName = v.pipe(v.string(), v.trim(), v.regex(/^[a-zA-Z0-9_.-]{3,32}$/, 'errors.validation.loginName'));
const displayName = v.pipe(v.string(), v.trim(), v.maxLength(64, 'errors.validation.maxLength'));

export const profileUpdateSchema = v.object({
	displayName: v.optional(displayName),
	loginName: v.optional(loginName),
}) satisfies v.GenericSchema<unknown, Schemas['ProfileUpdate']>;

export const passwordResetSchema = v.object({
	password: v.pipe(v.string(), v.minLength(8, 'errors.validation.passwordLength')),
});

export const passwordChangeSchema = v.object({
	current: v.pipe(v.string(), v.minLength(1, 'errors.validation.required')),
	next: v.pipe(v.string(), v.minLength(8, 'errors.validation.passwordLength')),
}) satisfies v.GenericSchema<unknown, Schemas['PasswordChange']>;

export const tokenSchema = v.object({
	name: v.pipe(v.string(), v.trim(), v.minLength(1, 'errors.validation.required'), v.maxLength(64, 'errors.validation.maxLength')),
	scope: v.picklist(['read', 'write']),
}) satisfies v.GenericSchema<unknown, Schemas['TokenInput']>;

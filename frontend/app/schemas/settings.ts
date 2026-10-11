import * as v from 'valibot';
import type { Schemas } from '~/composables/api/client';

const isoDate = v.pipe(v.string(), v.isoDate('errors.validation.date'));
const optionalNumber = v.optional(v.pipe(v.number('errors.validation.number'), v.minValue(0, 'errors.validation.positive')));

export const profileSchema = v.object({
	sex: v.picklist(['male', 'female']),
	birthDate: isoDate,
	heightCm: v.pipe(v.number('errors.validation.number'), v.gtValue(0, 'errors.validation.positive')),
	activity: v.picklist(['low', 'moderate', 'high']),
	goal: v.picklist(['lose', 'maintain', 'gain']),
	energyAdjustKcal: v.number('errors.validation.number'),
	menstruating: v.boolean(),
}) satisfies v.GenericSchema<unknown, Schemas['Profile']>;

export const overrideSchema = v.pipe(
	v.object({ code: v.string(), min: optionalNumber, max: optionalNumber }),
	v.check((o) => o.min === undefined || o.max === undefined || o.min <= o.max, 'errors.validation.range'),
) satisfies v.GenericSchema<unknown, Schemas['TargetOverride']>;

export const settingsSchema = v.object({
	effectiveFrom: v.optional(isoDate),
	profile: profileSchema,
	overrides: v.optional(v.array(overrideSchema)),
}) satisfies v.GenericSchema<unknown, Schemas['SettingsInput']>;

export const weightSchema = v.object({
	kg: v.pipe(v.number('errors.validation.number'), v.gtValue(0, 'errors.validation.positive')),
	bodyFatPct: v.optional(v.pipe(v.number('errors.validation.number'), v.minValue(0), v.maxValue(100))),
}) satisfies v.GenericSchema<unknown, Schemas['WeightInput']>;

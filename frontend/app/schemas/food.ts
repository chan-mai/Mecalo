import * as v from 'valibot';
import { isoDate } from './common';
import type { Schemas } from '~/composables/api/client';

export const portionSchema = v.object({
	unit: v.pipe(v.string(), v.trim(), v.minLength(1, 'errors.validation.required'), v.notValue('g', 'errors.validation.required')),
	grams: v.pipe(v.number('errors.validation.number'), v.gtValue(0, 'errors.validation.positive')),
}) satisfies v.GenericSchema<unknown, Schemas['Portion']>;

export const foodInputSchema = v.object({
	name: v.pipe(v.string(), v.trim(), v.minLength(1, 'errors.validation.required'), v.maxLength(200, 'errors.validation.maxLength')),
	brand: v.optional(v.pipe(v.string(), v.trim())),
	aliases: v.optional(v.pipe(v.array(v.pipe(v.string(), v.trim(), v.minLength(1), v.maxLength(100, 'errors.validation.maxLength'))), v.maxLength(20, 'errors.validation.maxLength'))),
	barcode: v.optional(v.pipe(v.string(), v.trim())),
	nutrients: v.record(v.string(), v.object({
		amount: v.optional(v.pipe(v.number(), v.minValue(0, 'errors.validation.positive'))),
		status: v.optional(v.picklist(['measured', 'estimated', 'trace', 'missing'])),
	})),
	basis: v.optional(v.object({
		unit: v.pipe(v.string(), v.trim(), v.minLength(1, 'errors.validation.required'), v.maxLength(30, 'errors.validation.maxLength'), v.notValue('g', 'errors.validation.required')),
		grams: v.optional(v.pipe(v.number('errors.validation.number'), v.gtValue(0, 'errors.validation.positive'))),
	})),
	portions: v.optional(v.array(portionSchema)),
}) satisfies v.GenericSchema<unknown, Schemas['FoodInput']>;

export const quantitySchema = v.object({
	amount: v.pipe(v.number('errors.validation.number'), v.gtValue(0, 'errors.validation.positive')),
	unit: v.pipe(v.string(), v.minLength(1, 'errors.validation.required')),
}) satisfies v.GenericSchema<unknown, Schemas['Quantity']>;

export const recipeInputSchema = v.object({
	name: v.pipe(v.string(), v.trim(), v.minLength(1, 'errors.validation.required'), v.maxLength(200, 'errors.validation.maxLength')),
	ingredients: v.pipe(v.array(v.object({ foodId: v.string(), quantity: quantitySchema })), v.minLength(1, 'errors.validation.required')),
	cookedGrams: v.optional(v.pipe(v.number(), v.gtValue(0, 'errors.validation.positive'))),
	servings: v.optional(v.pipe(v.number(), v.gtValue(0, 'errors.validation.positive'))),
}) satisfies v.GenericSchema<unknown, Schemas['RecipeInput']>;

// 食事記録の日付と区分
export const mealPlacementSchema = v.object({
	date: isoDate,
	slot: v.picklist(['breakfast', 'lunch', 'dinner', 'snack']),
}) satisfies v.GenericSchema<unknown, Schemas['UpdateMealEntry']>;

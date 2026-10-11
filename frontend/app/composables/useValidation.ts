import * as v from 'valibot';

// valibotの検証結果をフィールド単位のi18nキーへ変換
export function useValidation<S extends v.GenericSchema>(schema: S) {
	const errors = ref<Record<string, string>>({});

	function validate(value: unknown): v.InferOutput<S> | undefined {
		const result = v.safeParse(schema, value);
		if (result.success) {
			errors.value = {};
			return result.output;
		}
		const flat = v.flatten<S>(result.issues);
		const next: Record<string, string> = {};
		const nested = (flat.nested ?? {}) as Record<string, string[] | undefined>;
		for (const [path, messages] of Object.entries(nested)) {
			if (messages?.[0]) next[path] = messages[0];
		}
		if (flat.root?.[0]) next.$root = flat.root[0];
		errors.value = next;
		return undefined;
	}

	return { errors, validate };
}

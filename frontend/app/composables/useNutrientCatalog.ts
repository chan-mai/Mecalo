import { unwrap, useApi } from './api/client';
import type { Schemas } from './api/client';
import { unitLabel } from '~/utils/nutrients';

export function useNutrientCatalog() {
	const catalog = useState<Schemas['NutrientDefinition'][]>('nutrient-catalog', () => []);

	if (import.meta.client && catalog.value.length === 0) {
		void unwrap(useApi().GET('/reference/nutrients')).then((list) => {
			catalog.value = list;
		}).catch(() => undefined);
	}

	function unitOf(code: string): string {
		const def = catalog.value.find((d) => d.code === code);
		return def ? unitLabel[def.unit] : '';
	}

	const groups = computed(() => {
		const out: Record<Schemas['NutrientGroup'], Schemas['NutrientDefinition'][]> = { energy: [], macro: [], mineral: [], vitamin: [] };
		for (const d of catalog.value) out[d.group].push(d);
		return out;
	});

	return { catalog, unitOf, groups };
}

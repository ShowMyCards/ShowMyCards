export type CompareOp = '=' | '>' | '<';
export type ColorMode = 'all' | 'any' | 'exactly';

export interface SearchBuilderParams {
	cardName: string;
	setCode: string;
	cardType: string;
	colors: Set<string>;
	colorMode: ColorMode;
	numericValues: Record<string, string>;
	numericOps: Record<string, CompareOp>;
}

const NUMERIC_KEYWORDS: Record<string, string> = {
	mv: 'cmc',
	pow: 'pow',
	loy: 'loy',
	tou: 'tou'
};

const COLOR_ORDER = ['w', 'u', 'b', 'r', 'g', 'c'];

export function buildSearchQuery(p: SearchBuilderParams): string {
	const parts: string[] = [];

	if (p.cardName.trim()) {
		const n = p.cardName.trim();
		parts.push(n.includes(' ') ? `name:"${n}"` : `name:${n}`);
	}

	if (p.setCode.trim()) {
		parts.push(`s:${p.setCode.trim()}`);
	}

	if (p.cardType.trim()) {
		const t = p.cardType.trim();
		parts.push(t.includes(' ') ? `t:"${t}"` : `t:${t}`);
	}

	if (p.colors.size > 0) {
		const selected = COLOR_ORDER.filter((c) => p.colors.has(c));
		if (p.colorMode === 'all') {
			parts.push(`c:${selected.join('')}`);
		} else if (p.colorMode === 'exactly') {
			parts.push(`c=${selected.join('')}`);
		} else {
			parts.push(
				selected.length === 1
					? `c:${selected[0]}`
					: `(${selected.map((c) => `c:${c}`).join(' or ')})`
			);
		}
	}

	for (const [id, keyword] of Object.entries(NUMERIC_KEYWORDS)) {
		const val = (p.numericValues[id] ?? '').trim();
		if (val !== '') parts.push(`${keyword}${p.numericOps[id] ?? '='}${val}`);
	}

	return parts.join(' ');
}

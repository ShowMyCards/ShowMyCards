<script lang="ts">
	import { BookOpen } from '@lucide/svelte';

	interface ExampleExpression {
		expression: string;
		description: string;
		category: string;
	}

	let { onInsert }: { onInsert?: (expression: string) => void } = $props();

	let showFields = $state(false);
	let showExamples = $state(false);
	let showOperators = $state(false);

	const cardFields = [
		{
			field: 'treatment',
			type: 'string',
			description: 'Card treatment/finish (foil, nonfoil, etched, etc.)'
		},
		{ field: 'prices.usd', type: 'number', description: 'Price in USD' },
		{ field: 'prices.usd_foil', type: 'number', description: 'Foil price in USD' },
		{ field: 'prices.usd_etched', type: 'number', description: 'Etched foil price in USD' },
		{ field: 'prices.eur', type: 'number', description: 'Price in EUR' },
		{ field: 'prices.eur_foil', type: 'number', description: 'Foil price in EUR' },
		{
			field: 'rarity',
			type: 'string',
			description: 'Card rarity (common, uncommon, rare, mythic)'
		},
		{
			field: 'colors',
			type: 'array',
			description: 'Color array (["W", "U", "B", "R", "G"]), use: "W" in colors'
		},
		{
			field: 'color_identity',
			type: 'array',
			description: 'Color identity array, use: "U" in color_identity'
		},
		{ field: 'cmc', type: 'number', description: 'Converted mana cost / mana value' },
		{ field: 'type_line', type: 'string', description: 'Full type line' },
		{
			field: 'set_type',
			type: 'string',
			description: 'Set type (core, expansion, commander, etc.)'
		},
		{ field: 'set', type: 'string', description: 'Set code (e.g., "MH2")' },
		{ field: 'set_name', type: 'string', description: 'Full set name' },
		{ field: 'name', type: 'string', description: 'Card name' }
	];

	const exampleExpressions: ExampleExpression[] = [
		{
			category: 'Treatment-based',
			expression: 'treatment == "foil"',
			description: 'Only foil versions'
		},
		{
			category: 'Treatment-based',
			expression: 'treatment == "nonfoil"',
			description: 'Only non-foil versions'
		},
		{
			category: 'Treatment-based',
			expression: 'treatment == "foil" && prices.usd_foil > 20.0',
			description: 'Expensive foils (>$20)'
		},
		{
			category: 'Treatment-based',
			expression: 'treatment == "etched"',
			description: 'Etched foil cards'
		},
		{
			category: 'Price-based',
			expression: 'prices.usd > 10.0',
			description: 'Cards worth more than $10'
		},
		{
			category: 'Price-based',
			expression: 'prices.usd > 50.0 && rarity == "mythic"',
			description: 'Expensive mythics'
		},
		{
			category: 'Price-based',
			expression: 'prices.usd < 1.0',
			description: 'Budget cards (under $1)'
		},
		{
			category: 'Rarity-based',
			expression: 'rarity == "mythic"',
			description: 'All mythic rares'
		},
		{
			category: 'Rarity-based',
			expression: 'rarity == "rare" || rarity == "mythic"',
			description: 'Rares and mythics'
		},
		{
			category: 'Color-based',
			expression: 'len(colors) == 0',
			description: 'Colorless cards'
		},
		{
			category: 'Color-based',
			expression: 'len(colors) == 1',
			description: 'Mono-colored cards'
		},
		{
			category: 'Color-based',
			expression: 'len(colors) > 2',
			description: 'Multicolor (3+ colors)'
		},
		{
			category: 'Color-based',
			expression: '"U" in colors && "R" in colors',
			description: 'Blue and red cards'
		},
		{
			category: 'Color-based',
			expression: '"W" in colors && len(colors) == 1',
			description: 'Mono-white cards'
		},
		{
			category: 'Set-based',
			expression: 'set_type == "commander"',
			description: 'Commander set cards'
		},
		{
			category: 'Set-based',
			expression: 'set == "MH2"',
			description: 'Modern Horizons 2 cards'
		},
		{
			category: 'Type-based',
			expression: 'type_line contains "Legendary"',
			description: 'Legendary cards'
		},
		{
			category: 'Type-based',
			expression: 'type_line contains "Creature"',
			description: 'Creature cards'
		},
		{
			category: 'Type-based',
			expression: 'type_line contains "Planeswalker"',
			description: 'Planeswalker cards'
		},
		{
			category: 'Combined',
			expression: 'cmc >= 7 && type_line contains "Creature"',
			description: 'High-cost creatures'
		},
		{
			category: 'Combined',
			expression: 'type_line contains "Legendary" && prices.usd > 10.0',
			description: 'Expensive legendaries'
		},
		{
			category: 'Combined',
			expression: 'treatment == "nonfoil" && rarity == "mythic" && prices.usd > 20.0',
			description: 'Valuable non-foil mythics (>$20)'
		}
	];

	const operators = [
		{ op: '==', description: 'Equal to' },
		{ op: '!=', description: 'Not equal to' },
		{ op: '<', description: 'Less than' },
		{ op: '>', description: 'Greater than' },
		{ op: '<=', description: 'Less than or equal to' },
		{ op: '>=', description: 'Greater than or equal to' },
		{ op: '&&', description: 'Logical AND' },
		{ op: '||', description: 'Logical OR' },
		{
			op: 'contains',
			description: 'String contains substring (e.g., type_line contains "Legendary")'
		},
		{ op: 'in', description: 'Check if value in array (e.g., "W" in colors)' },
		{ op: 'len()', description: 'Length of array or string' }
	];

	function handleInsert(expression: string) {
		if (onInsert) {
			onInsert(expression);
		}
	}

	// Group examples by category
	const groupedExamples = $derived(
		exampleExpressions.reduce(
			(acc, example) => {
				if (!acc[example.category]) {
					acc[example.category] = [];
				}
				acc[example.category].push(example);
				return acc;
			},
			{} as Record<string, ExampleExpression[]>
		)
	);
</script>

<div class="card bg-base-100 shadow-sm border border-base-300">
	<div class="card-body p-4 space-y-4">
		<div class="flex items-center gap-2 text-sm font-semibold">
			<BookOpen class="w-4 h-4" />
			<span>Expression Syntax Guide</span>
		</div>

		<!-- Quick Tips -->
		<div class="alert alert-info py-2">
			<div class="text-xs">
				<div class="font-semibold mb-1">💡 Quick Tips:</div>
				<ul class="list-disc list-inside space-y-0.5 opacity-80">
					<li>Use quotes for string values: <code class="text-xs">rarity == "mythic"</code></li>
					<li>
						Chain conditions with <code class="text-xs">&&</code> (AND) or
						<code class="text-xs">||</code> (OR)
					</li>
					<li>
						Arrays use <code class="text-xs">in</code> operator:
						<code class="text-xs">"W" in colors</code>
					</li>
					<li>
						Strings use <code class="text-xs">contains</code>:
						<code class="text-xs">type_line contains "Legendary"</code>
					</li>
					<li>Colors are uppercase: <code class="text-xs">"W", "U", "B", "R", "G"</code></li>
					<li>Click any example to insert it into the expression field</li>
				</ul>
			</div>
		</div>

		<!-- Available Fields Section -->
		<div class="collapse collapse-arrow bg-base-200 rounded-lg">
			<input type="checkbox" bind:checked={showFields} />
			<div class="collapse-title text-sm font-medium">Available Card Fields</div>
			<div class="collapse-content">
				<div class="space-y-2">
					{#each cardFields as field (field.field)}
						<div class="flex items-start gap-2 text-xs">
							<code class="bg-base-300 px-2 py-0.5 rounded shrink-0 font-mono">
								{field.field}
							</code>
							<div class="flex-1">
								<span class="badge badge-xs badge-ghost">{field.type}</span>
								<span class="opacity-70 ml-2">{field.description}</span>
							</div>
						</div>
					{/each}
				</div>
			</div>
		</div>

		<!-- Example Expressions Section -->
		<div class="collapse collapse-arrow bg-base-200 rounded-lg">
			<input type="checkbox" bind:checked={showExamples} />
			<div class="collapse-title text-sm font-medium">Example Expressions</div>
			<div class="collapse-content">
				<div class="space-y-3">
					{#each Object.entries(groupedExamples) as [category, examples] (category)}
						<div>
							<div class="text-xs font-semibold opacity-70 mb-1">{category}</div>
							<div class="space-y-1">
								{#each examples as example (example.expression)}
									<button
										type="button"
										onclick={() => handleInsert(example.expression)}
										class="btn btn-ghost btn-xs w-full justify-start h-auto py-2 normal-case">
										<div class="text-left w-full">
											<div class="font-mono text-xs text-primary">{example.expression}</div>
											<div class="text-xs opacity-70">{example.description}</div>
										</div>
									</button>
								{/each}
							</div>
						</div>
					{/each}
				</div>
			</div>
		</div>

		<!-- Operators Section -->
		<div class="collapse collapse-arrow bg-base-200 rounded-lg">
			<input type="checkbox" bind:checked={showOperators} />
			<div class="collapse-title text-sm font-medium">Operators & Functions</div>
			<div class="collapse-content">
				<div class="grid grid-cols-2 gap-2">
					{#each operators as operator (operator.op)}
						<div class="text-xs">
							<code class="bg-base-300 px-2 py-0.5 rounded font-mono">{operator.op}</code>
							<span class="opacity-70 ml-2">{operator.description}</span>
						</div>
					{/each}
				</div>
			</div>
		</div>
	</div>
</div>

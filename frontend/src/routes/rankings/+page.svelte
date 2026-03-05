<script lang="ts">
	import { onMount } from 'svelte';
	import { getRankings, getAvailableYears, getAvailableWeightClasses } from '$lib/api';
	import type { RankingBot } from '$lib/types';
	import Card from '$lib/components/Card.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorMessage from '$lib/components/ErrorMessage.svelte';

	let rankings: RankingBot[] = $state([]);
	let years: string[] = $state([]);
	let weightClasses: string[] = $state([]);
	let loading = $state(true);
	let error = $state('');

	let selectedYear = $state('');
	let selectedWeightClass = $state('');

	async function loadInitialData() {
		loading = true;
		error = '';
		try {
			const fetchedYears = await getAvailableYears();
			const fetchedWeightClasses = await getAvailableWeightClasses();
			years = Array.isArray(fetchedYears) ? fetchedYears : [];
			weightClasses = Array.isArray(fetchedWeightClasses) ? fetchedWeightClasses : [];
			
			if (years.length > 0) {
				selectedYear = years[0];
			}
			if (weightClasses.length > 0) {
				selectedWeightClass = weightClasses[0];
			}

			await loadRankings();
		} catch (err) {
			error = 'Failed to load data';
			console.error(err);
		} finally {
			loading = false;
		}
	}

	async function loadRankings() {
		try {
			const fetchedRankings = await getRankings(selectedYear, selectedWeightClass);
			rankings = Array.isArray(fetchedRankings) ? fetchedRankings : [];
		} catch (err) {
			error = 'Failed to load rankings';
			console.error(err);
		}
	}

	onMount(() => {
		loadInitialData();
	});

	function handleFilterChange() {
		loadRankings();
	}
</script>

<svelte:head>
	<title>Rankings - Robot Registry</title>
	<meta name="description" content="View robot combat rankings" />
</svelte:head>

<div class="space-y-6">
	<div class="flex flex-col gap-2">
		<h1 class="text-4xl font-bold text-stone-900 dark:text-stone-100">Rankings</h1>
		<p class="text-stone-600 dark:text-stone-400">
			Top results per weight class and year.
		</p>
	</div>

	{#if loading}
		<LoadingSpinner />
	{:else if error}
		<ErrorMessage message={error} />
	{:else}
		<Card>
			<div class="p-6 space-y-4">
				<div class="flex flex-col md:flex-row md:items-end gap-4">
					<div class="flex-1 min-w-[200px]">
						<label for="year" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">
							Year
						</label>
						<select
							id="year"
							bind:value={selectedYear}
							onchange={handleFilterChange}
							class="w-full px-4 py-3 border border-stone-300 dark:border-stone-600 rounded-lg bg-white dark:bg-stone-800 text-stone-900 dark:text-stone-100"
						>
							{#each years as year}
								<option value={year}>{year}</option>
							{/each}
						</select>
					</div>

					<div class="flex-1 min-w-[240px]">
						<label for="weight-class" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">
							Weight class
						</label>
						<select
							id="weight-class"
							bind:value={selectedWeightClass}
							onchange={handleFilterChange}
							class="w-full px-4 py-3 border border-stone-300 dark:border-stone-600 rounded-lg bg-white dark:bg-stone-800 text-stone-900 dark:text-stone-100"
						>
							{#each weightClasses as weightClass}
								<option value={weightClass}>{weightClass}</option>
							{/each}
						</select>
					</div>

					<div class="text-sm text-stone-600 dark:text-stone-400 md:text-right">
						{#if rankings.length > 0}
							<span class="font-medium text-stone-900 dark:text-stone-100">{rankings.length}</span> bots
						{:else}
							No results
						{/if}
					</div>
				</div>
			</div>
		</Card>

		{#if rankings.length === 0}
			<div class="text-center py-12">
				<p class="text-stone-600 dark:text-stone-400">No rankings found</p>
			</div>
		{:else}
			<Card>
				<div class="hidden md:block overflow-x-auto">
					<table class="w-full">
						<thead class="bg-stone-50 dark:bg-stone-900/40">
							<tr>
								<th class="px-6 py-4 text-left text-xs font-medium text-stone-600 dark:text-stone-400 uppercase tracking-wider">
									Rank
								</th>
								<th class="px-6 py-4 text-left text-xs font-medium text-stone-600 dark:text-stone-400 uppercase tracking-wider">
									Bot
								</th>
								<th class="px-6 py-4 text-left text-xs font-medium text-stone-600 dark:text-stone-400 uppercase tracking-wider">
									Team
								</th>
								<th class="px-6 py-4 text-right text-xs font-medium text-stone-600 dark:text-stone-400 uppercase tracking-wider">
									Points
								</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-stone-200 dark:divide-stone-800">
							{#each rankings as bot}
								<tr class="hover:bg-stone-50 dark:hover:bg-stone-900/30">
									<td class="px-6 py-4">
										{#if bot.rank <= 3}
											<span class="inline-flex items-center justify-center min-w-10 px-2 py-1 rounded-full bg-orange-100 dark:bg-orange-900 text-orange-800 dark:text-orange-200 font-bold">
												#{bot.rank}
											</span>
										{:else}
											<span class="text-stone-900 dark:text-stone-100 font-semibold">#{bot.rank}</span>
										{/if}
									</td>
									<td class="px-6 py-4">
										<div class="flex items-center gap-3">
											<div class="w-12 h-12 rounded bg-stone-100 dark:bg-stone-800 overflow-hidden flex items-center justify-center">
												{#if bot.image_url}
													<img src={bot.image_url} alt={bot.name} class="w-full h-full object-cover" loading="lazy" />
												{:else}
													<span class="text-xs text-stone-500 dark:text-stone-400">No image</span>
												{/if}
											</div>
											<div class="min-w-0">
												<a href="/bots/{bot.id}" class="block font-semibold text-stone-900 dark:text-stone-100 hover:underline truncate">
													{bot.name}
												</a>
												<div class="text-xs text-stone-600 dark:text-stone-400 truncate">{bot.weight_class}</div>
											</div>
										</div>
									</td>
									<td class="px-6 py-4">
										{#if bot.team_id}
											<a href="/teams/{bot.team_id}" class="text-orange-600 dark:text-orange-400 hover:underline">
												{bot.team || 'Team'}
											</a>
										{:else}
											<span class="text-stone-600 dark:text-stone-400">—</span>
										{/if}
									</td>
									<td class="px-6 py-4 text-right">
										<span class="font-mono font-semibold text-stone-900 dark:text-stone-100">{bot.points}</span>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<div class="md:hidden divide-y divide-stone-200 dark:divide-stone-800">
					{#each rankings as bot}
						<a href="/bots/{bot.id}" class="block p-4 hover:bg-stone-50 dark:hover:bg-stone-900/30">
							<div class="flex items-center gap-3">
								<div class="w-10 h-10 rounded bg-stone-100 dark:bg-stone-800 overflow-hidden flex items-center justify-center">
									{#if bot.image_url}
										<img src={bot.image_url} alt={bot.name} class="w-full h-full object-cover" loading="lazy" />
									{:else}
										<span class="text-[10px] text-stone-500 dark:text-stone-400">No image</span>
									{/if}
								</div>
								<div class="min-w-0 flex-1">
									<div class="flex items-baseline justify-between gap-2">
										<div class="font-semibold text-stone-900 dark:text-stone-100 truncate">#{bot.rank} • {bot.name}</div>
										<div class="font-mono text-sm text-stone-900 dark:text-stone-100">{bot.points}</div>
									</div>
									<div class="text-xs text-stone-600 dark:text-stone-400 truncate">
										{bot.team || '—'}
									</div>
								</div>
							</div>
						</a>
					{/each}
				</div>
			</Card>
		{/if}
	{/if}
</div>
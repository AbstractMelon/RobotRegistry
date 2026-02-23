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
	<div>
		<h1 class="text-4xl font-bold text-stone-900 dark:text-stone-100">Rankings</h1>
	</div>

	{#if loading}
		<LoadingSpinner />
	{:else if error}
		<ErrorMessage message={error} />
	{:else}
		<div class="bg-white dark:bg-stone-800 rounded-lg shadow p-6 space-y-4">
			<div class="flex flex-wrap gap-4">
				<div class="flex-1 min-w-[200px]">
					<label for="year" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">
						Year
					</label>
					<select
						id="year"
						bind:value={selectedYear}
						onchange={handleFilterChange}
						class="w-full px-3 py-2 border border-stone-300 dark:border-stone-600 rounded-lg 
						       bg-white dark:bg-stone-700 text-stone-900 dark:text-stone-100"
					>
						{#each years as year}
							<option value={year}>{year}</option>
						{/each}
					</select>
				</div>

				<div class="flex-1 min-w-[200px]">
					<label for="weight-class" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">
						Weight Class
					</label>
					<select
						id="weight-class"
						bind:value={selectedWeightClass}
						onchange={handleFilterChange}
						class="w-full px-3 py-2 border border-stone-300 dark:border-stone-600 rounded-lg 
						       bg-white dark:bg-stone-700 text-stone-900 dark:text-stone-100"
					>
						{#each weightClasses as weightClass}
							<option value={weightClass}>{weightClass}</option>
						{/each}
					</select>
				</div>
			</div>
		</div>

		{#if rankings.length === 0}
			<div class="text-center py-12">
				<p class="text-stone-600 dark:text-stone-400">No rankings found</p>
			</div>
		{:else}
			<Card>
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead class="bg-stone-50 dark:bg-stone-700">
							<tr>
								<th class="px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">
									Rank
								</th>
								<th class="px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">
									Bot
								</th>
								<th class="px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">
									Team
								</th>
								<th class="px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">
									Points
								</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-stone-200 dark:divide-stone-700">
							{#each rankings as bot}
								<tr class="hover:bg-stone-50 dark:hover:bg-stone-700">
									<td class="px-6 py-4">
										<span class="text-2xl font-bold text-orange-600 dark:text-orange-400">
											#{bot.rank}
										</span>
									</td>
									<td class="px-6 py-4">
										<div class="flex items-center space-x-3">
											{#if bot.image_url}
												<img src={bot.image_url} alt={bot.name} class="w-12 h-12 object-cover rounded" />
											{/if}
											<a 
												href="/bots/{bot.id}" 
												class="text-orange-600 dark:text-orange-400 hover:underline font-medium"
											>
												{bot.name}
											</a>
										</div>
									</td>
									<td class="px-6 py-4">
										<a 
											href="/teams/{bot.team_id}" 
											class="text-orange-600 dark:text-orange-400 hover:underline"
										>
											{bot.team}
										</a>
									</td>
									<td class="px-6 py-4 text-stone-900 dark:text-stone-100 font-semibold">
										{bot.points}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</Card>
		{/if}
	{/if}
</div>
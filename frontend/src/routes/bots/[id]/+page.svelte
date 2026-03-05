<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { getBot, getAvailableYears } from '$lib/api';
	import type { Bot, BotRanking } from '$lib/types';
	import Card from '$lib/components/Card.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorMessage from '$lib/components/ErrorMessage.svelte';

	let bot: Bot | null = $state(null);
	let loading = $state(true);
	let error = $state('');
	let availableYears: string[] = $state([]);
	let selectedYear = $state('');

	$effect(() => {
		const botId = $page.params.id;
		if (botId) {
			loadBot(botId);
		}
	});

	onMount(async () => {
		try {
			availableYears = await getAvailableYears();
			if (availableYears.length > 0) {
				selectedYear = availableYears[0]; // Default to latest year
			}
		} catch (err) {
			console.error('Failed to load years', err);
		}
	});

	async function loadBot(id: string) {
		loading = true;
		error = '';
		try {
			bot = await getBot(id);
			// Set selected year to latest if bot has rankings
			if (bot?.rankings && bot.rankings.length > 0 && !selectedYear) {
				selectedYear = bot.rankings[0].year;
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load bot';
			console.error(err);
		} finally {
			loading = false;
		}
	}

	// Get the ranking for the selected year
	let currentRanking = $derived.by(() => {
		if (!bot?.rankings || bot.rankings.length === 0) return null;
		const ranking = bot.rankings.find((r: BotRanking) => r.year === selectedYear);
		return ranking || bot.rankings[0];
	});

</script>

<svelte:head>
	<title>{bot?.name || 'Bot'} - Robot Registry</title>
</svelte:head>

{#if loading}
	<LoadingSpinner />
{:else if error}
	<ErrorMessage message={error} />
{:else if bot}
	<div class="space-y-8">
		<div class="bg-white dark:bg-stone-800 rounded-lg shadow-lg overflow-hidden">
			<div class="p-8">
				<div class="flex flex-col md:flex-row gap-6">
					{#if bot.image_url}
						<div class="flex-shrink-0 w-auto h-full max-w-xl rounded overflow-hidden bg-stone-100">
							<img src={bot.image_url} alt={bot.name} class="w-full h-full object-contain" />
						</div>
					{/if}
					<div class="flex-1">
						<div class="flex items-start justify-between mb-6">
							<div>
								<h1 class="text-4xl font-bold text-stone-900 dark:text-stone-100 mb-2">
									{bot.name}
								</h1>
								<p class="text-xl text-stone-600 dark:text-stone-400">
									{currentRanking?.weight_class || bot.weight_class}
								</p>
							</div>
							{#if currentRanking?.rank}
								<div class="text-center">
									<div class="text-5xl font-bold text-orange-600 dark:text-orange-400">
										#{currentRanking.rank}
									</div>
									<div class="text-sm text-stone-600 dark:text-stone-400">
										Rank
									</div>
								</div>
							{/if}
						</div>

						{#if bot.rankings && bot.rankings.length > 0}
							<div class="mb-6">
								<label for="year-select" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">
									Select Year
								</label>
								<select
									id="year-select"
									bind:value={selectedYear}
									class="px-4 py-2 border border-stone-300 dark:border-stone-600 rounded-lg 
										bg-white dark:bg-stone-700 text-stone-900 dark:text-stone-100"
								>
									{#each bot.rankings as ranking}
										<option value={ranking.year}>{ranking.year}</option>
									{/each}
								</select>
							</div>
						{/if}

						<div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
							<div>
								<h3 class="text-sm font-medium text-stone-500 dark:text-stone-400 mb-1">Points</h3>
								<p class="text-2xl font-semibold text-stone-900 dark:text-stone-100">{currentRanking?.points || bot.points}</p>
							</div>
							<div>
								<h3 class="text-sm font-medium text-stone-500 dark:text-stone-400 mb-1">Team</h3>
								<a href="/teams/{bot.team_id}" class="text-2xl font-semibold text-orange-600 dark:text-orange-400 hover:underline">
									{bot.team}
								</a>
							</div>
							<div>
								<h3 class="text-sm font-medium text-stone-500 dark:text-stone-400 mb-1">Years Active</h3>
								<p class="text-2xl font-semibold text-stone-900 dark:text-stone-100">
									{bot.years?.join(', ') || 'N/A'}
								</p>
							</div>
						</div>

						{#if bot.weapons && bot.weapons.length > 0}
							<div class="mb-6">
								<h3 class="text-sm font-medium text-stone-500 dark:text-stone-400 mb-2">Weapons</h3>
								<div class="flex flex-wrap gap-2">
									{#each bot.weapons as weapon}
										<span class="px-3 py-1 bg-orange-100 dark:bg-orange-900 text-orange-800 dark:text-orange-200 rounded-full">
											{weapon}
										</span>
									{/each}
								</div>
							</div>
						{/if}

						{#if bot.description}
							<div class="mb-6">
								<h3 class="text-sm font-medium text-stone-500 dark:text-stone-400 mb-2">Description</h3>
								<p class="text-stone-700 dark:text-stone-300">{bot.description}</p>
							</div>
						{/if}

						<a
							href={bot.url}
							target="_blank"
							rel="noopener noreferrer"
							class="inline-block px-6 py-3 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors"
						>
							View on Robot Combat Events
						</a>
					</div>
				</div>
			</div>
		</div>

		{#if bot.history && bot.history.length > 0}
			<div>
				<h2 class="text-3xl font-bold text-stone-900 dark:text-stone-100 mb-6">Competition History</h2>
				<Card>
					<div class="overflow-x-auto">
						<table class="w-full">
							<thead class="bg-stone-50 dark:bg-stone-700">
								<tr>
									<th class="px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">
										Event
									</th>
									<th class="px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">
										Place
									</th>
									<th class="px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">
										Points
									</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-stone-200 dark:divide-stone-700">
								{#each bot.history as record}
									<tr class="hover:bg-stone-50 dark:hover:bg-stone-700">
										<td class="px-6 py-4">
											<a 
												href={record.competition_url} 
												target="_blank" 
												rel="noopener noreferrer"
												class="text-orange-600 dark:text-orange-400 hover:underline"
											>
												{record.event_name}
											</a>
										</td>
										<td class="px-6 py-4 text-stone-900 dark:text-stone-100">
											{record.place || 'N/A'}
										</td>
										<td class="px-6 py-4 text-stone-900 dark:text-stone-100">
											{record.points}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</Card>
			</div>
		{/if}
	</div>
{/if}
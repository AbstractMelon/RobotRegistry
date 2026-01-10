<script lang="ts">
	import { search as searchAPI } from '$lib/api';
	import type { SearchResult } from '$lib/types';
	import Card from '$lib/components/Card.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorMessage from '$lib/components/ErrorMessage.svelte';

	let query = $state('');
	let results: SearchResult | null = $state(null);
	let loading = $state(false);
	let error = $state('');
	let hasSearched = $state(false);

	async function handleSearch() {
		if (!query.trim()) {
			return;
		}

		loading = true;
		error = '';
		hasSearched = true;
		try {
			results = await searchAPI(query, 20);
		} catch (err) {
			error = 'Search failed';
			console.error(err);
		} finally {
			loading = false;
		}
	}

	function handleKeyPress(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			handleSearch();
		}
	}
</script>

<svelte:head>
	<title>Search - Robot Registry</title>
	<meta name="description" content="Search robot combat events, bots, and teams" />
</svelte:head>

<div class="space-y-6">
	<div>
		<h1 class="text-4xl font-bold text-gray-900 dark:text-gray-100 mb-6">Search</h1>
		
		<div class="flex gap-4">
			<input
				type="text"
				bind:value={query}
				onkeypress={handleKeyPress}
				placeholder="Search for events, bots, or teams..."
				class="flex-1 px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg 
				       bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 
				       focus:ring-2 focus:ring-blue-500 focus:border-transparent"
			/>
			<button
				onclick={handleSearch}
				class="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
			>
				Search
			</button>
		</div>
	</div>

	{#if loading}
		<LoadingSpinner />
	{:else if error}
		<ErrorMessage message={error} />
	{:else if hasSearched && results}
		<div class="space-y-8">
			{#if results.events && results.events.length > 0}
				<div>
					<h2 class="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-4">
						Events ({results.events.length})
					</h2>
					<div class="grid grid-cols-1 md:grid-cols-4 lg:grid-cols-6 gap-6">
						{#each results.events as event}
							<Card href="/events/{event.id}">
								<div class="p-6">
									{#if event.logo_url}
										<img src={event.logo_url} alt={event.name} class="w-full h-32 object-contain mb-4" />
									{/if}
									<h3 class="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
										{event.name}
									</h3>
									<p class="text-sm text-gray-600 dark:text-gray-400 mb-1">
										{event.location}
									</p>
									<p class="text-sm text-gray-500 dark:text-gray-500">
										{event.start_date}
									</p>
								</div>
							</Card>
						{/each}
					</div>
				</div>
			{/if}

			{#if results.bots && results.bots.length > 0}
				<div>
					<h2 class="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-4">
						Bots ({results.bots.length})
					</h2>
					<div class="grid grid-cols-1 md:grid-cols-4 lg:grid-cols-6 gap-6">
						{#each results.bots as bot}
							<Card href="/bots/{bot.id}">
								<div class="p-6">
									{#if bot.image_url}
										<img src={bot.image_url} alt={bot.name} class="w-full h-48 object-cover rounded mb-4" />
									{/if}
									<div class="flex items-center justify-between mb-2">
										<h3 class="text-xl font-semibold text-gray-900 dark:text-gray-100">
											{bot.name}
										</h3>
										{#if bot.rank}
											<span class="text-xl font-bold text-blue-600 dark:text-blue-400">
												#{bot.rank}
											</span>
										{/if}
									</div>
									<p class="text-sm text-gray-600 dark:text-gray-400">
										{bot.weight_class}
									</p>
									<p class="text-sm text-gray-500 dark:text-gray-500">
										Team: {bot.team}
									</p>
								</div>
							</Card>
						{/each}
					</div>
				</div>
			{/if}

			{#if results.teams && results.teams.length > 0}
				<div>
					<h2 class="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-4">
						Teams ({results.teams.length})
					</h2>
					<div class="grid grid-cols-1 md:grid-cols-4 lg:grid-cols-6 gap-6">
						{#each results.teams as team}
							<Card href="/teams/{team.id}">
								<div class="p-6">
									{#if team.logo_url}
										<img src={team.logo_url} alt={team.name} class="w-full h-32 object-contain mb-4" />
									{/if}
									<h3 class="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
										{team.name}
									</h3>
									<p class="text-sm text-gray-600 dark:text-gray-400">
										{team.bot_ids?.length || 0} bots
									</p>
								</div>
							</Card>
						{/each}
					</div>
				</div>
			{/if}

			{#if (!results.events || results.events.length === 0) && 
			     (!results.bots || results.bots.length === 0) && 
			     (!results.teams || results.teams.length === 0)}
				<div class="text-center py-12">
					<p class="text-gray-600 dark:text-gray-400">No results found for "{query}"</p>
				</div>
			{/if}
		</div>
	{:else if hasSearched}
		<div class="text-center py-12">
			<p class="text-gray-600 dark:text-gray-400">Enter a search query to get started</p>
		</div>
	{/if}
</div>
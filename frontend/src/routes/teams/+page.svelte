<script lang="ts">
	import { onMount } from 'svelte';
	import { getTeams } from '$lib/api';
	import type { Team } from '$lib/types';
	import Card from '$lib/components/Card.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorMessage from '$lib/components/ErrorMessage.svelte';
	import Pagination from '$lib/components/Pagination.svelte';

	let teams: Team[] = $state([]);
	let loading = $state(true);
	let error = $state('');
	let currentPage = $state(1);
	let totalPages = $state(1);
	let totalItems = $state(0);

	async function loadTeams() {
		loading = true;
		error = '';
		try {
			const response = await getTeams(currentPage, 12);
			teams = response.data;
			totalPages = response.total_pages;
			totalItems = response.total_items;
		} catch (err) {
			error = 'Failed to load teams';
			console.error(err);
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadTeams();
	});

	function handlePageChange(page: number) {
		currentPage = page;
		loadTeams();
	}
</script>

<svelte:head>
	<title>Teams - Robot Registry</title>
	<meta name="description" content="Browse robot combat teams" />
</svelte:head>

<div class="space-y-6">
	<div class="flex justify-between items-center">
		<h1 class="text-4xl font-bold text-stone-900 dark:text-stone-100">Teams</h1>
		<div class="text-sm text-stone-600 dark:text-stone-400">
			{totalItems} teams found
		</div>
	</div>

	{#if loading}
		<LoadingSpinner />
	{:else if error}
		<ErrorMessage message={error} />
	{:else if teams.length === 0}
		<div class="text-center py-12">
			<p class="text-stone-600 dark:text-stone-400">No teams found</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-4 lg:grid-cols-6 gap-6">
			{#each teams as team}
				<Card href="/teams/{team.id}">
					<div class="p-6">
						{#if team.logo_url}
							<img src={team.logo_url} alt={team.name} class="w-full h-32 object-contain mb-4" />
						{/if}
						<h3 class="text-xl font-semibold text-stone-900 dark:text-stone-100 mb-2">
							{team.name}
						</h3>
						<p class="text-sm text-stone-600 dark:text-stone-400">
							{team.bot_ids?.length || 0} bots
						</p>
					</div>
				</Card>
			{/each}
		</div>

		{#if totalPages > 1}
			<Pagination 
				{currentPage} 
				{totalPages} 
				onPageChange={handlePageChange}
			/>
		{/if}
	{/if}
</div>
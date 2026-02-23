<script lang="ts">
	import { onMount } from 'svelte';
	import { getEvents, getAvailableWeightClasses } from '$lib/api';
	import type { Event } from '$lib/types';
	import Card from '$lib/components/Card.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorMessage from '$lib/components/ErrorMessage.svelte';
	import Pagination from '$lib/components/Pagination.svelte';

	let events: Event[] = $state([]);
	let loading = $state(true);
	let error = $state('');
	let currentPage = $state(1);
	let totalPages = $state(1);
	let totalItems = $state(0);
	let weightClasses: string[] = $state([]);

	let locationFilter = $state('');
	let weightClassFilter = $state('');
	let viewMode = $state<'upcoming' | 'past' | 'all'>('upcoming');

	async function loadEvents() {
		loading = true;
		error = '';
		try {
			const today = new Date().toISOString().split('T')[0];
			const filters: any = {};
			
			if (locationFilter) filters.location = locationFilter;
			if (weightClassFilter) filters.weight_class = weightClassFilter;
			
			if (viewMode === 'upcoming') {
				filters.sort = 'start_date_asc';
				filters.start_date = today;
			} else if (viewMode === 'past') {
				filters.sort = 'start_date_desc';
				filters.end_date = today;
			}

			const response = await getEvents(currentPage, 18, filters);
			events = response.data;
			totalPages = response.total_pages;
			totalItems = response.total_items;
		} catch (err) {
			error = 'Failed to load events';
			console.error(err);
		} finally {
			loading = false;
		}
	}

	async function loadWeightClasses() {
		try {
			weightClasses = await getAvailableWeightClasses();
		} catch (err) {
			console.error('Failed to load weight classes', err);
		}
	}

	onMount(() => {
		loadEvents();
		loadWeightClasses();
	});

	function handlePageChange(page: number) {
		currentPage = page;
		loadEvents();
	}

	function handleFilterChange() {
		currentPage = 1;
		loadEvents();
	}
</script>

<svelte:head>
	<title>Events - Robot Registry</title>
	<meta name="description" content="Browse robot combat events" />
</svelte:head>

<div class="space-y-6">
	<div class="flex justify-between items-center">
		<h1 class="text-4xl font-bold text-stone-900 dark:text-stone-100">Events</h1>
		<div class="text-sm text-stone-600 dark:text-stone-400">
			{totalItems} events found
		</div>
	</div>

	<div class="bg-white dark:bg-stone-800 rounded-xl shadow p-6 space-y-4">
		<div class="flex flex-wrap gap-4">
			<div class="flex-1 min-w-[200px]">
				<label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">
					Location
				</label>
				<input
					type="text"
					bind:value={locationFilter}
					on:change={handleFilterChange}
					placeholder="Filter by location"
					class="w-full px-3 py-2 border border-stone-300 dark:border-stone-600 rounded-xl 
					       bg-white dark:bg-stone-700 text-stone-900 dark:text-stone-100"
				/>
			</div>

			<div class="flex-1 min-w-[200px]">
				<label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">
					Weight Class
				</label>
				<select
					bind:value={weightClassFilter}
					on:change={handleFilterChange}
					class="w-full px-3 py-2 border border-stone-300 dark:border-stone-600 rounded-xl 
						bg-white dark:bg-stone-700 text-stone-900 dark:text-stone-100"
				>
					<option value="">All weight classes</option>
					{#each weightClasses as weightClass}
						<option value={weightClass}>{weightClass}</option>
					{/each}
				</select>
			</div>
		</div>

		<div class="flex space-x-2">
			<button
				on:click={() => { viewMode = 'upcoming'; handleFilterChange(); }}
				class="px-4 py-2 rounded-xl {viewMode === 'upcoming' 
					? 'bg-orange-600 text-white' 
					: 'bg-stone-200 dark:bg-stone-700 text-stone-900 dark:text-stone-100'}"
			>
				Upcoming
			</button>
			<button
				on:click={() => { viewMode = 'past'; handleFilterChange(); }}
				class="px-4 py-2 rounded-xl {viewMode === 'past' 
					? 'bg-orange-600 text-white' 
					: 'bg-stone-200 dark:bg-stone-700 text-stone-900 dark:text-stone-100'}"
			>
				Past
			</button>
			<button
				on:click={() => { viewMode = 'all'; handleFilterChange(); }}
				class="px-4 py-2 rounded-xl {viewMode === 'all' 
					? 'bg-orange-600 text-white' 
					: 'bg-stone-200 dark:bg-stone-700 text-stone-900 dark:text-stone-100'}"
			>
				All
			</button>
		</div>
	</div>

	{#if loading}
		<LoadingSpinner />
	{:else if error}
		<ErrorMessage message={error} />
	{:else if events.length === 0}
		<div class="text-center py-12">
			<p class="text-stone-600 dark:text-stone-400">No events found</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-4 lg:grid-cols-6 gap-6">
			{#each events as event}
				<Card href="/events/{event.id}">
					<div class="p-6">
						{#if event.logo_url}
							<img src={event.logo_url} alt={event.name} class="w-full h-32 object-contain mb-4" />
						{/if}
						<h3 class="text-xl font-semibold text-stone-900 dark:text-stone-100 mb-2">
							{event.name}
						</h3>
						<p class="text-sm text-stone-600 dark:text-stone-400 mb-1">
							{event.location}
						</p>
						<p class="text-sm text-stone-500 dark:text-stone-500">
							{event.start_date} {event.end_date && event.end_date !== event.start_date ? `- ${event.end_date}` : ''}
						</p>
						<p class="text-sm text-stone-500 dark:text-stone-500 mt-2">
							{event.bots_count} bots registered
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
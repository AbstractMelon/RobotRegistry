<script lang="ts">
	import { onMount } from 'svelte';
	import { getBots, getAvailableWeightClasses, getAvailableYears } from '$lib/api';
	import type { Bot } from '$lib/types';
	import Card from '$lib/components/Card.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorMessage from '$lib/components/ErrorMessage.svelte';
	import Pagination from '$lib/components/Pagination.svelte';

	let bots: Bot[] = $state([]);
	let loading = $state(true);
	let error = $state('');
	let currentPage = $state(1);
	let totalPages = $state(1);
	let totalItems = $state(0);

	let searchQuery = $state('');
	let weightClassFilter = $state('');
	let weaponFilter = $state('');
	let yearFilter = $state('');
	let weightClasses: string[] = $state([]);
	let availableYears: string[] = $state([]);

	async function loadBots() {
		loading = true;
		error = '';
		try {
			const filters: any = {};
			if (searchQuery) filters.search = searchQuery;
			if (weightClassFilter) filters.weight_class = weightClassFilter;
			if (weaponFilter) filters.weapon = weaponFilter;
			if (yearFilter) filters.year = yearFilter;

			const response = await getBots(currentPage, 18, filters);
			bots = response.data;
			totalPages = response.total_pages;
			totalItems = response.total_items;
		} catch (err) {
			error = 'Failed to load bots';
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

	async function loadYears() {
		try {
			availableYears = await getAvailableYears();
			if (availableYears.length > 0 && !yearFilter) {
				yearFilter = availableYears[0]; // Default to latest year
			}
		} catch (err) {
			console.error('Failed to load years', err);
		}
	}

	onMount(() => {
		loadWeightClasses();
		loadYears();
		loadBots();
	});

	function handlePageChange(page: number) {
		currentPage = page;
		loadBots();
	}

	function handleFilterChange() {
		currentPage = 1;
		loadBots();
	}
</script>

<svelte:head>
	<title>Bots - Robot Registry</title>
	<meta name="description" content="Browse combat robots" />
</svelte:head>

<div class="space-y-6">
	<div class="flex justify-between items-center">
		<h1 class="text-4xl font-bold text-gray-900 dark:text-gray-100">Bots</h1>
		<div class="text-sm text-gray-600 dark:text-gray-400">
			{totalItems} bots found
		</div>
	</div>

	<div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-4">
		<div class="flex flex-wrap gap-4">
			<div class="flex-1 min-w-[200px]">
				<label for="search" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
					Search
				</label>
				<input
					id="search"
					type="text"
					bind:value={searchQuery}
					onchange={handleFilterChange}
					placeholder="Search bots..."
					class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg 
					       bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
				/>
			</div>

			<div class="flex-1 min-w-[200px]">
				<label for="year" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
					Year
				</label>
				<select
					id="year"
					bind:value={yearFilter}
					onchange={handleFilterChange}
					class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg 
						bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
				>
					{#each availableYears as year}
						<option value={year}>{year}</option>
					{/each}
				</select>
			</div>

			<div class="flex-1 min-w-[200px]">
				<label for="weight-class" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
					Weight Class
				</label>
				<select
					id="weight-class"
					bind:value={weightClassFilter}
					onchange={handleFilterChange}
					class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg 
						bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
				>
					<option value="">All weight classes</option>
					{#each weightClasses as weightClass}
						<option value={weightClass}>{weightClass}</option>
					{/each}
				</select>

			</div>

			<div class="flex-1 min-w-[200px]">
				<label for="weapon" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
					Weapon
				</label>
				<input
					id="weapon"
					type="text"
					bind:value={weaponFilter}
					onchange={handleFilterChange}
					placeholder="Filter by weapon"
					class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg 
					       bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
				/>
			</div>
		</div>
	</div>

	{#if loading}
		<LoadingSpinner />
	{:else if error}
		<ErrorMessage message={error} />
	{:else if bots.length === 0}
		<div class="text-center py-12">
			<p class="text-gray-600 dark:text-gray-400">No bots found</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-4 lg:grid-cols-6 gap-6">
			{#each bots as bot}
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
						<p class="text-sm text-gray-600 dark:text-gray-400 mb-1">
							{bot.weight_class}
						</p>
						<p class="text-sm text-gray-500 dark:text-gray-500">
							{bot.points} points
						</p>
						<p class="text-sm text-gray-500 dark:text-gray-500 mt-1">
							Team: {bot.team}
						</p>
						{#if bot.weapons && bot.weapons.length > 0}
							<div class="mt-2 flex flex-wrap gap-1">
								{#each bot.weapons.slice(0, 3) as weapon}
									<span class="px-2 py-1 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 text-xs rounded">
										{weapon}
									</span>
								{/each}
							</div>
						{/if}
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
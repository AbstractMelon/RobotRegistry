<script lang="ts">
	import { onMount } from 'svelte';
	import { getEvents, getRankings } from '$lib/api';
	import type { Event, RankingBot } from '$lib/types';
	import Card from '$lib/components/Card.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorMessage from '$lib/components/ErrorMessage.svelte';

	let upcomingEvents: Event[] = $state([]);
	let topBots: RankingBot[] = $state([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			const today = new Date().toISOString().split('T')[0];
			const eventsResponse = await getEvents(1, 6, { start_date: today });
			upcomingEvents = eventsResponse.data;

			const rankings = await getRankings();
			topBots = rankings.slice(0, 6);
		} catch (err) {
			error = 'Failed to load data';
			console.error(err);
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Robot Registry - Robot Combat Events Browser</title>
	<meta name="description" content="Browse robot combat events, bots, teams, and rankings" />
</svelte:head>

<div class="space-y-12">
	<section class="text-center py-12">
		<h1 class="text-5xl font-bold text-stone-900 dark:text-stone-100 mb-4">
			Robot Registry
		</h1>
		<p class="text-xl text-stone-600 dark:text-stone-400 max-w-2xl mx-auto">
			Welcome to robot registry! The data viewer for RCE.
		</p>
		<div class="mt-8 flex justify-center space-x-4">
			<a
				href="/events"
				class="px-6 py-3 bg-orange-600 text-white rounded-lg font-medium hover:bg-orange-700 transition-colors"
			>
				Browse Events
			</a>
			<a
				href="/search"
				class="px-6 py-3 bg-stone-200 dark:bg-stone-700 text-stone-900 dark:text-stone-100 rounded-lg font-medium hover:bg-stone-300 dark:hover:bg-stone-600 transition-colors"
			>
				Search
			</a>
		</div>
	</section>

	{#if loading}
		<LoadingSpinner />
	{:else if error}
		<ErrorMessage message={error} />
	{:else}
		<section>
			<div class="flex justify-between items-center mb-6">
				<h2 class="text-3xl font-bold text-stone-900 dark:text-stone-100">Upcoming Events</h2>
				<a href="/events" class="text-orange-600 dark:text-orange-400 hover:underline">View all</a>
			</div>
			
			{#if upcomingEvents.length === 0}
				<p class="text-stone-600 dark:text-stone-400">No upcoming events found</p>
			{:else}
				<div class="grid grid-cols-1 md:grid-cols-4 lg:grid-cols-6 gap-6">
					{#each upcomingEvents as event}
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
			{/if}
		</section>

		<section>
			<div class="flex justify-between items-center mb-6">
				<h2 class="text-3xl font-bold text-stone-900 dark:text-stone-100">Top Ranked Bots</h2>
				<a href="/rankings" class="text-orange-600 dark:text-orange-400 hover:underline">View rankings</a>
			</div>
			
			{#if topBots.length === 0}
				<p class="text-stone-600 dark:text-stone-400">No rankings data available</p>
			{:else}
				<div class="grid grid-cols-1 md:grid-cols-4 lg:grid-cols-6 gap-6">
					{#each topBots as bot}
						<Card href="/bots/{bot.id}">
							<div class="p-6">
								{#if bot.image_url}
									<img src={bot.image_url} alt={bot.name} class="w-full h-32 object-cover rounded mb-4" />
								{/if}
								<div class="flex items-center justify-between mb-2">
									<h3 class="text-xl font-semibold text-stone-900 dark:text-stone-100">
										{bot.name}
									</h3>
									<span class="text-2xl font-bold text-orange-600 dark:text-orange-400">
										#{bot.rank}
									</span>
								</div>
								<p class="text-sm text-stone-600 dark:text-stone-400 mb-1">
									{bot.weight_class}
								</p>
								<p class="text-sm text-stone-500 dark:text-stone-500">
									{bot.points} points
								</p>
								<p class="text-sm text-stone-500 dark:text-stone-500 mt-1">
									Team: {bot.team}
								</p>
							</div>
						</Card>
					{/each}
				</div>
			{/if}
		</section>
	{/if}
</div>

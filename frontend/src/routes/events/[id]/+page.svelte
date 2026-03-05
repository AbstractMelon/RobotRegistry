<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { getEvent } from '$lib/api';
	import type { Event } from '$lib/types';
	import Card from '$lib/components/Card.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorMessage from '$lib/components/ErrorMessage.svelte';

	let event: Event | null = $state(null);
	let loading = $state(true);
	let error = $state('');

	// derive the eventId from the page store
	let eventId = $derived($page.params.id);

	onMount(async () => {
		await loadEvent();
	});

	async function loadEvent() {
		if (!eventId) return;
		
		loading = true;
		error = '';
		try {
			event = await getEvent(eventId);
		} catch (err) {
			error = 'Failed to load event';
			console.error(err);
		} finally {
			loading = false;
		}
	}
</script>


<svelte:head>
	<title>{event?.name || 'Event'} - Robot Registry</title>
</svelte:head>

{#if loading}
	<LoadingSpinner />
{:else if error}
	<ErrorMessage message={error} />
{:else if event}
	<div class="space-y-8">
		<div class="bg-white dark:bg-stone-800 rounded-lg shadow-lg p-8">
			<div class="flex flex-col md:flex-row gap-8">
				{#if event.logo_url}
					<div class="md:w-64">
						<img src={event.logo_url} alt={event.name} class="w-full h-auto object-contain" />
					</div>
				{/if}
				<div class="flex-1">
					<h1 class="text-4xl font-bold text-stone-900 dark:text-stone-100 mb-4">
						{event.name}
					</h1>
					<div class="space-y-2 text-stone-600 dark:text-stone-400">
						<p class="text-lg">
							<span class="font-medium">Location:</span> {event.location}
						</p>
						<p class="text-lg">
							<span class="font-medium">Date:</span> 
							{event.start_date} {event.end_date && event.end_date !== event.start_date ? `- ${event.end_date}` : ''}
						</p>
						<p class="text-lg">
							<span class="font-medium">Registered Bots:</span> {event.bots_count}
						</p>
						{#if event.organizer}
							<p class="text-lg">
								<span class="font-medium">Organizer:</span> {event.organizer}
							</p>
						{/if}
					</div>
					
					{#if event.description_html || event.description}
						<div class="mt-4 text-stone-700 dark:text-stone-300">
							<p class="font-medium mb-2">About this event:</p>
							{#if event.description_html}
								<div class="rce-content text-sm leading-relaxed">{@html event.description_html}</div>
							{:else}
								<p class="text-sm leading-relaxed whitespace-pre-line">{event.description}</p>
							{/if}
						</div>
					{/if}
					
					<div class="flex gap-4 mt-6">
						<a
							href={event.url}
							target="_blank"
							rel="noopener noreferrer"
							class="inline-block px-6 py-3 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors"
						>
							View on Robot Combat Events
						</a>
						{#if event.website}
							<a
								href={event.website}
								target="_blank"
								rel="noopener noreferrer"
								class="inline-block px-6 py-3 bg-stone-600 text-white rounded-lg hover:bg-stone-700 transition-colors"
							>
								Event Website
							</a>
						{/if}
					</div>
				</div>
			</div>
		</div>

		<div>
			<h2 class="text-3xl font-bold text-stone-900 dark:text-stone-100 mb-6">Competitions</h2>
			{#if event.competitions && event.competitions.length > 0}
				<div class="space-y-6">
					{#each event.competitions as competition}
						<Card>
							<div class="p-6">
								<div class="flex justify-between items-start mb-4">
									<div>
										<h3 class="text-2xl font-semibold text-stone-900 dark:text-stone-100">
											{competition.name}
										</h3>
										<p class="text-stone-600 dark:text-stone-400 mt-1">
											{competition.weight_class}
										</p>
									</div>
									<a
										href={competition.url}
										target="_blank"
										rel="noopener noreferrer"
										class="px-4 py-2 bg-orange-600 text-white text-sm rounded-lg hover:bg-orange-700"
									>
										Register
									</a>
								</div>

								<div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4 text-sm">
									<div>
										<span class="text-stone-500 dark:text-stone-400">Date:</span>
										<p class="text-stone-900 dark:text-stone-100">{competition.date || 'TBA'}</p>
									</div>
									<div>
										<span class="text-stone-500 dark:text-stone-400">Time:</span>
										<p class="text-stone-900 dark:text-stone-100">
											{competition.begin_time || 'TBA'}
											{competition.end_time ? `- ${competition.end_time}` : ''}
										</p>
									</div>
									<div>
										<span class="text-stone-500 dark:text-stone-400">Participants:</span>
										<p class="text-stone-900 dark:text-stone-100">
											{competition.participants?.length || 0} / {competition.max_combatants || '∞'}
										</p>
									</div>
									<div>
										<span class="text-stone-500 dark:text-stone-400">Fee:</span>
										<p class="text-stone-900 dark:text-stone-100">{competition.registration_fee || 'N/A'}</p>
									</div>
								</div>

								{#if competition.participants && competition.participants.length > 0}
									<div>
										<h4 class="font-medium text-stone-900 dark:text-stone-100 mb-3">
											Registered Bots ({competition.participants.length})
										</h4>
										<div class="grid grid-cols-1 md:grid-cols-4 lg:grid-cols-6 gap-3">
											{#each competition.participants as participant}
												<a
													href="/bots/{participant.bot_id}"
													class="flex items-center space-x-3 p-3 bg-stone-50 dark:bg-stone-700 rounded-lg hover:bg-stone-100 dark:hover:bg-stone-600 transition-colors"
												>
													{#if participant.bot_image}
														<img
															src={participant.bot_image}
															alt={participant.bot_name}
															class="w-16 h-16 object-cover rounded"
														/>
													{/if}
													<div class="flex-1 min-w-0">
														<p class="font-medium text-stone-900 dark:text-stone-100 truncate">
															{participant.bot_name}
														</p>
														<p class="text-sm text-stone-600 dark:text-stone-400 truncate">
															{participant.team_name}
														</p>
													</div>
												</a>
											{/each}
										</div>
									</div>
								{/if}
							</div>
						</Card>
					{/each}
				</div>
			{:else}
				<p class="text-stone-600 dark:text-stone-400">No competitions listed</p>
			{/if}
		</div>
	</div>
{/if}
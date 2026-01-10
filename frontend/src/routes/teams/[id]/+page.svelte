<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { getTeam } from '$lib/api';
	import type { Team } from '$lib/types';
	import Card from '$lib/components/Card.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorMessage from '$lib/components/ErrorMessage.svelte';

	let team: Team | null = $state(null);
	let loading = $state(true);
	let error = $state('');

	$effect(() => {
		const teamId = $page.params.id;
		if (teamId) {
			loadTeam(teamId);
		}
	});

	async function loadTeam(id: string) {
		loading = true;
		error = '';
		try {
			team = await getTeam(id);
		} catch (err) {
			error = 'Failed to load team';
			console.error(err);
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>{team?.name || 'Team'} - Robot Registry</title>
</svelte:head>

{#if loading}
	<LoadingSpinner />
{:else if error}
	<ErrorMessage message={error} />
{:else if team}
	<div class="space-y-8">
		<div class="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8">
			<div class="flex flex-col md:flex-row gap-8 items-center">
				{#if team.logo_url}
					<div class="md:w-48">
						<img src={team.logo_url} alt={team.name} class="w-full h-auto object-contain" />
					</div>
				{/if}
				<div class="flex-1 text-center md:text-left">
					<h1 class="text-4xl font-bold text-gray-900 dark:text-gray-100 mb-4">
						{team.name}
					</h1>
					<p class="text-lg text-gray-600 dark:text-gray-400 mb-4">
						{team.bot_ids?.length || 0} bots
					</p>
					<a
						href={team.url}
						target="_blank"
						rel="noopener noreferrer"
						class="inline-block px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
					>
						View on Robot Combat Events
					</a>
				</div>
			</div>
		</div>

		<div>
			<h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6">Team Roster</h2>
			{#if team.bot_ids && team.bot_ids.length > 0}
				<div class="grid grid-cols-1 md:grid-cols-4 lg:grid-cols-6 gap-6">
					{#each team.bot_ids as botId, index}
						<Card href="/bots/{botId}">
							<div class="p-6">
								<h3 class="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
									{team.bot_names?.[index] || 'Unknown Bot'}
								</h3>
								<p class="text-sm text-blue-600 dark:text-blue-400 hover:underline">
									View Profile
								</p>
							</div>
						</Card>
					{/each}
				</div>
			{:else}
				<p class="text-gray-600 dark:text-gray-400">No bots found for this team</p>
			{/if}
		</div>
	</div>
{/if}
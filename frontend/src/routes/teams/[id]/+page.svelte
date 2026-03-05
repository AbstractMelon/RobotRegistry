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
		<div class="bg-white dark:bg-stone-800 rounded-lg shadow-lg p-8">
			<div class="flex flex-col md:flex-row gap-8 items-center">
				{#if team.logo_url}
					<div class="md:w-48">
						<img src={team.logo_url} alt={team.name} class="w-full h-auto object-contain" />
					</div>
				{/if}
				<div class="flex-1 text-center md:text-left">
					<h1 class="text-4xl font-bold text-stone-900 dark:text-stone-100 mb-4">
						{team.name}
					</h1>
					<p class="text-lg text-stone-600 dark:text-stone-400 mb-4">
						{team.bot_ids?.length || 0} bots
					</p>
					{#if team.description}
						<p class="text-stone-700 dark:text-stone-300 leading-relaxed whitespace-pre-line mb-4">
							{team.description}
						</p>
					{/if}
					{#if team.website || team.email || team.phone || team.address}
						<div class="text-sm text-stone-600 dark:text-stone-400 space-y-1 mb-4">
							{#if team.website}
								<p>
									<span class="font-medium">Website:</span>
									<a class="text-orange-600 dark:text-orange-400 hover:underline" href={team.website} target="_blank" rel="noopener noreferrer">
										{team.website}
									</a>
								</p>
							{/if}
							{#if team.email}
								<p><span class="font-medium">Email:</span> {team.email}</p>
							{/if}
							{#if team.phone}
								<p><span class="font-medium">Phone:</span> {team.phone}</p>
							{/if}
							{#if team.address}
								<p><span class="font-medium">Address:</span> {team.address}</p>
							{/if}
						</div>
					{/if}
					<a
						href={team.url}
						target="_blank"
						rel="noopener noreferrer"
						class="inline-block px-6 py-3 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors"
					>
						View on Robot Combat Events
					</a>
				</div>
			</div>
		</div>

		{#if team.members && team.members.length > 0}
			<Card>
				<div class="p-6">
					<h2 class="text-2xl font-bold text-stone-900 dark:text-stone-100 mb-4">Team members</h2>
					<ul class="space-y-1 text-stone-700 dark:text-stone-300">
						{#each team.members as member}
							<li>{member}</li>
						{/each}
					</ul>
				</div>
			</Card>
		{/if}

		<div>
			<h2 class="text-3xl font-bold text-stone-900 dark:text-stone-100 mb-6">Team Roster</h2>
			{#if team.bot_ids && team.bot_ids.length > 0}
				<div class="grid grid-cols-1 md:grid-cols-4 lg:grid-cols-6 gap-6">
					{#each team.bot_ids as botId, index}
						<Card href="/bots/{botId}">
							<div class="p-6">
								<h3 class="text-xl font-semibold text-stone-900 dark:text-stone-100 mb-2">
									{team.bot_names?.[index] || 'Unknown Bot'}
								</h3>
								<p class="text-sm text-orange-600 dark:text-orange-400 hover:underline">
									View Profile
								</p>
							</div>
						</Card>
					{/each}
				</div>
			{:else}
				<p class="text-stone-600 dark:text-stone-400">No bots found for this team</p>
			{/if}
		</div>
	</div>
{/if}
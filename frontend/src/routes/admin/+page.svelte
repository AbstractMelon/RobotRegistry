<script lang="ts">
	import Card from '$lib/components/Card.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorMessage from '$lib/components/ErrorMessage.svelte';
	import { adminCancelJob, adminGetJob, adminListJobs, adminScrapeURL, adminStartJob } from '$lib/api';
	import { onMount } from 'svelte';

	import type { AdminJob as Job } from '$lib/types';

	let starting = $state(false);
	let job: Job | null = $state(null);
	let recentJobs: Job[] = $state([]);
	let error = $state('');
	let url = $state('');
	let polling = $state(false);
	let loadingRecent = $state(true);
	let rankingsYear = $state(String(new Date().getFullYear()));

	onMount(() => {
		void loadRecent();
	});

	async function loadRecent() {
		loadingRecent = true;
		try {
			recentJobs = await adminListJobs(20) || [];
		} catch (e) {
			// ignore list failures; admin page can still run jobs
		} finally {
			loadingRecent = false;
		}
	}

	async function startJob(kind: string) {
		starting = true;
		error = '';
		try {
			job = await adminStartJob(kind);
			poll();
			await loadRecent();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			starting = false;
		}
	}

	async function startRankings(includeBots: boolean) {
		starting = true;
		error = '';
		try {
			job = await adminStartJob('rankings', {
				year: rankingsYear.trim(),
				include_bots: includeBots
			});
			poll();
			await loadRecent();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			starting = false;
		}
	}

	async function cancelCurrentJob() {
		if (!job) return;
		error = '';
		try {
			job = await adminCancelJob(job.id);
			poll();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	}

	async function scrapeURL() {
		starting = true;
		error = '';
		try {
			job = await adminScrapeURL(url);
			poll();
			await loadRecent();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			starting = false;
		}
	}

	async function poll() {
		if (!job) return;
		if (polling) return;
		polling = true;
		try {
			while (job && job.state === 'running') {
				job = await adminGetJob(job.id);
				await new Promise((r) => setTimeout(r, 1000));
			}
			await loadRecent();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			polling = false;
		}
	}

	let progress = $derived.by(() =>
		job && job.total > 0 ? Math.round((job.done / job.total) * 100) : 0
	);
</script>

<svelte:head>
	<title>Admin - Robot Registry</title>
</svelte:head>

<div class="space-y-6">
	<div>
		<h1 class="text-3xl font-bold text-stone-900 dark:text-stone-100">Admin</h1>
		<p class="text-stone-600 dark:text-stone-400">Run rescrapes and watch progress.</p>
	</div>

	{#if error}
		<ErrorMessage message={error} />
	{/if}

	<Card>
		<div class="p-6 space-y-4">
			<h2 class="text-xl font-semibold text-stone-900 dark:text-stone-100">Full rescrape</h2>
			<div class="flex flex-wrap gap-2">
				<button class="px-4 py-2 rounded-lg bg-orange-600 text-white hover:bg-orange-700" disabled={starting} onclick={() => startJob('events')}>Events</button>
				<button class="px-4 py-2 rounded-lg bg-orange-600 text-white hover:bg-orange-700" disabled={starting} onclick={() => startJob('bots')}>Bots</button>
				<button class="px-4 py-2 rounded-lg bg-orange-600 text-white hover:bg-orange-700" disabled={starting} onclick={() => startJob('teams')}>Teams</button>
				<button class="px-4 py-2 rounded-lg bg-orange-600 text-white hover:bg-orange-700" disabled={starting} onclick={() => startJob('competitions')}>Competitions</button>
				<button class="px-4 py-2 rounded-lg bg-stone-700 text-white hover:bg-stone-800" disabled={starting} onclick={() => startJob('all')}>All</button>
			</div>
		</div>
	</Card>

	<Card>
		<div class="p-6 space-y-4">
			<h2 class="text-xl font-semibold text-stone-900 dark:text-stone-100">Rankings</h2>
			<div class="flex flex-col md:flex-row gap-2">
				<input
					class="w-full md:w-40 px-4 py-3 border border-stone-300 dark:border-stone-600 rounded-lg bg-white dark:bg-stone-800 text-stone-900 dark:text-stone-100"
					placeholder="Year (YYYY)"
					bind:value={rankingsYear}
				/>
				<button
					class="px-4 py-3 rounded-lg bg-orange-600 text-white hover:bg-orange-700"
					disabled={starting || !rankingsYear.trim()}
					onclick={() => startRankings(false)}
				>
					Scrape rankings
				</button>
				<button
					class="px-4 py-3 rounded-lg bg-stone-700 text-white hover:bg-stone-800"
					disabled={starting || !rankingsYear.trim()}
					onclick={() => startRankings(true)}
				>
					Rankings + ranked bots
				</button>
			</div>
			<p class="text-sm text-stone-600 dark:text-stone-400">
				“Rankings + ranked bots” fetches the rankings pages for the year, then scrapes every bot listed.
			</p>
		</div>
	</Card>

	<Card>
		<div class="p-6 space-y-4">
			<h2 class="text-xl font-semibold text-stone-900 dark:text-stone-100">Scrape a specific RCE URL</h2>
			<div class="flex flex-col md:flex-row gap-2">
				<input
					class="flex-1 px-4 py-3 border border-stone-300 dark:border-stone-600 rounded-lg bg-white dark:bg-stone-800 text-stone-900 dark:text-stone-100"
					placeholder="https://www.robotcombatevents.com/groups/10125 or /resources/22499"
					bind:value={url}
				/>
				<button class="px-4 py-3 rounded-lg bg-orange-600 text-white hover:bg-orange-700" disabled={starting || !url.trim()} onclick={scrapeURL}>
					Run
				</button>
			</div>
		</div>
	</Card>

	{#if starting && !job}
		<LoadingSpinner text="Starting job..." />
	{/if}

	{#if job}
		<Card>
			<div class="p-6 space-y-3">
				<div class="flex items-center justify-between">
					<h2 class="text-xl font-semibold text-stone-900 dark:text-stone-100">Job</h2>
					<div class="flex items-center gap-3">
						<span class="text-sm text-stone-600 dark:text-stone-400">{job.kind} • {job.state}</span>
						{#if job.state === 'running'}
							<button
								class="px-3 py-1.5 rounded-lg bg-stone-200 dark:bg-stone-800 text-stone-900 dark:text-stone-100 hover:bg-stone-300 dark:hover:bg-stone-700"
								onclick={cancelCurrentJob}
							>
								Cancel
							</button>
						{/if}
					</div>
				</div>

				{#if job.total > 0}
					<div class="space-y-2">
						<div class="h-2 w-full bg-stone-200 dark:bg-stone-800 rounded-full overflow-hidden">
							<div class="h-full bg-orange-600" style="width: {progress}%"></div>
						</div>
						<div class="text-sm text-stone-600 dark:text-stone-400">
							{job.done}/{job.total} done • {job.failed} failed
						</div>
					</div>
				{/if}

				{#if job.current}
					<div class="text-sm text-stone-700 dark:text-stone-300">Current: {job.current}</div>
				{/if}

				{#if job.logs?.length}
					<div class="rounded-lg border border-stone-200 dark:border-stone-800 bg-stone-50 dark:bg-stone-900 p-3 max-h-64 overflow-auto">
						{#each job.logs as line}
							<div class="text-xs text-stone-700 dark:text-stone-300 font-mono whitespace-pre-wrap">{line}</div>
						{/each}
					</div>
				{/if}
			</div>
		</Card>
	{/if}

	<Card>
		<div class="p-6 space-y-3">
			<h2 class="text-xl font-semibold text-stone-900 dark:text-stone-100">Recent runs</h2>
			{#if loadingRecent}
				<p class="text-sm text-stone-600 dark:text-stone-400">Loading…</p>
			{:else if recentJobs.length === 0}
				<p class="text-sm text-stone-600 dark:text-stone-400">No runs yet.</p>
			{:else}
				<div class="space-y-2">
					{#each recentJobs as j}
						<button
							class="w-full text-left rounded-lg border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 p-3 hover:bg-stone-50 dark:hover:bg-stone-800"
							onclick={() => {
								job = j;
								poll();
							}}
						>
							<div class="flex items-center justify-between">
								<div class="text-sm font-medium text-stone-900 dark:text-stone-100">{j.kind}</div>
								<div class="text-xs text-stone-600 dark:text-stone-400">{j.state}</div>
							</div>
							<div class="text-xs text-stone-600 dark:text-stone-400">{j.done}/{j.total} done • {j.failed} failed</div>
						</button>
					{/each}
				</div>
			{/if}
		</div>
	</Card>
</div>

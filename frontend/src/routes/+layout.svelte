<script lang="ts">
	import '../app.css';
	
	let { children } = $props();
	
	let darkMode = $state(false);
	
	$effect(() => {
		if (typeof window !== 'undefined') {
			const stored = localStorage.getItem('darkMode');
			darkMode = stored === 'true' || (!stored && window.matchMedia('(prefers-color-scheme: dark)').matches);
			updateDarkMode();
		}
	});
	
	function updateDarkMode() {
		if (darkMode) {
			document.documentElement.classList.add('dark');
		} else {
			document.documentElement.classList.remove('dark');
		}
		localStorage.setItem('darkMode', String(darkMode));
	}
	
	function toggleDarkMode() {
		darkMode = !darkMode;
		updateDarkMode();
	}
</script>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900 transition-colors">
	<nav class="bg-white dark:bg-gray-800 shadow-lg border-b border-gray-200 dark:border-gray-700">
		<div class="mx-auto px-4 sm:px-6 lg:px-8">
			<div class="flex justify-between h-16">
				<div class="flex items-center space-x-8">
					<a href="/" class="flex items-center space-x-2">
						<span class="text-2xl font-bold text-blue-600 dark:text-blue-400">Robot Registry</span>
					</a>
					<div class="hidden md:flex space-x-4">
						<a href="/events" class="px-3 py-2 rounded-md text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700">Events</a>
						<a href="/bots" class="px-3 py-2 rounded-md text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700">Bots</a>
						<a href="/teams" class="px-3 py-2 rounded-md text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700">Teams</a>
						<a href="/rankings" class="px-3 py-2 rounded-md text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700">Rankings</a>
					</div>
				</div>
				<div class="flex items-center space-x-4">
					<a href="/search" class="text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
						</svg>
					</a>
					<button
						on:click={toggleDarkMode}
						class="p-2 rounded-md text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
						aria-label="Toggle dark mode"
					>
						{#if darkMode}
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"/>
							</svg>
						{:else}
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/>
							</svg>
						{/if}
					</button>
				</div>
			</div>
		</div>
	</nav>

	<main class="mx-auto px-4 lg:px-8 py-8">
		{@render children()}
	</main>

	<footer class="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 mt-12">
		<div class="mx-auto px-4 lg:px-8 py-8">
			<p class="text-center text-gray-600 dark:text-gray-400 text-sm">
				Data sourced from Robot Combat Events
			</p>
		</div>
	</footer>
</div>

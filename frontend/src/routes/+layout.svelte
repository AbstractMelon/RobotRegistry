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


<div class="min-h-screen flex flex-col bg-stone-50 dark:bg-stone-950 text-stone-900 dark:text-stone-100 transition-colors">
	<nav class="bg-white dark:bg-stone-900 border-b border-stone-200/70 dark:border-stone-800">
		<div class="mx-auto px-4 sm:px-6 lg:px-8">
			<div class="flex justify-between h-16">
				<div class="flex items-center space-x-8">
					<a href="/" class="flex items-center gap-3">
						<img src="/logo.svg" alt="Robot Registry" class="h-7 w-auto" />
					</a>
					<div class="hidden md:flex space-x-4">
						<a href="/events" class="px-3 py-2 rounded-lg text-sm font-medium text-stone-700 dark:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-800">Events</a>
						<a href="/bots" class="px-3 py-2 rounded-lg text-sm font-medium text-stone-700 dark:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-800">Bots</a>
						<a href="/teams" class="px-3 py-2 rounded-lg text-sm font-medium text-stone-700 dark:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-800">Teams</a>
						<a href="/rankings" class="px-3 py-2 rounded-lg text-sm font-medium text-stone-700 dark:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-800">Rankings</a>
					</div>
				</div>
				<div class="flex items-center space-x-4">
					<a href="/search" class="rounded-lg p-2 text-stone-700 dark:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-800" aria-label="Search">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
						</svg>
					</a>
					<button
						onclick={toggleDarkMode}
						class="p-2 rounded-lg text-stone-700 dark:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-800"
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

	<main class="mx-auto w-full flex-1 px-4 lg:px-8 py-8">
		{@render children()}
	</main>

	<footer class="mt-auto bg-white dark:bg-stone-900 border-t border-stone-200/70 dark:border-stone-800">
		<div class="mx-auto px-4 lg:px-8 py-8">
			<p class="text-center text-stone-600 dark:text-stone-400 text-sm">
				Data sourced from Robot Combat Events
			</p>
		</div>
	</footer>
</div>

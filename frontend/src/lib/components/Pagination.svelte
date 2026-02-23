<script lang="ts">
	export let currentPage: number;
	export let totalPages: number;
	export let onPageChange: (page: number) => void;

	function goToPage(page: number) {
		if (page >= 1 && page <= totalPages) {
			onPageChange(page);
		}
	}

	$: pages = Array.from({ length: Math.min(totalPages, 7) }, (_, i) => {
		if (totalPages <= 7) return i + 1;
		if (currentPage <= 4) return i + 1;
		if (currentPage >= totalPages - 3) return totalPages - 6 + i;
		return currentPage - 3 + i;
	});
</script>

<div class="flex items-center justify-center space-x-2 mt-6">
	<button
		on:click={() => goToPage(currentPage - 1)}
		disabled={currentPage === 1}
		class="px-3 py-2 rounded-lg border border-stone-300 dark:border-stone-600 
		       bg-white dark:bg-stone-800 text-stone-700 dark:text-stone-300
		       hover:bg-stone-50 dark:hover:bg-stone-700
		       disabled:opacity-50 disabled:cursor-not-allowed"
	>
		Previous
	</button>

	{#each pages as page}
		<button
			on:click={() => goToPage(page)}
			class="px-3 py-2 rounded-lg border 
			       {page === currentPage 
			         ? 'bg-orange-600 border-orange-600 text-white' 
			         : 'border-stone-300 dark:border-stone-600 bg-white dark:bg-stone-800 text-stone-700 dark:text-stone-300 hover:bg-stone-50 dark:hover:bg-stone-700'}"
		>
			{page}
		</button>
	{/each}

	<button
		on:click={() => goToPage(currentPage + 1)}
		disabled={currentPage === totalPages}
		class="px-3 py-2 rounded-lg border border-stone-300 dark:border-stone-600 
		       bg-white dark:bg-stone-800 text-stone-700 dark:text-stone-300
		       hover:bg-stone-50 dark:hover:bg-stone-700
		       disabled:opacity-50 disabled:cursor-not-allowed"
	>
		Next
	</button>
</div>

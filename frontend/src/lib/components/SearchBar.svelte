<script lang="ts">
	import { Search } from 'lucide-svelte';
	import { createEventDispatcher } from 'svelte';

	export let value = '';
	export let placeholder = 'Search...';

	const dispatch = createEventDispatcher();

	function handleInput(e: Event) {
		const target = e.target as HTMLInputElement;
		value = target.value;
		dispatch('input', value);
	}
</script>

<div class="relative w-full">
	<div class="absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none">
		<Search class="w-5 h-5 text-gray-400 dark:text-gray-500" />
	</div>
	<input
		type="text"
		{value}
		{placeholder}
		on:input={handleInput}
		class="block w-full pl-10 pr-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg 
		       bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 
		       focus:ring-2 focus:ring-blue-500 focus:border-transparent
		       placeholder-gray-400 dark:placeholder-gray-500"
	/>
</div>

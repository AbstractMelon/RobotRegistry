<script lang="ts">
	export let text = 'Loading...';
</script>

<div class="flex items-center justify-center p-8">
	<div class="flex flex-col items-center space-y-4">
		<div class="w-12 h-12 border-4 border-orange-200 dark:border-orange-950 border-t-orange-600 dark:border-t-orange-400 rounded-full animate-spin"></div>
		<p class="text-stone-600 dark:text-stone-400 text-sm">{text}</p>
	</div>
</div>

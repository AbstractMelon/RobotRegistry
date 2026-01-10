<script lang="ts">
	export let text = 'Loading...';
</script>

<div class="flex items-center justify-center p-8">
	<div class="flex flex-col items-center space-y-4">
		<div class="w-12 h-12 border-4 border-blue-200 dark:border-blue-900 border-t-blue-600 dark:border-t-blue-400 rounded-full animate-spin"></div>
		<p class="text-gray-600 dark:text-gray-400 text-sm">{text}</p>
	</div>
</div>

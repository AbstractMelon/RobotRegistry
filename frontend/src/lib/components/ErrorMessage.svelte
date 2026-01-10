<script lang="ts">
	export let message: string;
</script>

<div class="flex items-center justify-center p-8">
	<div class="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-6 max-w-md">
		<p class="text-red-800 dark:text-red-200 text-center">{message}</p>
	</div>
</div>

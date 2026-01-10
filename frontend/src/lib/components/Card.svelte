<script lang="ts">
	export let href: string = '';
	export let hoverable: boolean = true;

	const baseClasses = 'bg-white dark:bg-gray-800 rounded-lg shadow-md overflow-hidden border border-gray-200 dark:border-gray-700';
	const hoverClasses = hoverable ? 'hover:shadow-lg transition-shadow duration-200' : '';
	
	$: classes = `${baseClasses} ${hoverClasses}`;
</script>

{#if href}
	<a {href} class={classes}>
		<slot />
	</a>
{:else}
	<div class={classes}>
		<slot />
	</div>
{/if}

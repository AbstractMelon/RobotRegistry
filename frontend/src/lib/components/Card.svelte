<script lang="ts">
	export let href: string = '';
	export let hoverable: boolean = true;

	const baseClasses =
		'bg-white dark:bg-stone-900 rounded-lg overflow-hidden border border-stone-200/80 dark:border-stone-800 shadow-sm';
	const hoverClasses = hoverable ? 'hover:shadow-md transition-shadow duration-200' : '';
	
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

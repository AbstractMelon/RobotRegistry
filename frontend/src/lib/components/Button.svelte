<script lang="ts">
	export let href: string = '';
	export let variant: 'primary' | 'secondary' | 'ghost' = 'primary';
	export let size: 'sm' | 'md' | 'lg' = 'md';

	const baseClasses = 'inline-flex items-center justify-center rounded-lg font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2';
	
	const variantClasses = {
		primary:
			'bg-orange-600 text-white hover:bg-orange-700 focus:ring-orange-500 focus:ring-offset-stone-50 dark:bg-orange-500 dark:hover:bg-orange-600 dark:focus:ring-offset-stone-950',
		secondary:
			'bg-stone-200 text-stone-900 hover:bg-stone-300 focus:ring-stone-500 focus:ring-offset-stone-50 dark:bg-stone-800 dark:text-stone-100 dark:hover:bg-stone-700 dark:focus:ring-offset-stone-950',
		ghost:
			'bg-transparent text-stone-700 hover:bg-stone-100 focus:ring-stone-500 focus:ring-offset-stone-50 dark:text-stone-200 dark:hover:bg-stone-800 dark:focus:ring-offset-stone-950'
	};

	const sizeClasses = {
		sm: 'px-3 py-1.5 text-sm',
		md: 'px-4 py-2 text-base',
		lg: 'px-6 py-3 text-lg'
	};

	$: classes = `${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]}`;
</script>

{#if href}
	<a {href} class={classes}>
		<slot />
	</a>
{:else}
	<button class={classes} on:click>
		<slot />
	</button>
{/if}

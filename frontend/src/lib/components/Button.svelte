<script lang="ts">
	export let href: string = '';
	export let variant: 'primary' | 'secondary' | 'ghost' = 'primary';
	export let size: 'sm' | 'md' | 'lg' = 'md';

	const baseClasses = 'inline-flex items-center justify-center rounded-lg font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2';
	
	const variantClasses = {
		primary: 'bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500 dark:bg-blue-500 dark:hover:bg-blue-600',
		secondary: 'bg-gray-200 text-gray-900 hover:bg-gray-300 focus:ring-gray-500 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600',
		ghost: 'bg-transparent text-gray-700 hover:bg-gray-100 focus:ring-gray-500 dark:text-gray-300 dark:hover:bg-gray-800'
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

<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Secure Package Registry</title>
</svelte:head>

<div class="app">
	<header>
		<div class="header-content">
			<h1>🔒 Secure Package Registry</h1>
		</div>
	</header>
	
	<main>
		{@render children()}
	</main>
	
	<footer>
		<p>Secure Package Registry Dashboard</p>
	</footer>
</div>

<style>
	:global(*, *::before, *::after) {
		box-sizing: border-box;
	}
	
	:global(html, body) {
		margin: 0;
		padding: 0;
		height: 100%;
	}
	
	.app {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
	}
	
	header {
		background: linear-gradient(135deg, #1976d2 0%, #1565c0 100%);
		color: white;
		padding: 1rem 2rem;
		box-shadow: 0 2px 4px rgba(0,0,0,0.1);
	}
	
	.header-content {
		max-width: 800px;
		margin: 0 auto;
	}
	
	header h1 {
		margin: 0;
		font-size: 1.5rem;
		font-weight: 600;
	}
	
	main {
		flex: 1;
		padding: 2rem;
	}
	
	footer {
		background: #263238;
		color: #90a4ae;
		padding: 1rem;
		text-align: center;
		font-size: 0.875rem;
	}
	
	footer p {
		margin: 0;
	}
</style>

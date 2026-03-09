<script lang="ts">
	export let isDark = true;
	export let onToggleTheme: () => void;
</script>

<header>
	<div class="logo">SPR</div>
	<div class="header-right">
		<button class="theme-toggle" on:click={onToggleTheme}>
			{isDark ? 'Light' : 'Dark'}
		</button>
		<a href="/landing" class="nav-link">Get Started</a>
	</div>
</header>

<style>
	header {
		padding: 1.5rem 2rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
		position: sticky;
		top: 0;
		background: var(--bg-primary);
		border-bottom: 1px solid var(--border);
		z-index: 100;
		backdrop-filter: blur(10px);
	}

	.logo {
		font-size: 2rem;
		font-weight: 700;
		color: var(--accent);
	}

	.header-right {
		display: flex;
		align-items: center;
		gap: 1.5rem;
	}

	.theme-toggle {
		background: transparent;
		border: none;
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--text-secondary);
		cursor: pointer;
		transition: color 0.3s;
	}

	.theme-toggle:hover {
		color: var(--accent);
	}

	.nav-link {
		color: var(--text-secondary);
		text-decoration: none;
		font-size: 0.875rem;
		font-weight: 500;
		transition: color 0.3s;
	}

	.nav-link:hover {
		color: var(--accent);
	}
</style>

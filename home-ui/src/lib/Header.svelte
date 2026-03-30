<script lang="ts">
	// import { browser } from '$app/environment';
	// import { isLoggedIn } from './utils/auth';

	export let isDark = true;
	export let onToggleTheme: () => void;

	import { PUBLIC_DASHBOARD_BASE_URL } from '$env/static/public';
</script>

<header>
	<div class="header-left">
		<div class="logo">
			<a class="clear-a-stylings" href="/">SPR</a>
		</div>
		<a href="{PUBLIC_DASHBOARD_BASE_URL}/docs" class="nav-link">Docs</a>
		<a href="{PUBLIC_DASHBOARD_BASE_URL}/pricing" class="nav-link">Pricing</a>
	</div>
	<div class="header-right">
		<button class="theme-toggle" onclick={onToggleTheme}>
			{isDark ? 'Light' : 'Dark'}
		</button>
		<a href="{PUBLIC_DASHBOARD_BASE_URL}/" class="navbar-button"> Get Started </a>
	</div>
</header>

<style>
	header {
		padding: 0.75rem 2.5rem;
		display: flex;
		/* justify-content: space-between; */
		align-items: center;
		position: sticky;
		top: 0;
		background: var(--bg-primary);
		border-bottom: 1px solid var(--border);
		z-index: 100;
		backdrop-filter: blur(10px);
	}

	.navbar-button {
		background: var(--accent);
		color: var(--bg-primary);
		text-decoration: none;
		padding: 0.5rem 1.75rem;
		border-radius: 6px;
		font-size: 0.95rem;
		font-weight: 500;
		transition: background 0.2s;
		font-size: 0.875rem;
		font-weight: 500;
		transition: color 0.3s;
	}

	.clear-a-stylings {
		text-decoration: none;
		color: inherit;
	}

	.logo {
		font-size: 2rem;
		font-weight: 700;
		color: var(--accent);
	}

	.header-right {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-left: auto;
	}

	.header-left {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.theme-toggle {
		background: transparent;
		border: none;
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--wb-bg);

		background: var(--wb-bg-invert);
		text-decoration: none;
		padding: 0.5rem;
		border-radius: 6px;
		font-size: 0.95rem;
		font-weight: 500;
		transition: background 0.2s;
		font-size: 0.875rem;
		font-weight: 500;
		transition: color 0.3s;
		cursor: pointer;
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

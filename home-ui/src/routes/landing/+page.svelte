<script lang="ts">
	// Landing page - sits between an ad and the main homepage
	// Goal: get someone curious enough to click through
	// Pre rendering line in +page.ts
</script>

<div class="page">
	<header>
		<div class="logo">SPR</div>
		<a href="/home" class="home-link">Visit homepage &rarr;</a>
	</header>

	<main>
		<section class="hero">
			<div class="tag">Now in early access</div>

			<h1>Your packages, verified.</h1>

			<p>
				Supply chain attacks cost UK businesses billions every year. SPR catches malicious npm, Go,
				Cargo and PyPI packages before they ever reach your codebase — using EBPF behavioral
				analysis and source verification.
			</p>

			<a href="/home" class="cta-button">See how it works &rarr;</a>
		</section>

		<section class="proof">
			<div class="stat">
				<span class="number">704,102</span>
				<span class="label">Malicious packages found in npm since 2019</span>
			</div>
			<div class="divider"></div>
			<div class="stat">
				<span class="number">£3.29M</span>
				<span class="label">Average UK data breach cost</span>
			</div>
			<div class="divider"></div>
			<div class="stat">
				<span class="number">4</span>
				<span class="label">Ecosystems covered — NPM, Go, Cargo, PyPI</span>
			</div>
		</section>
	</main>

	<footer>
		<p>SPR &copy; 2026</p>
	</footer>
</div>

<style>
	:global(*, *::before, *::after) {
		box-sizing: border-box;
		margin: 0;
		padding: 0;
	}

	:global(body) {
		font-family:
			system-ui,
			-apple-system,
			sans-serif;
		background: #fff;
		color: #111;
		line-height: 1.6;
	}

	.page {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
	}

	/* Header */
	header {
		padding: 1.5rem 0.75rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.logo {
		font-size: 3.5rem;
		font-weight: 700;
		color: #1d4ed8;
	}

	.home-link {
		font-size: 0.875rem;
		color: #6b7280;
		text-decoration: none;
		transition: color 0.2s;
	}

	.home-link:hover {
		color: #1d4ed8;
	}

	/* Main */
	main {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 4rem 0.5rem;
		gap: 4rem;
	}

	/* Hero */
	.hero {
		max-width: 860px;
		text-align: center;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 1.25rem;
	}

	.tag {
		display: inline-block;
		background: #eff6ff;
		color: #1d4ed8;
		border: 1px solid #bfdbfe;
		padding: 0.3rem 0.9rem;
		border-radius: 20px;
		font-size: 0.8rem;
		font-weight: 500;
	}

	h1 {
		font-size: 2.8rem;
		font-weight: 700;
		line-height: 1.15;
		color: #111827;
		letter-spacing: -0.5px;
	}

	p {
		font-size: 1.05rem;
		color: #4b5563;
		line-height: 1.7;
	}

	.cta-button {
		display: inline-block;
		background: #1d4ed8;
		color: white;
		text-decoration: none;
		padding: 0.8rem 1.75rem;
		border-radius: 6px;
		font-size: 0.95rem;
		font-weight: 500;
		margin-top: 0.5rem;
		transition: background 0.2s;
	}

	.cta-button:hover {
		background: #1e40af;
	}

	/* Stats */
	.proof {
		display: flex;
		align-items: center;
		gap: 3rem;
	}

	.stat {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: 0.3rem;
	}

	.number {
		font-size: 1.75rem;
		font-weight: 700;
		color: #111827;
	}

	.label {
		font-size: 0.8rem;
		color: #6b7280;
		max-width: 130px;
		line-height: 1.4;
	}

	.divider {
		width: 1px;
		height: 40px;
		background: #e5e7eb;
		flex-shrink: 0;
	}

	/* Footer */
	footer {
		padding: 1.5rem 2rem;
		display: flex;
		justify-content: center;
		align-items: center;
	}

	footer p {
		font-size: 0.8rem;
		color: #9ca3af;
	}
</style>

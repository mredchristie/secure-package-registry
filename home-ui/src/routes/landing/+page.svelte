<script lang="ts">
	// Landing page - entry point from ads / external links
	// Simpler, faster-loading page; Header and global styles come from +layout.svelte
</script>

<main>
	<section class="hero">
		<div class="tag">Now in early access</div>

		<h1>Your packages, verified.</h1>

		<p>
			Supply chain attacks cost UK businesses billions every year. SPR catches malicious npm, Go,
			Cargo and PyPI packages before they ever reach your codebase — using EBPF behavioral analysis
			and source verification.
		</p>

		<a href="/" class="cta-button">See how it works &rarr;</a>
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

<style>
	main {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 4rem 0.5rem;
		gap: 4rem;
	}

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
		background: rgba(29, 78, 216, 0.08);
		color: var(--accent);
		border: 1px solid var(--card-border);
		padding: 0.3rem 0.9rem;
		border-radius: 20px;
		font-size: 0.8rem;
		font-weight: 500;
	}

	h1 {
		font-size: 2.8rem;
		font-weight: 700;
		line-height: 1.15;
		color: var(--text-primary);
		letter-spacing: -0.5px;
	}

	p {
		font-size: 1.05rem;
		color: var(--text-secondary);
		line-height: 1.7;
	}

	.cta-button {
		display: inline-block;
		background: var(--accent);
		color: var(--bg-primary);
		text-decoration: none;
		padding: 0.8rem 1.75rem;
		border-radius: 6px;
		font-size: 0.95rem;
		font-weight: 500;
		margin-top: 0.5rem;
		transition: background 0.2s;
	}

	.cta-button:hover {
		background: var(--accent-hover);
	}

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
		color: var(--text-primary);
	}

	.label {
		font-size: 0.8rem;
		color: var(--text-secondary);
		max-width: 130px;
		line-height: 1.4;
	}

	.divider {
		width: 1px;
		height: 40px;
		background: var(--border);
		flex-shrink: 0;
	}

	/* global footer has border-top; landing page uses a lighter treatment */
	footer {
		border-top: none;
		padding: 1.5rem 2rem;
		display: flex;
		justify-content: center;
		align-items: center;
	}

	footer p {
		font-size: 0.8rem;
	}
</style>

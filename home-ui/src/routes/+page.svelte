<script lang="ts">
	import { onMount } from 'svelte';

	let showIncidents = false;

	// real supply chain incidents for the problem section
	const incidents = [
		{
			severity: 'CVSS 10.0',
			title: 'XZ Backdoor',
			description:
				'Multi-year state-sponsored operation discovered by chance when a developer noticed a 500ms SSH delay',
			date: 'March 2024',
			animation: 'slide-in-left',
			link: 'https://tukaani.org/xz/'
		},
		{
			severity: '754+ packages',
			title: 'Shai Hulud',
			description:
				'Self-propagating worm that stole credentials and exposed 33K secrets across GitHub Actions',
			date: 'November 2025',
			animation: 'slide-in-right',
			link: 'https://www.aikido.dev/blog/github-actions-incident-shai-hulud-supply-chain-attack'
		},
		{
			severity: '6 months',
			title: 'Notepad++ Compromise',
			description:
				'Infrastructure compromised via hosting provider for half a year before detection',
			date: '2025',
			animation: 'slide-in-left',
			link: 'https://notepad-plus-plus.org/news/hijacked-incident-info-update/'
		},
		{
			severity: 'First AI Attack',
			title: 'S1ngularity',
			description:
				'AI-weaponized supply chain attack using Claude and Gemini with --dangerously-skip-permissions flags',
			date: 'August 2025',
			animation: 'slide-in-right',
			link: 'https://nx.dev/blog/s1ngularity-postmortem'
		}
	];

	const headlineStats = [
		{ number: '512K', label: 'Malicious packages in 2024', delay: '' },
		{ number: '+156%', label: 'Year-over-year growth', delay: 'delay-1' },
		{ number: '$4.88M', label: 'Average breach cost', delay: 'delay-2' },
		{ number: '$46B', label: 'Total global damage', delay: 'delay-3' }
	];

	const downloadStats = [
		{ count: '4.5 trillion', label: 'npm downloads per year', delay: '' },
		{ count: '530 billion', label: 'PyPI downloads per year', delay: 'delay-1' },
		{ count: '150', label: 'Average dependencies per app', delay: 'delay-2' }
	];

	const steps = [
		{
			number: '01',
			title: 'Package Upload',
			description:
				'Request any npm package for verification. SPR fetches it from the public registry.',
			animation: 'slide-in-left'
		},
		{
			number: '02',
			title: 'Reproducible Builds',
			description: 'Built from source to ensure reprocdiucability.',
			animation: 'slide-in-right'
		},
		{
			number: '03',
			title: 'Diff Check',
			description:
				'Compare the published package to the GitHub source. Catches backdoor injections like the event-stream attack.',
			animation: 'slide-in-right'
		},
		{
			number: '04',
			title: 'EBPF Monitor',
			description:
				'Run in an isolated Podman container with kernel-level monitoring. Watches network calls, file access, and process spawning.',
			animation: 'slide-in-left'
		},
		{
			number: '05',
			title: 'Behavioral Analysis',
			description:
				'Execute package in sandboxed environment. Monitor runtime behavior to detect malicious activity patterns.',
			animation: 'slide-in-right'
		},
		{
			number: '06',
			title: 'Verified',
			description: 'Package is added to your private registry. Safe to install.',
			animation: 'slide-in-left'
		}
	];

	const marketCards = [
		{
			label: 'Market Size',
			number: '$1.95B – $5.53B',
			sublabel: 'Supply chain security market',
			animation: 'slide-in-left'
		},
		{
			label: 'Growth Rate',
			number: '10.9% – 12.8%',
			sublabel: 'CAGR through 2030',
			animation: 'slide-in-right'
		}
	];

	onMount(() => {
		const scrollObserver = new IntersectionObserver(
			(entries) => {
				entries.forEach((e) => e.target.classList.toggle('visible', e.isIntersecting));
			},
			{ threshold: 0.15 }
		);

		document.querySelectorAll('.animate').forEach((el) => scrollObserver.observe(el));

		return () => scrollObserver.disconnect();
	});
</script>

<div class="page">
	<!-- hero - full height, centred, animates on load -->
	<section class="hero">
		<div class="container">
			<div class="hero-badge">Trust but Verify</div>
			<h1 class="hero-title">Secure Package Registry</h1>
			<p class="hero-subtitle">
				Verified package registry for npm, PyPI, Go, and Cargo — blocking malicious code before it
				reaches your codebase
			</p>
			<a href="/landing" class="cta-button">Request Early Access</a>
		</div>
	</section>

	<section class="section section-alt">
		<div class="container">
			<h2 class="section-title animate fade-in">The Scale is Staggering</h2>

			<div class="headline-stats">
				{#each headlineStats as { number, label, delay }}
					<div class="headline-stat animate zoom-in {delay}">
						<div class="headline-number">{number}</div>
						<div class="headline-label">{label}</div>
					</div>
				{/each}
			</div>

			<div class="download-stats">
				{#each downloadStats as { count, label, delay }, i}
					{#if i > 0}<div class="divider"></div>{/if}
					<div class="download-stat animate slide-in-bottom {delay}">
						<span class="download-count">{count}</span>
						<span class="download-label">{label}</span>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<section class="section">
		<div class="container">
			<h2 class="section-title animate slide-in-left">The Problem</h2>

			<p class="problem-intro animate fade-in">
				Supply chain attacks exploit the trust developers place in open-source packages. When you
				run <code>npm install</code>, malicious code can execute immediately—stealing credentials,
				injecting backdoors, or compromising your entire infrastructure. These aren't theoretical
				risks. Real attacks are happening right now.
			</p>

			<button
				class="toggle-button animate fade-in delay-1"
				on:click={() => (showIncidents = !showIncidents)}
			>
				{showIncidents ? 'Hide Examples ▲' : 'See Real Examples ▼'}
			</button>

			{#if showIncidents}
				<div class="incidents-grid">
					{#each incidents as { severity, title, description, date, link }}
						<a class="incident-card" href={link} target="_blank" rel="noopener noreferrer">
							<div class="severity-tag">{severity}</div>
							<h3>{title}</h3>
							<p>{description}</p>
							<span class="incident-date">{date}</span>
						</a>
					{/each}
				</div>
			{/if}
		</div>
	</section>

	<section class="section section-alt">
		<div class="container">
			<h2 class="section-title animate slide-in-right">Our Solution</h2>

			<div class="steps-grid">
				{#each steps as { number, title, description, animation }}
					<div class="step animate {animation}">
						<div class="step-number">{number}</div>
						<h3>{title}</h3>
						<p>{description}</p>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<section class="section">
		<div class="container">
			<h2 class="section-title animate fade-in">Market Opportunity</h2>

			<div class="market-cards">
				{#each marketCards as { label, number, sublabel, animation }}
					<div class="market-card animate {animation}">
						<div class="card-label">{label}</div>
						<div class="card-number">{number}</div>
						<div class="card-sublabel">{sublabel}</div>
					</div>
				{/each}
			</div>

			<div class="market-timing animate zoom-in">
				<h3>Why Now?</h3>
				<ul>
					<li>Attacks growing by a staggering <strong>+156% Year on Year</strong></li>
					<li>AI-powered attacks emerging and evolving</li>
					<li>Regulatory mandates taking effect (EO 14028, EU Cyber Resilience Act)</li>
					<li>No preventive solution exists</li>
				</ul>
			</div>
		</div>
	</section>

	<section class="cta-section">
		<div class="container">
			<h2 class="animate fade-in">Ready to secure your supply chain?</h2>
			<p class="animate fade-in delay-1">
				Join us at SPR to be the change protecting against the $46B attack problem
			</p>
			<a href="/landing" class="cta-button animate zoom-in delay-2">Get Started</a>
		</div>
	</section>

	<footer>
		<p>SPR &copy; 2026</p>
	</footer>
</div>

<style>
	@keyframes fadeInUp {
		from {
			opacity: 0;
			transform: translateY(30px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.hero {
		min-height: 90vh;
		display: flex;
		align-items: center;
		justify-content: center;
		text-align: center;
		padding: 4rem 2rem;
	}

	.hero-badge {
		display: inline-block;
		background: rgba(79, 195, 247, 0.1);
		color: var(--accent);
		border: 1px solid var(--accent);
		padding: 0.5rem 1.5rem;
		border-radius: 30px;
		font-size: 0.875rem;
		font-weight: 600;
		margin-bottom: 2rem;
		animation: fadeInUp 0.8s ease-out;
	}

	.hero-title {
		font-size: clamp(2.5rem, 8vw, 5rem);
		font-weight: 700;
		margin-bottom: 1.5rem;
		line-height: 1.1;
		animation: fadeInUp 0.8s ease-out 0.2s both;
	}

	.hero-subtitle {
		font-size: clamp(1rem, 2vw, 1.25rem);
		color: var(--text-secondary);
		max-width: 800px;
		margin: 0 auto 2.5rem;
		animation: fadeInUp 0.8s ease-out 0.4s both;
	}

	.cta-button {
		display: inline-block;
		background: var(--accent);
		color: var(--bg-primary);
		padding: 1rem 2.5rem;
		border-radius: 8px;
		text-decoration: none;
		font-weight: 600;
		font-size: 1.05rem;
		transition: all 0.3s;
		animation: fadeInUp 0.8s ease-out 0.6s both;
	}

	.cta-button:hover {
		background: var(--accent-hover);
		transform: translateY(-2px);
		box-shadow: 0 8px 20px rgba(0, 0, 0, 0.2);
	}

	.problem-intro {
		max-width: 800px;
		margin: 0 auto 2rem;
		text-align: center;
		font-size: 1.1rem;
		color: var(--text-secondary);
		line-height: 1.8;
	}

	.problem-intro code {
		background: rgba(79, 195, 247, 0.1);
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		font-family: 'Courier New', monospace;
		color: var(--accent);
		font-size: 1rem;
	}

	.toggle-button {
		display: block;
		margin: 2rem auto;
		background: transparent;
		border: 2px solid var(--accent);
		color: var(--accent);
		padding: 0.75rem 2rem;
		border-radius: 8px;
		font-size: 0.95rem;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.3s;
	}

	.toggle-button:hover {
		background: var(--accent);
		color: var(--bg-primary);
	}

	.incidents-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
		gap: 2rem;
		margin-top: 3rem;
	}
	a.incident-card {
		text-decoration: none;
		color: inherit;
		display: block;
	}

	.incident-card {
		background: var(--card-bg);
		border: 1px solid var(--card-border);
		border-radius: 12px;
		padding: 2rem;
		transition: all 0.3s;
		position: relative;
		overflow: hidden;
		animation: fadeInUp 0.4s ease-out both;
	}

	.incident-card::before {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		width: 4px;
		height: 100%;
		background: #ff6b6b;
		opacity: 0;
		transition: opacity 0.3s;
	}

	.incident-card:hover {
		transform: translateY(-8px);
		box-shadow: 0 12px 40px rgba(0, 0, 0, 0.1);
	}

	.incident-card:hover::before {
		opacity: 1;
	}

	.severity-tag {
		display: inline-block;
		background: rgba(255, 107, 107, 0.1);
		color: #ff6b6b;
		padding: 0.25rem 0.75rem;
		border-radius: 20px;
		font-size: 0.75rem;
		font-weight: 600;
		margin-bottom: 1rem;
	}

	.incident-card h3 {
		font-size: 1.5rem;
		margin-bottom: 0.75rem;
		color: var(--text-primary);
	}

	.incident-card p {
		color: var(--text-secondary);
		font-size: 0.95rem;
		line-height: 1.6;
		margin-bottom: 1rem;
	}

	.incident-date {
		font-size: 0.8rem;
		color: var(--text-secondary);
		font-weight: 500;
	}

	.headline-stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		gap: 3rem;
		margin-bottom: 4rem;
	}

	.headline-stat {
		text-align: center;
	}

	.headline-number {
		font-size: clamp(3rem, 8vw, 5rem);
		font-weight: 700;
		color: #ff6b6b;
		line-height: 1;
		margin-bottom: 0.5rem;
	}

	.headline-label {
		font-size: 1rem;
		color: var(--text-secondary);
	}

	.download-stats {
		display: flex;
		justify-content: center;
		align-items: center;
		gap: 3rem;
		flex-wrap: wrap;
		padding: 2rem 0;
	}

	.download-stat {
		text-align: center;
	}

	.download-count {
		display: block;
		font-size: 2rem;
		font-weight: 700;
		color: var(--accent);
		margin-bottom: 0.5rem;
	}

	.download-label {
		font-size: 0.9rem;
		color: var(--text-secondary);
	}

	.divider {
		width: 1px;
		height: 60px;
		background: var(--border);
	}

	.steps-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
		gap: 3rem;
		margin-top: 3rem;
	}

	.step {
		position: relative;
		padding-left: 5rem;
	}

	.step-number {
		position: absolute;
		left: 0;
		top: -0.5rem;
		font-size: 4rem;
		font-weight: 700;
		color: var(--accent);
		opacity: 0.15;
		line-height: 1;
	}

	.step h3 {
		font-size: 1.5rem;
		margin-bottom: 0.75rem;
		color: var(--text-primary);
	}

	.step p {
		color: var(--text-secondary);
		line-height: 1.7;
	}

	.market-cards {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
		gap: 3rem;
		margin-bottom: 4rem;
	}

	.market-card {
		text-align: center;
		padding: 2rem;
		background: var(--card-bg);
		border: 1px solid var(--card-border);
		border-radius: 12px;
	}

	.card-label {
		font-size: 1rem;
		color: var(--text-secondary);
		margin-bottom: 1rem;
	}

	.card-number {
		font-size: clamp(2rem, 5vw, 3rem);
		font-weight: 700;
		color: var(--accent);
		margin-bottom: 0.5rem;
	}

	.card-sublabel {
		font-size: 0.9rem;
		color: var(--text-secondary);
	}

	.market-timing {
		max-width: 600px;
		margin: 0 auto;
		text-align: center;
	}

	.market-timing h3 {
		font-size: 2rem;
		margin-bottom: 2rem;
		color: var(--text-primary);
	}

	.market-timing ul {
		list-style: none;
		text-align: left;
	}

	.market-timing li {
		padding: 1rem 0;
		border-bottom: 1px solid var(--border);
		color: var(--text-secondary);
		font-size: 1.05rem;
	}

	.market-timing li:last-child {
		border-bottom: none;
	}

	.cta-section {
		padding: 6rem 2rem;
		text-align: center;
		background: var(--bg-secondary);
	}

	.cta-section h2 {
		font-size: clamp(2rem, 5vw, 3rem);
		margin-bottom: 1rem;
	}

	.cta-section p {
		font-size: 1.1rem;
		color: var(--text-secondary);
		margin-bottom: 2.5rem;
	}

	@media (max-width: 768px) {
		.download-stats {
			flex-direction: column;
		}

		.divider {
			width: 60px;
			height: 1px;
		}

		.step {
			padding-left: 0;
			padding-top: 3rem;
		}

		.step-number {
			top: -1rem;
		}
	}
</style>

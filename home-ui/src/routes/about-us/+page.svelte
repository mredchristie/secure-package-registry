<script lang="ts">
	const lastUpdated = '2026-02-03';

	type SlideKey = 'Philosophies' | 'Values' | 'Team' | 'Impact' | 'Contact';

	interface Slide {
		key: SlideKey;
		title: string;
	}

	const slides: Slide[] = [
		{ key: 'Philosophies', title: 'Security philosophies' },
		{ key: 'Values', title: 'Core values' },
		{ key: 'Team', title: 'Meet the team' },
		{ key: 'Impact', title: 'Project impact' },
		{ key: 'Contact', title: 'Get in touch' }
	];

	let active: SlideKey = 'Philosophies';

	type AccordionItem = {
		title: string;
		summary: string;
		bullets?: string[];
	};

	const philosophy: AccordionItem[] = [
		{
			title: 'Preventive, not reactive',
			summary:
				'SPR aims to reduce software supply-chain risk by verifying packages before they are installed,' +
				' helping organisations prevent security issues before they enter their development environments.'
		},
		{
			title: 'Defence-in-depth',
			summary: 'No single control is sufficient. SPR layers multiple verification signals.',
			bullets: [
				'Reproducible builds',
				'Behavioural analysis in an isolated environment',
				'Source-to-release verification',
				'LLM-assisted diff review (supporting signal)',
				'Human review for high-risk cases'
			]
		},
		{
			title: 'Risk reduction, not a guarantee',
			summary:
				'We communicate coverage limits and residual risk clearly so teams can make informed decisions.',
			bullets: [
				'Transparent coverage limits',
				'Clear status indicators and rationale',
				'Encourages defence-in-depth in client organisations'
			]
		},
		{
			title: 'Internal security standards',
			summary: 'Operational security must meet or exceed the standards we expect from clients.',
			bullets: [
				'Zero-trust principles',
				'Isolation of build and analysis environments',
				'Mandatory review for critical cases',
				'Resilience and auditability'
			]
		}
	];

	let openIndex: number | null = 0;
	function toggle(i: number) {
		openIndex = openIndex === i ? null : i;
	}

	type Team = {
		role: string;
		name: string;
	};

	const team: Team[] = [
		{ role: 'Project Lead', name: '(Name)' },
		{ role: 'Backend', name: '(Name)' },
		{ role: 'Frontend', name: '(Name)' },
		{ role: 'Security & Verification', name: '(Name)' }
	];

	type ContactInfo = {
		label: string;
		value: string;
		href: string;
	};

	const contactInfo: ContactInfo[] = [
		{ label: 'Email', value: 'spr@example.com', href: 'mailto:spr@example.com' },
		{ label: 'Phone', value: '+123 456 789', href: 'tel:+123456789' }
	];

	// PascalCase types for clarity
	type AboutOverview = {
		title: string;
		content: string;
	};

	const aboutInfo: AboutOverview[] = [
		{
			title: 'Overview',
			content:
				'SPR (Secure Package Registry) is an alternative package registry that surfaces ' +
				'verification status so teams can make safer dependency choices.'
		}
	];

	type MissionItem = {
		title: string;
		content: string;
	};

	const mission: MissionItem[] = [
		{
			title: 'Mission',
			content:
				'SPR helps reduce software supply-chain risk by verifying packages before they reach developers. ' +
				'We focus on transparency: what was checked, what was not, and the resulting confidence signals. '
		},
		{
			title: 'Mission-muted',
			content:
				'SPR is designed to be drop-in compatible with existing package ' +
				'manager workflows (configured via registry settings).'
		}
	];

	type ValuesItem = {
		title: string;
		summary: string;
	};

	const values: ValuesItem[] = [
		{
			title: 'Values',
			summary:
				'We prioritise security, reliability, and developer usability. ' +
				'Our outputs are designed to be actionable and auditable.'
		}
	];

	type ImpactItem = {
		title: string;
		content: string;
	};

	const impact: ImpactItem[] = [
		{
			title: 'Project impact',
			content:
				'By verifying packages before they reach developers,' +
				' SPR helps reduce the risk of supply-chain attacks. ' +
				'Our transparent signals enable teams to make informed decisions and prioritise security' +
				' in their dependencies.'
		}
	];
</script>

<div class="about-page">
	<header class="hero">
		<p class="eyebrow">Secure Package Registry</p>
		<h1>About SPR</h1>
		<p class="lead">
			{#each aboutInfo as info}
				{info.content}
			{/each}
		</p>
	</header>

	<section class="card" aria-labelledby="mission-title">
		<div class="mission">
			<h2 id="mission-title">Mission</h2>
			{#each mission as m}
				<p>{m.content}</p>
			{/each}
		</div>
	</section>

	<section class="slideshow-container" aria-label="slides">
		<!-- Clickable titles -->
		<nav class="slide-tabs">
			{#each slides as s}
				<button
					type="button"
					class="tab"
					class:is-active={active === s.key}
					on:click={() => (active = s.key)}
				>
					{s.title}
				</button>
			{/each}
		</nav>
		<!-- Content for each slide -->
		<div class="slide-panel">
			{#if active === 'Philosophies'}
				<section class="card" aria-labelledby="philosophy-title">
					<div class="philosophy">
						<h2 id="philosophy-title">Security philosophy</h2>

						<div class="accordion" role="list">
							{#each philosophy as item, i}
								<div class="acc-item" role="listitem">
									<button
										type="button"
										class="acc-trigger"
										aria-expanded={openIndex === i}
										on:click={() => toggle(i)}
									>
										<span>{item.title}</span>
										<span class="chev" aria-hidden="true">{openIndex === i ? '–' : '+'}</span>
									</button>

									{#if openIndex === i}
										<div class="acc-panel">
											<p>{item.summary}</p>
											{#if item.bullets?.length}
												<ul>
													{#each item.bullets as b}
														<li>{b}</li>
													{/each}
												</ul>
											{/if}
										</div>
									{/if}
								</div>
							{/each}
						</div>
					</div>
				</section>
			{:else if active === 'Values'}
				<section class="card" aria-labelledby="values-title">
					<h2 id="values-title">Values</h2>
					{#each values as values}
						<p>{values.summary}</p>
					{/each}
				</section>
			{:else if active === 'Team'}
				<section class="card" aria-labelledby="team-title">
					<h2 id="team-title">Team</h2>
					<ul class="team-list">
						{#each team as member}
							<li>
								<span class="team-role">{member.role}</span>
								<span class="team-name">{member.name}</span>
							</li>
						{/each}
					</ul>
					<p class="muted">
						This project is built as part of a third-year Software Engineering group project.
					</p>
				</section>
			{:else if active === 'Impact'}
				<section class="card" aria-labelledby="impact-title">
					<h2 id="impact-title">Project impact</h2>
					{#each impact as i}
						<p>{i.content}</p>
					{/each}
				</section>
			{:else if active === 'Contact'}
				<section class="card contact-card" aria-labelledby="contact-title">
					<h2 id="contact-title">Contact</h2>
					<ul class="contact-list">
						{#each contactInfo as info}
							<li>
								<span class="contact-label">{info.label}</span>
								<a href={info.href}>{info.value}</a>
							</li>
						{/each}
					</ul>
					<p class="meta">Last updated: {lastUpdated}</p>
				</section>
			{/if}
		</div>
	</section>
	<footer>
		<p>SPR &copy; 2026</p>
	</footer>
</div>

<style>
	.about-page {
		max-width: 1080px;
		margin: 2.5rem auto 0;
		padding: 0 1.25rem 1.5rem;
		display: grid;
		gap: 1.25rem;
	}

	/* Shared surfaces */
	.hero,
	.card,
	.slideshow-container {
		background: var(--card-bg);
		border: 1px solid var(--card-border);
		border-radius: 16px;
		box-shadow: none;
	}

	.hero {
		padding: clamp(1.5rem, 3vw, 2.5rem);
	}

	.eyebrow {
		margin: 0;
		display: inline-block;
		background: color-mix(in srgb, var(--accent) 10%, transparent);
		color: var(--accent);
		border: 1px solid var(--card-border);
		padding: 0.3rem 0.9rem;
		border-radius: 999px;
		font-size: 0.8rem;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
	}

	h1 {
		margin: 0.6rem 0 0;
		font-size: clamp(2.1rem, 3vw, 2.6rem);
		font-weight: 750;
		line-height: 1.15;
		color: var(--text-primary);
		letter-spacing: -0.5px;
	}

	h2 {
		margin: 0 0 0.65rem;
		color: var(--text-primary);
		font-size: 1.25rem;
		font-weight: 700;
	}

	p {
		margin: 0;
		font-size: 1.02rem;
		color: var(--text-secondary);
		line-height: 1.7;
	}

	.lead {
		margin-top: 0.85rem;
		max-width: 70ch;
	}

	.card {
		padding: clamp(1.25rem, 2.6vw, 1.8rem);
		display: grid;
		gap: 0.65rem;
	}

	.muted,
	.meta {
		margin-top: 0.65rem;
		color: var(--text-secondary);
		font-size: 0.95rem;
	}

	/* Accordion */
	.accordion {
		display: grid;
		gap: 0.6rem;
		margin-top: 1rem;
	}

	.acc-item {
		border: 1px solid var(--border);
		border-radius: 12px;
		overflow: hidden;
		background: color-mix(in srgb, var(--bg-secondary) 60%, transparent);
	}

	.acc-trigger {
		width: 100%;
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		padding: 0.85rem 1rem;
		border: 0;
		background: transparent;
		text-align: left;
		font-weight: 650;
		color: var(--text-primary);
		cursor: pointer;
	}

	.chev {
		width: 22px;
		height: 22px;
		display: grid;
		place-items: center;
		border-radius: 50%;
		color: var(--accent);
		font-weight: 700;
	}

	.acc-panel {
		padding: 0 1rem 0.95rem;
		display: grid;
		gap: 0.65rem;
	}

	.acc-panel ul,
	.team-list {
		margin: 0;
		padding-left: 1.1rem;
		color: var(--text-secondary);
		line-height: 1.65;
		display: grid;
		gap: 0.25rem;
	}

	/* Team */
	.team-list li {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.65rem 0.9rem;
		border: 1px solid var(--border);
		border-radius: 10px;
		font-size: 0.95rem;
		background: color-mix(in srgb, var(--bg-secondary) 60%, transparent);
	}

	.team-role {
		font-weight: 650;
		color: var(--text-primary);
	}

	.team-name {
		color: var(--text-secondary);
	}

	/* Slides */
	.slideshow-container {
		overflow: hidden;
		display: grid;
		grid-template-rows: auto 1fr;
	}

	.slide-tabs {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
		padding: 0.75rem 1rem;
		border-bottom: 1px solid var(--border);
	}

	.tab {
		border: 1px solid transparent;
		background: transparent;
		padding: 0.6rem 1rem;
		border-radius: 8px;
		cursor: pointer;
		font-weight: 600;
		font-size: 0.92rem;
		color: var(--text-secondary);
		transition:
			background 0.2s,
			border-color 0.2s,
			color 0.2s;
	}

	.tab:hover {
		background: color-mix(in srgb, var(--accent) 8%, transparent);
		border-color: var(--border);
		color: var(--text-primary);
	}

	.tab.is-active {
		background: color-mix(in srgb, var(--accent) 14%, transparent);
		border-color: var(--border);
		color: var(--text-primary);
	}

	.slide-panel {
		padding: 1.5rem;
		min-height: 0;
		display: block;
	}

	.slide-panel > section {
		width: 100%;
		border: 0;
		box-shadow: none;
		padding: 0;
		background: transparent;
	}

	/* Contact */
	.contact-list {
		list-style: none;
		padding: 0;
		margin: 0.2rem 0 0;
		display: grid;
		gap: 0.8rem;
	}

	.contact-list li {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		padding: 0.75rem 0.9rem;
		border: 1px solid var(--border);
		border-radius: 10px;
		background: color-mix(in srgb, var(--bg-secondary) 60%, transparent);
	}

	.contact-label {
		color: var(--text-primary);
		font-weight: 650;
	}

	.contact-list a {
		color: var(--accent);
		text-decoration: none;
		font-weight: 600;
	}

	.contact-list a:hover {
		text-decoration: underline;
		color: var(--accent-hover);
	}

	/* Footer*/
	footer {
		padding: 1.5rem 0;
		text-align: center;
	}

	footer p {
		font-size: 0.85rem;
		color: var(--text-secondary);
	}

	@media (max-width: 760px) {
		.about-page {
			margin-top: 1.5rem;
			gap: 1rem;
		}

		.slide-panel {
			padding: 1.25rem 1rem;
		}

		.contact-list li {
			align-items: flex-start;
			flex-direction: column;
			gap: 0.35rem;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		* {
			transition: none !important;
			animation: none !important;
		}
	}
</style>

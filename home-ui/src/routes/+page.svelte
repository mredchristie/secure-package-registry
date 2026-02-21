<script lang="ts">
  const lastUpdated = '2026-02-03';

  type SlideKey = 'Philosophies' | 'Values' | 'Team' | 'Impact' | 'Contact';

  interface Slide {
    key: SlideKey;
    title: string;
  }

  const slides: Slide[] = [
    {key: 'Philosophies', title: 'Security philosophies'},
    {key: 'Values', title: 'Core values'},
    {key: 'Team', title: 'Meet the team'},
    {key: 'Impact', title: 'Project impact'},
    {key: 'Contact', title: 'Get in touch'}
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
      summary:
              'Operational security must meet or exceed the standards we expect from clients.',
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
    href: string };

  const  contactInfo: ContactInfo[] = [
    { label: 'Email', value: 'spr@example.com', href: 'mailto:spr@example.com' },
    { label: 'Phone', value: '+123 456 789',    href: 'tel:+123456789'         }
  ];

  // PascalCase types for clarity
  type AboutOverview = {
    title: string;
    content: string;
  };

  const aboutInfo: AboutOverview[] = [
    { title: 'Overview', content: 'SPR (Secure Package Registry) is an alternative package registry that surfaces ' +
              'verification status so teams can make safer dependency choices.' },
  ];

  type MissionItem = {
    title: string;
    content: string;
  };

  const mission: MissionItem[] = [
     { title: 'Mission',
       content: 'SPR helps reduce software supply-chain risk by verifying packages before they reach developers. ' +
               'We focus on transparency: what was checked, what was not, and the resulting confidence signals. '},
     { title: 'Mission-muted',
       content: 'SPR is designed to be drop-in compatible with existing package ' +
               'manager workflows (configured via registry settings).' }
   ];

  type ValuesItem = {
    title: string;
    summary: string;
  };

  const values: ValuesItem[] = [
    { title: 'Values', summary: 'We prioritise security, reliability, and developer usability. ' +
              'Our outputs are designed to be actionable and auditable.' }
  ];

  type ImpactItem = {
    title: string;
    content: string;
  };

  const impact: ImpactItem[] = [
    { title: 'Project impact', content: 'By verifying packages before they reach developers,' +
              ' SPR helps reduce the risk of supply-chain attacks. ' +
              'Our transparent signals enable teams to make informed decisions and prioritise security' +
              ' in their dependencies.' }
  ];

</script>

<main class="about-page">
   <header class="hero">
     <p class="eyebrow">Secure Package Registry</p>
     <h1>About SPR</h1>
     <p class="lead">
       {#each aboutInfo as info}
         {info.content}
       {/each}
     </p>
   </header>

   <section class="card " aria-labelledby="mission-title">
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
                 type="button" class="tab" class:is-active={active === s.key} on:click={() => active = s.key}>
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
                           on:click={() => toggle(i)}>
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

 </main>

 <style>
  :global(body) {
    margin: 0;
    background:
            radial-gradient(circle at 10% 0%, #dce8f6 0%, rgba(220, 232, 246, 0) 46%),
            radial-gradient(circle at 90% 100%, #d8efec 0%, rgba(216, 239, 236, 0) 40%),
            linear-gradient(180deg, #f5f8fc 0%, #ecf2f8 100%);
  }

  .about-page {
    --surface: #ffffff;
    --surface-muted: #f8fbff;
    --text: #1a2736;
    --text-muted: #4f6074;
    --accent: #0d6bb5;
    --accent-strong: #0a4f87;
    --border: #d7e2ef;
    --radius: 16px;
    --shadow: 0 22px 40px -28px rgba(13, 48, 85, 0.45);
    max-width: 1080px;
    margin: 2.5rem auto 4rem;
    padding: 0 1.25rem;
    display: grid;
    gap: 1.25rem;
    color: var(--text);
    font-family: system-ui, -apple-system, 'Segoe UI', sans-serif;
  }

  .hero {
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) + 2px);
    background: linear-gradient(135deg, #fafdff 0%, #eef4fb 100%);
    box-shadow: var(--shadow);
    padding: clamp(1.5rem, 3vw, 2.5rem);
    animation: fade-in-up 500ms ease both;
  }

  .eyebrow {
    margin: 0;
    color: var(--accent);
    font-size: 0.78rem;
    letter-spacing: 0.13em;
    text-transform: uppercase;
    font-weight: 700;
  }

  h1 {
    margin: 0.35rem 0 0;
    color: #102336;
    font-size: clamp(1.8rem, 3vw, 2.6rem);
    line-height: 1.1;
    font-weight: 750;
  }

  .lead {
    margin: 0.85rem 0 0;
    max-width: 70ch;
    color: var(--text-muted);
    line-height: 1.65;
    font-size: 1.02rem;
  }

  .card {
    background: var(--surface);
    border-radius: var(--radius);
    border: 1px solid var(--border);
    box-shadow: var(--shadow);
    padding: clamp(1.25rem, 2.6vw, 1.8rem);
    display: grid;
    gap: 0.65rem;
    animation: fade-in-up 560ms ease both;
  }

  h2 {
    margin: 0 0 0.65rem;
    color: #12263a;
    font-size: 1.25rem;
    letter-spacing: 0.01em;
  }

  p {
    margin: 0;
    color: var(--text-muted);
    line-height: 1.68;
  }

  .muted {
    margin-top: 0.65rem;
    color: #5a6b80;
    font-size: 0.95rem;
  }

  /* Accordion */
  .accordion {
    display: grid;
    gap: 0.6rem;
    margin-top: 0.35rem;
  }

  .acc-item {
    border: 1px solid #d8e4f0;
    border-radius: 12px;
    overflow: hidden;
    background: #ffffff;
    transition: border-color 200ms ease, box-shadow 200ms ease;
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
    color: #12263a;
    cursor: pointer;
    transition: background 150ms ease;
  }

  .acc-trigger:hover {
    background: #f6faff;
  }

  .chev {
    flex-shrink: 0;
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    background: #eef4fb;
    color: var(--accent-strong);
    font-weight: 700;
    font-size: 1rem;
    transition: background 150ms ease;
  }

  .acc-panel {
    padding: 0 1rem 0.95rem;
    display: grid;
    gap: 0.65rem;
  }

  .acc-panel ul {
    margin: 0;
    padding-left: 1.1rem;
    color: var(--text-muted);
    line-height: 1.65;
    display: grid;
    gap: 0.25rem;
  }

  .team-list {
    margin: 0;
    padding-left: 1.1rem;
    color: var(--text-muted);
    line-height: 1.6;
    display: grid;
    gap: 0.35rem;
  }

  .team-list li {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.65rem 0.9rem;
    background: #f6faff;
    border: 1px solid #e5eef7;
    border-radius: 10px;
    font-size: 0.95rem;
  }

  .team-role {
    font-weight: 600;
    color: #12263a;
  }

  .team-name {
    color: var(--text-muted);
  }

  .contact-card {
    background: linear-gradient(160deg, var(--surface) 0%, var(--surface-muted) 100%);
  }

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
    background: #ffffff;
    border: 1px solid #d8e4f0;
    border-radius: 10px;
  }

  .contact-list span {
    color: #31465d;
    font-weight: 600;
    letter-spacing: 0.01em;
  }

  .contact-list a {
    color: var(--accent-strong);
    text-decoration: none;
    font-weight: 600;
  }

  .contact-list a:hover {
    color: var(--accent);
    text-decoration: underline;
  }

  .meta {
    margin-top: 0.9rem;
    color: #5a6b80;
    font-size: 0.92rem;
  }

  @keyframes fade-in-up {
    from { opacity: 0; transform: translateY(10px); }
    to { opacity: 1; transform: translateY(0); }
  }

  @media (max-width: 760px) {
    .about-page { margin-top: 1.5rem; gap: 1rem; }
    .contact-list li { align-items: flex-start; flex-direction: column; gap: 0.35rem; }
  }

  @media (prefers-reduced-motion: reduce) {
    .hero, .card { animation: none; }
  }

  /* Slide */
  .slideshow-container {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    box-shadow: var(--shadow);
    overflow: hidden;
    display: grid;
    grid-template-rows: auto 1fr;
  }

  .slide-tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    padding: 0.75rem 1rem;
    border-bottom: 2px solid var(--border);
    background: linear-gradient(135deg, #fafdff 0%, #f5f9fe 100%);
  }

  .tab {
    border: 1px solid transparent;
    background: transparent;
    padding: 0.6rem 1rem;
    border-radius: 8px;
    cursor: pointer;
    font-weight: 600;
    font-size: 0.92rem;
    color: var(--text-muted);
    transition: all 180ms ease;
    position: relative;
  }

  .tab:hover {
    background: rgba(255, 255, 255, 0.8);
    border-color: #c5d5e8;
    color: var(--text);
    transform: translateY(-1px);
  }

  .tab.is-active {
    background: linear-gradient(135deg, var(--accent) 0%, var(--accent-strong) 100%);
    color: #ffffff;
    border-color: var(--accent-strong);
    box-shadow: 0 4px 12px rgba(13, 107, 181, 0.25);
  }

  .tab.is-active:hover {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(13, 107, 181, 0.35);
  }

  .slide-panel {
    padding: 2rem 1.5rem;
    min-height: 480px;
    animation: slide-fade 240ms cubic-bezier(0.4, 0, 0.2, 1);
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    background: linear-gradient(180deg, #ffffff 0%, #fcfdff 100%);
  }

  .slide-panel > section {
    width: 100%;
    max-width: 100%;
    border: none;
    box-shadow: none;
    padding: 0;
    background: transparent;
  }

  .slide-panel .accordion {
    margin-top: 1rem;
  }

  .slide-panel .acc-item {
    border: 1px solid #e5eef7;
    background: #ffffff;
    transition: all 200ms ease;
  }

  .slide-panel .acc-item:hover {
    border-color: #d0e0f0;
    box-shadow: 0 2px 8px rgba(13, 107, 181, 0.08);
  }

  .slide-panel{
    box-shadow: 0 8px 20px rgba(13, 48, 85, 0.15);
    border: 2px solid #e8f0f8;
  }

  @keyframes slide-fade {
    from {
      opacity: 0;
      transform: translateY(8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @media (max-width: 760px) {
    .slide-tabs {
      padding: 0.6rem 0.75rem;
      gap: 0.3rem;
    }

    .tab {
      padding: 0.5rem 0.75rem;
      font-size: 0.88rem;
    }

    .slide-panel {
      padding: 1.5rem 1rem;
      min-height: 400px;
    }
  }

  @media (prefers-reduced-motion: no-preference) {
    .tab {
      transition: all 180ms ease;
    }

    .slide-panel {
      animation: slide-fade 240ms cubic-bezier(0.4, 0, 0.2, 1);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .tab {
      transition: none;
    }

    .slide-panel {
      animation: none;
    }
  }
</style>

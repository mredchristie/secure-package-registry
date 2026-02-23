<script lang="ts">
	// imports kinda self explanitory
	import { onMount } from 'svelte';
	import Header from '$lib/Header.svelte';
	import favicon from '$lib/assets/favicon.svg';
	// svelte 5 feature props - children contains all page content.
	let { children } = $props();
	// Theme state = true (dark mode) and = false (light mode)
	let isDark = $state(true);

	// Restore saved theme preference from browser storage
	onMount(() => {
		const saved = localStorage.getItem('isDark');
		if (saved !== null) isDark = saved === 'true';
	});

	function toggleTheme() {
		// Toggle theme and persist to localStorage
		isDark = !isDark;
		localStorage.setItem('isDark', isDark.toString());
	}
</script>

<svelte:head>
	<!-- Favicon for browser tab - this was already in the file when I rebased Mohammeds branch so kept it in -->
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="app" class:dark={isDark}>
	<Header {isDark} onToggleTheme={toggleTheme} />
	<!-- Header component with theme toggle (appears on all pages) -->
	{@render children()}
	<!-- Page content rendered here via {@render children()} -->
</div>

<style>
	/*
	  Color variable reference — light / dark
	  --bg-primary:     #fff            / #0a0a0f
	  --bg-secondary:   #fafafa         / #1a1a2e
	  --text-primary:   #111            / #fff
	  --text-secondary: #4b5563         / #ccc
	  --accent:         #1d4ed8 (blue)  / #4fc3f7 (sky blue)
	  --accent-hover:   #1e40af         / #51cf66 (green)
	  --border:         #e5e7eb         / #2a2a3e
	  --card-bg:        #fff            / rgba(26,26,46,0.5)
	  --card-border:    #e5e7eb         / rgba(79,195,247,0.2)
	*/

	/*Light mode default*/

	:global(.app) {
		/* light mode */
		--bg-primary: #fff;
		--bg-secondary: #fafafa;
		--text-primary: #111;
		--text-secondary: #4b5563;
		--accent: #1d4ed8;
		--accent-hover: #1e40af;
		--border: #e5e7eb;
		--card-bg: #fff;
		--card-border: #e5e7eb;

		/* Base app styles */
		font-family:
			system-ui,
			-apple-system,
			sans-serif;
		background: var(--bg-primary);
		color: var(--text-primary);
		line-height: 1.6;
		min-height: 100vh;

		/* Smooth theme transitions */
		transition:
			background 0.3s,
			color 0.3s;
	}

	/* Dark Mode */
	:global(.app.dark) {
		/* dark mode */
		--bg-primary: #0a0a0f;
		--bg-secondary: #1a1a2e;
		--text-primary: #fff;
		--text-secondary: #ccc;
		--accent: #4fc3f7;
		--accent-hover: #51cf66;
		--border: #2a2a3e;
		--card-bg: rgba(26, 26, 46, 0.5);
		--card-border: rgba(79, 195, 247, 0.2);
	}

	/*global resets*/
	:global(*) {
		margin: 0;
		padding: 0;
		box-sizing: border-box;
	}

	/* Prevent horizontal scroll, and Disable bounce effect on overscroll */
	:global(body) {
		margin: 0;
		padding: 0;
		overflow-x: hidden;
		overscroll-behavior: none;
	}

	/* Layout containers */
	:global(.container) {
		max-width: 1400px;
		margin: 0 auto;
		padding: 0; /* Pages control their own padding */
	}

	:global(.section) {
		padding: 6rem 0; /* Vertical spacing between sections */
	}

	/* Alternate background for some variety */
	:global(.section-alt) {
		background: var(--bg-secondary);
	}

	/* Responsive font size */
	:global(.section-title) {
		font-size: clamp(2rem, 5vw, 3.5rem);
		font-weight: 700;
		margin-bottom: 4rem;
		text-align: center;
	}

	:global(footer) {
		padding: 2rem;
		text-align: center;
		border-top: 1px solid var(--border);
	}

	:global(footer p) {
		font-size: 0.875rem;
		color: var(--text-secondary);
	}

	/* scroll animations — IntersectionObserver toggles .visible on viewport enter/exit */
	/* SCROLL ANIMATIONS
	   How they work:
	   1. .animate starts invisible (opacity: 0) with transform offset
	   2. IntersectionObserver in page JS adds .visible class when element enters viewport
	   3. CSS transition animates from hidden -> visible state
	   4. .visible is removed on scroll-out so animations replay
  	*/

	:global(.animate) {
		opacity: 0;
		transition: all 1s cubic-bezier(0.16, 1, 0.3, 1);
	}

	:global(.animate.visible) {
		opacity: 1;
	}

	:global(.slide-in-left) {
		transform: translateX(-100px);
	}

	:global(.slide-in-left.visible) {
		transform: translateX(0);
	}

	:global(.slide-in-right) {
		transform: translateX(100px);
	}

	:global(.slide-in-right.visible) {
		transform: translateX(0);
	}

	:global(.slide-in-bottom) {
		transform: translateY(50px);
	}

	:global(.slide-in-bottom.visible) {
		transform: translateY(0);
	}

	:global(.fade-in) {
		opacity: 0;
	}

	:global(.fade-in.visible) {
		opacity: 1;
	}

	:global(.zoom-in) {
		transform: scale(0.8);
	}

	:global(.zoom-in.visible) {
		transform: scale(1);
	}

	/* stagger delays for grouped items */
	:global(.delay-1) {
		transition-delay: 0.15s;
	}

	:global(.delay-2) {
		transition-delay: 0.3s;
	}

	:global(.delay-3) {
		transition-delay: 0.45s;
	}
</style>

<script lang="ts">
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";

  // Redirect to /search on mount — the root page is the public search
  onMount(() => {
    goto("/search", { replaceState: true });
  });
</script>

<main class="redirect">Redirecting to search...</main>

<style>
  .redirect {
    display: flex;
    min-height: 60vh;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
  }
</style>

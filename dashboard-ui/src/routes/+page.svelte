<script lang="ts">
  import { PUBLIC_HOME_BASE_URL } from "$env/static/public";
  import { onMount } from "svelte";

  onMount(() => {
    window.location.replace(PUBLIC_HOME_BASE_URL);
  });
</script>

<main class="redirect">Redirecting...</main>

<style>
  .redirect {
    display: flex;
    min-height: 60vh;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
  }
</style>

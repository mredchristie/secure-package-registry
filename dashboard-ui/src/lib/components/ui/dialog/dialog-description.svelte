<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    class?: string;
    children: Snippet;
  }

  let { children }: Props = $props();
</script>

<p class="text-sm text-slate-500">
  {@render children()}
</p>

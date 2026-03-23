<script lang="ts">
  import { cn } from "$lib/utils";
  import type { HTMLInputAttributes } from "svelte/elements";

  interface InputProps extends HTMLInputAttributes {
    class?: string;
  }

  let { class: className, ...props }: InputProps = $props();
</script>

<input
  class={cn(
    "flex h-9 w-full rounded-md border border-slate-200 bg-transparent px-3 py-1 text-sm shadow-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-slate-500 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-slate-950 disabled:cursor-not-allowed disabled:opacity-50",
    className,
  )}
  {...props}
/>

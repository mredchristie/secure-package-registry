<script lang="ts">
  import { cn } from "$lib/utils";
  import { type VariantProps, cva } from "class-variance-authority";
  import type {
    HTMLAnchorAttributes,
    HTMLButtonAttributes,
  } from "svelte/elements";

  const buttonVariants = cva(
    "inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-slate-950 disabled:pointer-events-none disabled:opacity-50",
    {
      defaultVariants: {
        size: "default",
        variant: "default",
      },
      variants: {
        size: {
          default: "h-9 px-4 py-2",
          icon: "h-9 w-9",
          lg: "h-10 rounded-md px-8",
          sm: "h-8 rounded-md px-3 text-xs",
        },
        variant: {
          default: "bg-slate-900 text-slate-50 shadow hover:bg-slate-900/90",
          destructive: "bg-red-600 text-slate-50 shadow-sm hover:bg-red-600/90",
          ghost: "hover:bg-slate-100 hover:text-slate-900",
          link: "text-slate-900 underline-offset-4 hover:underline",
          outline:
            "border border-slate-200 bg-white shadow-sm hover:bg-slate-100 hover:text-slate-900",
          secondary:
            "bg-slate-100 text-slate-900 shadow-sm hover:bg-slate-100/80",
        },
      },
    },
  );

  type ButtonVariant = VariantProps<typeof buttonVariants>["variant"];
  type ButtonSize = VariantProps<typeof buttonVariants>["size"];

  interface ButtonProps extends HTMLButtonAttributes {
    variant?: ButtonVariant;
    size?: ButtonSize;
    href?: string;
  }

  let {
    variant = "default",
    size = "default",
    href,
    class: className,
    children,
    ...props
  }: ButtonProps = $props();
</script>

{#if href}
  <a {href} class={cn(buttonVariants({ size, variant }), className)}>
    {@render children?.()}
  </a>
{:else}
  <button class={cn(buttonVariants({ size, variant }), className)} {...props}>
    {@render children?.()}
  </button>
{/if}

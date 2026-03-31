// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
import type { Session } from "$lib/client";

declare global {
	namespace App {
		// interface Error {}
		interface Locals {
			user: Session["user"] | null;
			session: Session["session"] | null;
		}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

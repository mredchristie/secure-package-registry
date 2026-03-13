import { createAuthClient } from "better-auth/svelte";
import { PUBLIC_HOME_BASE_URL } from "$env/static/public";

export const authClient = createAuthClient({
	baseURL: PUBLIC_HOME_BASE_URL,
});

export type Session = typeof authClient.$Infer.Session;

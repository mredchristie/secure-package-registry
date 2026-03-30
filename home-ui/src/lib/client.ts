import { createAuthClient } from "better-auth/svelte";
import { PUBLIC_DASHBOARD_BASE_URL } from "$env/static/public";

export const authClient = createAuthClient({
	baseURL: PUBLIC_DASHBOARD_BASE_URL,
});

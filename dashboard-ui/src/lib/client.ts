import { apiKeyClient } from "@better-auth/api-key/client";
import { createAuthClient } from "better-auth/svelte";
import { PUBLIC_DASHBOARD_BASE_URL } from "$env/static/public";

export const authClient = createAuthClient({
	baseURL: PUBLIC_DASHBOARD_BASE_URL,
	plugins: [apiKeyClient()],
});

export type Session = typeof authClient.$Infer.Session;

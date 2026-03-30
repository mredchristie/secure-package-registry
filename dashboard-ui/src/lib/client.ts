import { apiKeyClient } from "@better-auth/api-key/client";
import { createAuthClient } from "better-auth/svelte";

export const authClient = createAuthClient({
	plugins: [apiKeyClient()],
});

export type Session = typeof authClient.$Infer.Session;

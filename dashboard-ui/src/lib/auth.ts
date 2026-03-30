import { apiKey } from "@better-auth/api-key";
import { betterAuth } from "better-auth";
import { admin, organization } from "better-auth/plugins";
import { sveltekitCookies } from "better-auth/svelte-kit";
import { Pool } from "pg";
import { getRequestEvent } from "$app/server";

export const auth = betterAuth({
	database: new Pool({
		database: process.env.POSTGRES_DB,
		host: process.env.POSTGRES_HOST,
		password: process.env.POSTGRES_PASSWORD,
		port: process.env.POSTGRES_PORT
			? Number.parseInt(process.env.POSTGRES_PORT, 10)
			: 5432,
		user: process.env.POSTGRES_USER,
	}),

	emailAndPassword: {
		autoSignIn: false,
		enabled: true,
	},

	plugins: [
		admin(),
		organization(),
		apiKey(),
		sveltekitCookies(getRequestEvent), // make sure this is the last plugin in the array
	],

	trustedOrigins: process.env.TRUSTED_ORIGINS
		? process.env.TRUSTED_ORIGINS.split(",")
		: [
				"http://localhost:8000",
				"http://localhost:5174",
				"http://localhost:5173",
				"http://localhost:3001",
			],
});

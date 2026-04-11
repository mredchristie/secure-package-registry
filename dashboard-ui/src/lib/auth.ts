import { apiKey } from "@better-auth/api-key";
import { betterAuth } from "better-auth";
import { admin, jwt, organization } from "better-auth/plugins";
import { sveltekitCookies } from "better-auth/svelte-kit";
import { Pool } from "pg";
import { getRequestEvent } from "$app/server";

// Parse TRUSTED_ORIGINS from environment variable
// Format: comma-separated list of origins, e.g., "http://localhost:7001
const trustedOrigins = process.env.TRUSTED_ORIGINS?.split(",")
	.map((origin) => origin.trim())
	.filter((origin) => origin.length > 0) ?? ["http://localhost:7001"];

const pool = new Pool({
	database: process.env.POSTGRES_DB,
	host: process.env.POSTGRES_HOST,
	password: process.env.POSTGRES_PASSWORD,
	port: process.env.POSTGRES_PORT
		? Number.parseInt(process.env.POSTGRES_PORT, 10)
		: 5432,
	user: process.env.POSTGRES_USER,
});

export const auth = betterAuth({
	baseURL: process.env.PUBLIC_DASHBOARD_BASE_URL || "http://localhost:7001",

	database: pool,

	emailAndPassword: {
		enabled: true,
	},

	plugins: [
		organization(),
		apiKey(),
		admin(),
		jwt(),
		sveltekitCookies(getRequestEvent), // make sure this is the last plugin in the array
	],

	trustedOrigins,

	databaseHooks: {
		user: {
			create: {
				async before(user) {
					// The first user to sign up becomes admin.
					const { rows } = await pool.query(
						'SELECT COUNT(*)::int AS count FROM "user"',
					);
					const userCount = rows[0]?.count ?? 0;
					return {
						data: {
							...user,
							role: userCount === 0 ? "admin" : "user",
						},
					};
				},
			},
		},
	},
});

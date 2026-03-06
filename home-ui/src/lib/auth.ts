import { getRequestEvent } from "$app/server";
import { betterAuth } from "better-auth";
import { sveltekitCookies } from "better-auth/svelte-kit";
import { Pool } from "pg";
import { organization } from "better-auth/plugins";

export const auth = betterAuth({
    database: new Pool({
        host: process.env.POSTGRES_HOST,
        port: process.env.POSTGRES_PORT ? parseInt(process.env.POSTGRES_PORT) : 5432,
        user: process.env.POSTGRES_USER,
        password: process.env.POSTGRES_PASSWORD,
        database: process.env.POSTGRES_DB,
    }),

    emailAndPassword: {
        enabled: true,
    },

    plugins: [
        organization(),
        sveltekitCookies(getRequestEvent) // make sure this is the last plugin in the array
    ],
});

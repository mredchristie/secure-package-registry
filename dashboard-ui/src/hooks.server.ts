import type { Handle } from "@sveltejs/kit";
import { svelteKitHandler } from "better-auth/svelte-kit";
import { building } from "$app/environment";
import { auth } from "$lib/auth";
// import { seed } from "$lib/server/seed";

// let seeded = false;
export const handle: Handle = async ({ event, resolve }) => {
	// Fetch current session from Better Auth
	// const session = await auth.api.getSession({
	// 	headers: event.request.headers,
	// });

	// Make session and user available on server
	// TODO: type errors occuring
	// if (session) {
	// 	event.locals.session = session.session;
	// 	event.locals.user = session.user;
	// }

	// if (!seeded) {
	//     await seed();
	//     seeded = true;
	// }

	return svelteKitHandler({ auth, building, event, resolve });
};

import type { Handle } from "@sveltejs/kit";
import { building } from "$app/environment";

// DASHBOARD_INTERNAL_URL is used for server-to-server calls (container network).
// Falls back to PUBLIC_DASHBOARD_BASE_URL for local dev.
const dashboardUrl =
	process.env.DASHBOARD_INTERNAL_URL ??
	process.env.PUBLIC_DASHBOARD_BASE_URL ??
	"http://localhost:5174";

export const handle: Handle = async ({ event, resolve }) => {
	if (!building) {
		try {
			const response = await fetch(`${dashboardUrl}/api/auth/get-session`, {
				headers: event.request.headers,
			});
			if (response.ok) {
				const data = await response.json();
				if (data?.user) {
					event.locals.user = data.user;
					event.locals.session = data.session;
				}
			}
		} catch {
			// dashboard-ui may not be running
		}
	}
	return resolve(event);
};

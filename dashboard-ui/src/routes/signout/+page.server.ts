import { redirect } from "@sveltejs/kit";
import { PUBLIC_HOME_BASE_URL } from "$env/static/public";
import { auth } from "$lib/auth";

export const load = async ({ request }) => {
	await auth.api.signOut({ headers: request.headers });
	redirect(302, PUBLIC_HOME_BASE_URL);
};

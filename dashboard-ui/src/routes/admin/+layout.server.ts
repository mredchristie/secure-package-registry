import { redirect } from "@sveltejs/kit";
import type { LayoutServerLoad } from "./$types";

export const load: LayoutServerLoad = ({ locals }) => {
	if (!locals.user) throw redirect(302, "/login");
	if (locals.user.role !== "admin") throw redirect(302, "/");
	return {
		user: locals.user,
	};
};

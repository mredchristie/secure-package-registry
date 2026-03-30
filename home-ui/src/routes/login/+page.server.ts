import { redirect } from "@sveltejs/kit";
import { PUBLIC_DASHBOARD_BASE_URL } from "$env/static/public";

export const load = () => {
	redirect(302, `${PUBLIC_DASHBOARD_BASE_URL}/login`);
};

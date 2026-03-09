import type { LayoutServerLoad } from "./$types";

export const load: LayoutServerLoad = ({ cookies }) => {
	return {
		isLoggedIn: cookies.get("isLoggedIn") === "true",
	};
};

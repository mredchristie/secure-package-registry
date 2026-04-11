import { authClient } from "$lib/client.js";
import type {
	AddPackageRequest,
	AddPackageResponse,
	DependencyType,
	Ecosystem,
	ListPackagesResponse,
	ListProjectDependenciesResponse,
	ListProjectsResponse,
	ListTasksResponse,
	PackageVersion,
	PackageVersionDetail,
	ProcessTree,
	Project,
	ProjectSummaryResponse,
	ReviewQueueResponse,
	ReviewStatusResponse,
	SearchResult,
	TriggerScanRequest,
	TriggerScanResponse,
	UploadProjectResponse,
	VerifyResponse,
	VersionListResult,
} from "$lib/types/api.js";

const API_BASE = "/api/v1";

class APIError extends Error {
	constructor(
		public status: number,
		public override message: string,
	) {
		super(message);
	}
}

async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
	const response = await fetch(url, {
		...options,
		headers: {
			"Content-Type": "application/json",
			...options?.headers,
		},
	});

	if (!response.ok) {
		const error = await response
			.json()
			.catch(() => ({ error: "Unknown error" }));
		throw new APIError(
			response.status,
			error.error || `HTTP ${response.status}`,
		);
	}

	return response.json();
}

// Authenticated fetch — obtains a BetterAuth JWT and sends it as a Bearer token.
// Used for project API calls that require user authentication.
async function authenticatedFetchJSON<T>(
	url: string,
	options?: RequestInit,
): Promise<T> {
	const { data, error } = await authClient.token();
	if (error || !data?.token) {
		throw new APIError(401, "Not authenticated");
	}
	return fetchJSON(url, {
		...options,
		headers: {
			...options?.headers,
			Authorization: `Bearer ${data.token}`,
		},
	});
}

// Admin API — for the /admin dashboard
export const packagesAPI = {
	add: (data: AddPackageRequest): Promise<AddPackageResponse> => {
		return fetchJSON(`${API_BASE}/admin/packages`, {
			body: JSON.stringify(data),
			method: "POST",
		});
	},

	behavior: (
		ecosystem: string,
		identifier: string,
		version: string,
	): Promise<ProcessTree> => {
		return fetchJSON(
			`${API_BASE}/admin/packages/${encodeURIComponent(ecosystem)}/${encodeURIComponent(identifier)}/behavior?version=${encodeURIComponent(version)}`,
		);
	},

	behaviorRaw: (
		ecosystem: string,
		identifier: string,
		version: string,
	): Promise<ProcessTree> => {
		return fetchJSON(
			`${API_BASE}/admin/packages/${encodeURIComponent(ecosystem)}/${encodeURIComponent(identifier)}/behavior/raw?version=${encodeURIComponent(version)}`,
		);
	},

	getReviewStatus: (
		ecosystem: string,
		identifier: string,
		version: string,
	): Promise<ReviewStatusResponse> => {
		return fetchJSON(
			`${API_BASE}/admin/packages/${encodeURIComponent(ecosystem)}/${encodeURIComponent(identifier)}/versions/${encodeURIComponent(version)}/review`,
		);
	},

	list: (ecosystem: Ecosystem): Promise<ListPackagesResponse> => {
		return fetchJSON(`${API_BASE}/admin/packages?ecosystem=${ecosystem}`);
	},

	reviewQueue: (params?: {
		ecosystem?: string;
		status?: string;
	}): Promise<ReviewQueueResponse> => {
		const searchParams = new URLSearchParams();
		if (params?.ecosystem) searchParams.set("ecosystem", params.ecosystem);
		if (params?.status) searchParams.set("status", params.status);
		const qs = searchParams.toString();
		const url = qs
			? `${API_BASE}/admin/review?${qs}`
			: `${API_BASE}/admin/review`;
		return fetchJSON(url);
	},

	scan: (
		ecosystem: string,
		identifier: string,
		data: TriggerScanRequest,
	): Promise<TriggerScanResponse> => {
		return fetchJSON(
			`${API_BASE}/admin/packages/${encodeURIComponent(ecosystem)}/${encodeURIComponent(identifier)}/scan`,
			{
				body: JSON.stringify(data),
				method: "POST",
			},
		);
	},

	submitReview: (
		ecosystem: string,
		identifier: string,
		version: string,
		data: { approved: boolean; comment: string },
	): Promise<ReviewStatusResponse> => {
		return fetchJSON(
			`${API_BASE}/admin/packages/${encodeURIComponent(ecosystem)}/${encodeURIComponent(identifier)}/versions/${encodeURIComponent(version)}/review`,
			{
				body: JSON.stringify(data),
				method: "POST",
			},
		);
	},

	versions: (
		ecosystem: string,
		identifier: string,
	): Promise<PackageVersion> => {
		return fetchJSON(
			`${API_BASE}/admin/packages/${encodeURIComponent(ecosystem)}/${encodeURIComponent(identifier)}/versions`,
		);
	},
};

export const tasksAPI = {
	downloadArtifact: (taskId: number): Promise<Response> => {
		return fetch(`${API_BASE}/admin/tasks/${taskId}/artifact`);
	},
	list: (params?: {
		ecosystem?: Ecosystem;
		page?: number;
		page_size?: number;
	}): Promise<ListTasksResponse> => {
		const searchParams = new URLSearchParams();
		if (params?.ecosystem) searchParams.set("ecosystem", params.ecosystem);
		if (params?.page) searchParams.set("page", params.page.toString());
		if (params?.page_size)
			searchParams.set("page_size", params.page_size.toString());

		const queryString = searchParams.toString();
		const url = queryString
			? `${API_BASE}/admin/tasks?${queryString}`
			: `${API_BASE}/admin/tasks`;

		return fetchJSON(url);
	},
};

// Project API — for user dependency tracking
export const projectsAPI = {
	delete: async (projectId: number): Promise<void> => {
		const { data, error } = await authClient.token();
		if (error || !data?.token) {
			throw new APIError(401, "Not authenticated");
		}
		const res = await fetch(`${API_BASE}/projects/${projectId}`, {
			headers: { Authorization: `Bearer ${data.token}` },
			method: "DELETE",
		});
		if (!res.ok) throw new APIError(res.status, "Failed to delete project");
	},

	dependencies: (
		projectId: number,
		type?: DependencyType,
	): Promise<ListProjectDependenciesResponse> => {
		const params = new URLSearchParams();
		if (type) params.set("type", type);
		const qs = params.toString();
		const url = qs
			? `${API_BASE}/projects/${projectId}/dependencies?${qs}`
			: `${API_BASE}/projects/${projectId}/dependencies`;
		return authenticatedFetchJSON(url);
	},

	get: (projectId: number): Promise<Project> => {
		return authenticatedFetchJSON(`${API_BASE}/projects/${projectId}`);
	},

	list: (): Promise<ListProjectsResponse> => {
		return authenticatedFetchJSON(`${API_BASE}/projects`);
	},

	summary: (projectId: number): Promise<ProjectSummaryResponse> => {
		return authenticatedFetchJSON(`${API_BASE}/projects/${projectId}/summary`);
	},

	upload: (name: string, file: string): Promise<UploadProjectResponse> => {
		return authenticatedFetchJSON(`${API_BASE}/projects`, {
			body: JSON.stringify({ file, name }),
			method: "POST",
		});
	},
};

// Public search API — for /api/v1/svc/ endpoints
export const searchAPI = {
	getVersion: (
		ecosystem: string,
		identifier: string,
		version: string,
	): Promise<PackageVersionDetail> => {
		const safeIdentifier = encodeURIComponent(identifier);
		return fetchJSON(
			`${API_BASE}/svc/packages/${ecosystem}/${safeIdentifier}/${version}`,
		);
	},

	listVersions: (
		ecosystem: string,
		identifier: string,
	): Promise<VersionListResult> => {
		const safeIdentifier = encodeURIComponent(identifier);
		return fetchJSON(
			`${API_BASE}/svc/packages/${ecosystem}/${safeIdentifier}/versions`,
		);
	},

	search: (
		query?: string,
		ecosystem?: string,
		page?: number,
		pageSize?: number,
	): Promise<SearchResult> => {
		const params = new URLSearchParams();
		if (query?.trim()) params.append("q", query);
		if (ecosystem) params.append("ecosystem", ecosystem);
		if (page) params.append("page", page.toString());
		if (pageSize) params.append("page_size", pageSize.toString());

		const queryString = params.toString();
		const url = queryString
			? `${API_BASE}/svc/packages?${queryString}`
			: `${API_BASE}/svc/packages`;

		return fetchJSON(url);
	},

	verify: (
		ecosystem: string,
		identifier: string,
		version: string,
	): Promise<VerifyResponse> => {
		const safeIdentifier = encodeURIComponent(identifier);
		return fetchJSON(
			`${API_BASE}/svc/packages/${ecosystem}/${safeIdentifier}/${version}/verify`,
			{ method: "POST" },
		);
	},
};

export type {
	CollectionTaskStatus,
	DependencyType,
	Ecosystem,
} from "$lib/types/api.js";

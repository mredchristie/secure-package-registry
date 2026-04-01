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
	SearchResult,
	TriggerScanRequest,
	TriggerScanResponse,
	UploadProjectResponse,
	VerifyResponse,
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

	list: (ecosystem: Ecosystem): Promise<ListPackagesResponse> => {
		return fetchJSON(`${API_BASE}/admin/packages?ecosystem=${ecosystem}`);
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
	delete: (projectId: number): Promise<void> => {
		return fetch(`${API_BASE}/projects/${projectId}`, {
			method: "DELETE",
		}).then((res) => {
			if (!res.ok) throw new APIError(res.status, "Failed to delete project");
		});
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
		return fetchJSON(url);
	},

	get: (projectId: number): Promise<Project> => {
		return fetchJSON(`${API_BASE}/projects/${projectId}`);
	},

	list: (): Promise<ListProjectsResponse> => {
		return fetchJSON(`${API_BASE}/projects`);
	},

	summary: (projectId: number): Promise<ProjectSummaryResponse> => {
		return fetchJSON(`${API_BASE}/projects/${projectId}/summary`);
	},

	upload: (name: string, file: string): Promise<UploadProjectResponse> => {
		return fetchJSON(`${API_BASE}/projects`, {
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

	search: (query?: string, ecosystem?: string): Promise<SearchResult> => {
		const params = new URLSearchParams();
		if (query?.trim()) params.append("q", query);
		if (ecosystem) params.append("ecosystem", ecosystem);

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

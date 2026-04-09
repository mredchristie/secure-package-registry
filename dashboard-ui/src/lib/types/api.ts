export type Ecosystem = "npm" | "go" | "cargo" | "pypi";

export type CollectionTaskStatus =
	| "pending"
	| "running"
	| "succeeded"
	| "failed"
	| "cancelled";

export interface Package {
	id: number;
	identifier: string;
	ecosystem: string;
	latest_version?: string;
}

export interface PackageVersion {
	identifier: string;
	ecosystem: string;
	latest_version?: string;
	versions: string[];
}

export interface CollectionTask {
	id: number;
	identifier: string;
	ecosystem: string;
	version: string;
	source: string;
	status: CollectionTaskStatus;
	failure_reason?: string;
	has_artifact: boolean;
	started_at?: string;
	completed_at?: string;
	created_at: string;
}

export interface ListPackagesResponse {
	items: Package[];
}

export interface AddPackageRequest {
	identifier: string;
	ecosystem: Ecosystem;
}

export interface AddPackageResponse {
	id: number;
	identifier: string;
	ecosystem: string;
	already_exists: boolean;
}

export interface TriggerScanRequest {
	version?: string;
}

export interface TriggerScanResponse {
	task_id: number;
	identifier: string;
	ecosystem: string;
	version: string;
	status: string;
}

export interface ListTasksResponse {
	items: CollectionTask[];
}

// Behavioral analysis types — mirrors Go pkg/behavior types

export interface ProcessTree {
	root: ProcessNode;
}

export interface ProcessNode {
	name: string;
	identity: string;
	behaviors: ProcessBehaviors;
	children?: ProcessNode[];
}

export interface ProcessBehaviors {
	execs?: ExecBehavior[];
	files_opened?: string[];
	connections?: ConnectionBehavior[];
	dns_queries?: string[];
}

export interface ExecBehavior {
	pathname: string;
	argv: string[];
}

export interface ConnectionBehavior {
	addr: string;
	port: number;
}

// Public search types (used by /api/v1/svc/ endpoints)

export interface PackageSummary {
	ecosystem: Ecosystem;
	identifier: string;
	latest_version: string;
}

export interface SearchResult {
	items: PackageSummary[];
	page: number;
	page_size: number;
	total_count: number;
}

export interface PackageVersionDetail {
	ecosystem: string;
	identifier: string;
	latest: boolean;
	maintainer_notes: string;
	source: {
		commit: string;
		tag: string;
		url: string;
	};
	tags: Array<{
		data: string; // base64 encoded JSON
		label: string;
		value_type: "boolean" | "integer" | "float";
	}>;
	trust_level: number;
	version: string;
}

export interface VerifyResponse {
	ecosystem: string;
	identifier: string;
	oss_rebuild: boolean;
	upstream_attestation: boolean;
	version: string;
}

// Version list types (public endpoint)

export interface VersionSummary {
	behavior_passed: boolean;
	has_attestation: boolean;
	has_oss_rebuild: boolean;
	latest: boolean;
	source: {
		commit: string;
		tag: string;
		url: string;
	};
	version: string;
}

export interface VersionListResult {
	ecosystem: Ecosystem;
	identifier: string;
	versions: VersionSummary[];
}

// Project dependency tracking types

export type DependencyType = "direct" | "transitive";

export interface Project {
	id: number;
	name: string;
	source_type: string;
	created_at?: string;
	updated_at?: string;
}

export interface ProjectDependency {
	behavior_passed: boolean;
	dependency_type: DependencyType;
	ecosystem: string;
	has_attestation: boolean;
	has_oss_rebuild: boolean;
	id: number;
	identifier: string;
	version: string;
	version_constraint?: string;
}

export interface ProjectSummaryRow {
	behavior_passed: number;
	dependency_type: DependencyType;
	has_attestation: number;
	has_oss_rebuild: number;
	total: number;
}

export interface ListProjectsResponse {
	items: Project[];
}

export interface UploadProjectResponse {
	id: number;
	name: string;
	source_type: string;
	total_deps: number;
	direct_deps: number;
}

export interface ListProjectDependenciesResponse {
	items: ProjectDependency[];
}

export interface ProjectSummaryResponse {
	project_id: number;
	summary: ProjectSummaryRow[];
}

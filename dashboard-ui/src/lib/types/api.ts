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
	identifier: string;
	ecosystem: "npm" | "go" | "cargo" | "pypi";
	latest_version: string;
	description: string;
	author: string;
	updatedAgo: string;
	trustScore: number;
	tier: string;
	tags: string[];
}

export interface SearchResult {
	items: PackageSummary[];
}

export interface PackageVersionDetail {
	identifier: string;
	ecosystem: string;
	version: string;
	latest: boolean;
	source: {
		url: string;
		tag: string;
		commit: string;
	};
	trust_level: number;
	maintainer_notes: string;
	tags: Array<{
		label: string;
		value_type: "boolean" | "integer" | "float";
		data: string; // base64 encoded JSON
	}>;
}

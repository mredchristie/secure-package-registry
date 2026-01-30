export default {
	async fetch(request, env) {
		const secret = request.headers.get("X-Gitlab-Token");
		if (secret !== env.GITLAB_WEBHOOK_SECRET) {
			return new Response("Unauthorized", { status: 401 });
		}

		if (request.method !== "POST") {
			return new Response("Method not allowed", {
				status: 405,
			});
		}

		try {
			const payload = await request.json();
			const eventType = payload.object_kind;

			switch (eventType) {
				case "merge_request":
					return handleMergeRequest(payload, env);
				case "issue":
					return handleIssue(payload, env);
				case "note":
					return handleComment(payload, env);
				default:
					return new Response(
						`Ignored: ${eventType}`,
						{ status: 200 },
					);
			}
		} catch (error) {
			console.error("Webhook error:", error);
			return new Response(`Error: ${error.message}`, {
				status: 500,
			});
		}
	},
};

async function handleMergeRequest(payload, env) {
	const mr = payload.object_attributes;
	const action = mr.action;
	const project = payload.project;
	const user = payload.user;

	if (action === "update") {
		return new Response("Ignored: MR update", { status: 200 });
	}

	if (!["open", "merge", "close", "reopen"].includes(action)) {
		return new Response(`Ignored: MR ${action}`, { status: 200 });
	}

	const config = {
		open: {
			text: "New Merge Request",
			emoji: "🆕",
		},
		reopen: {
			text: "Reopened Merge Request",
			emoji: "🔁",
		},
		merge: { text: "Merged", emoji: "✅" },
		close: { text: "Closed", emoji: "❌" },
	}[action];

	const title = `${config.emoji} MR !${mr.iid}: ${mr.title}`;
	
	const messageLines = [
		`${title.length > 200 ? title.substring(0, 197) + "..." : title}`,
		`[View Merge Request](${mr.url})`,
		`\n${config.text} by **${user.name}**`,
		`Branch: \`${mr.source_branch}\` → \`${mr.target_branch}\``,
		`📁 Project: ${project.namespace}/${project.name}`,
	`	Author: ${mr.author?.name || user.name}`,
	];

	if (mr.description?.trim()) {
		const desc = mr.description.length > 200
			? mr.description.substring(0, 200) + "..."
			: mr.description;
		messageLines.push(`\n📝 Description:\n${desc}`);
	}

	await sendToDiscord(env.DISCORD_WEBHOOK_IMPORTANT, {
		username: "GitLab MR Bot",
		avatar_url: "https://about.gitlab.com/images/press/logo/png/gitlab-icon-rgb.png",
		content: messageLines.join("\n"),
	});

	return new Response(`Sent MR ${action}`, { status: 200 });
}

async function handleIssue(payload, env) {
	const issue = payload.object_attributes;
	const action = issue.action;
	const project = payload.project;
	const user = payload.user;

	if (action === "update") {
		return new Response("Ignored: Issue update", { status: 200 });
	}

	if (!["open", "close", "reopen"].includes(action)) {
		return new Response(`Ignored: Issue ${action}`, {
			status: 200,
		});
	}

	const config = {
		open: { text: "New Issue", emoji: "🐛" },
		reopen: {
			text: "Reopened Issue",
			emoji: "🔁",
		},
		close: {
			text: "Closed Issue",
			emoji: "✅",
		},
	}[action];

	const title = `${config.emoji} Issue #${issue.iid}: ${issue.title}`;
	
	const labels = issue.labels?.map((l) => l.title).join(", ") || "None";
	
	const messageLines = [
		`${title.length > 200 ? title.substring(0, 197) + "..." : title}`,
		`[View Issue](${issue.url})`,
		`\n${config.text} by **${user.name}**`,
		`📁 Project: ${project.namespace}/${project.name}`,
	`	🏷️ Labels: ${labels}`,
	];

	if (issue.description?.trim()) {
		const desc = issue.description.length > 200
			? issue.description.substring(0, 200) + "..."
			: issue.description;
		messageLines.push(`\n📝 Description:\n${desc}`);
	}

	await sendToDiscord(env.DISCORD_WEBHOOK_IMPORTANT, {
		username: "GitLab Issue Bot",
		avatar_url: "https://about.gitlab.com/images/press/logo/png/gitlab-icon-rgb.png",
		content: messageLines.join("\n"),
	});

	return new Response(`Sent Issue ${action}`, { status: 200 });
}

async function handleComment(payload, env) {
	const note = payload.object_attributes;
	const project = payload.project;

	// FIXED: Use robust author detection
	const authorName = payload.user.name;

	let parentTitle = "";
	let parentUrl = "";
	let parentType = "";
	let color = 9807270;

	switch (note.noteable_type) {
		case "MergeRequest":
			parentType = "Merge Request";
			color = 3447003;
			if (payload.merge_request) {
				parentTitle = `MR !${payload.merge_request.iid}: ${payload.merge_request.title}`;
				parentUrl = payload.merge_request.url;
			} else {
				parentTitle = "Merge Request";
				parentUrl = note.url;
			}
			break;

		case "Issue":
			parentType = "Issue";
			color = 3447003;
			if (payload.issue) {
				parentTitle = `Issue #${payload.issue.iid}: ${payload.issue.title}`;
				parentUrl = payload.issue.url;
			} else {
				parentTitle = "Issue";
				parentUrl = note.url;
			}
			break;

		case "Commit":
			parentType = "Commit";
			color = 7506394;
			const shortSha =
				note.commit_id?.substring(0, 7) || "unknown";
			parentTitle = `Commit ${shortSha}`;
			parentUrl = note.url;
			break;

		case "Snippet":
			parentType = "Snippet";
			parentTitle = `Snippet #${note.noteable_id}`;
			parentUrl = note.url;
			break;

		default:
			parentType = note.noteable_type || "Comment";
			parentTitle = "Comment";
			parentUrl = note.url;
	}

	// Truncate note text intelligently
	const noteText = note.note || "";
	const maxLength = 400;
	const truncatedNote =
		noteText.length > maxLength
			? noteText.substring(0, maxLength) + "..."
			: noteText;

	const message = [
		`💬 **New comment on ${parentType}**`,
		`[${parentTitle}](${parentUrl})`,
		`
**${authorName}** commented:
${truncatedNote}
`,
		`📁 Project: ${project.namespace}/${project.name}`,
	].join("\n");

	await sendToDiscord(env.DISCORD_WEBHOOK_UNIMPORTANT, {
		username: "GitLab Comments",
		avatar_url: "https://about.gitlab.com/images/press/logo/png/gitlab-icon-rgb.png",
		content: message,
	});

	return new Response("Sent comment", { status: 200 });
}

async function sendToDiscord(webhookUrl, payload) {
	const response = await fetch(webhookUrl, {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});

	if (!response.ok) {
		const text = await response.text();
		throw new Error(`Discord error ${response.status}: ${text}`);
	}

	return response;
}

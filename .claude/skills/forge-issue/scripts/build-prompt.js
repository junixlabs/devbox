#!/usr/bin/env node
/**
 * Build minimal agent prompt that invokes the forge-issue skill.
 *
 * Usage:
 *   echo '{"issues": [...]}' | node build-prompt.js
 *   node build-prompt.js input.json
 *
 * Input JSON shape:
 *   { issues: Issue[] }
 *   { task: Task }
 *
 * Output: skill invocation string to stdout
 *
 * The skill itself fetches all issue data via MCP tools,
 * so the prompt only needs documentIds.
 */

const fs = require("fs");

async function main() {
  let raw;
  const filePath = process.argv[2];
  if (filePath) {
    raw = fs.readFileSync(filePath, "utf-8");
  } else {
    raw = fs.readFileSync("/dev/stdin", "utf-8");
  }

  const input = JSON.parse(raw);

  let prompt;
  if (input.task) {
    const issueId = input.task.issue?.documentId;
    if (issueId) {
      prompt = `/forge-issue ${issueId}\n\nFocus on task: ${input.task.title} (${input.task.documentId})`;
    } else {
      prompt = `Work on task: ${input.task.title}\n\n${input.task.description ?? ""}\n\nTask DocumentId: ${input.task.documentId}`;
    }
  } else if (input.issues && input.issues.length > 0) {
    const ids = input.issues.map((i) => i.documentId).join(" ");
    prompt = `/forge-issue ${ids}`;
  } else {
    console.error("Error: input must have 'issues' array or 'task' object");
    process.exit(1);
  }

  process.stdout.write(prompt);
}

main().catch((err) => {
  console.error(err.message);
  process.exit(1);
});

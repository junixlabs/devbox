import { readFileSync, existsSync } from 'node:fs';
import { basename, extname } from 'node:path';
/**
 * Forge API CLI for Antigravity agents.
 * High-level commands that wrap the Strapi REST API.
 *
 * Commands:
 *   node forge-api.mjs get-issue <documentId>
 *   node forge-api.mjs get-issue <documentId> --fields=status,plan,acceptanceCriteria
 *   node forge-api.mjs update-issue <documentId> --status=confirmed --category=bug --plan=@plan.md
 *   node forge-api.mjs update-issue <documentId> --data-file=payload.json
 *   node forge-api.mjs search-issues "keyword1 keyword2" --exclude=<documentId> --limit=10
 *   node forge-api.mjs list-comments <issueDocumentId> --limit=5
 *   node forge-api.mjs create-comment <issueDocumentId> --body=@file.md --author=Snorlax
 *   node forge-api.mjs create-comment <issueDocumentId> --body=@file.md --author=Snorlax --attachments=42,43
 *   node forge-api.mjs upload <filepath>
 */

const args = process.argv.slice(2);
const command = args[0] || '';
const baseUrl = process.env.FORGE_API_URL;
const apiKey = process.env.FORGE_API_KEY;

if (!baseUrl || !apiKey) {
  console.error('FORGE_API_URL and FORGE_API_KEY env vars required.');
  process.exit(1);
}

function parseFlags(args) {
  const flags = {};
  const positional = [];
  for (const arg of args) {
    if (arg.startsWith('--')) {
      const eq = arg.indexOf('=');
      if (eq === -1) { flags[arg.slice(2)] = true; }
      else { flags[arg.slice(2, eq)] = arg.slice(eq + 1); }
    } else { positional.push(arg); }
  }
  return { flags, positional };
}

function resolveValue(value) {
  if (typeof value !== 'string') return value;
  if (value.startsWith('@')) {
    const content = readFileSync(value.slice(1), 'utf-8');
    if (value.endsWith('.json')) { try { return JSON.parse(content); } catch { return content; } }
    return content;
  }
  if ((value.startsWith('[') && value.endsWith(']')) || (value.startsWith('{') && value.endsWith('}'))) {
    try { return JSON.parse(value); } catch { /* keep as string */ }
  }
  if (value === 'true') return true;
  if (value === 'false') return false;
  if (value === 'null') return null;
  if (/^\d+$/.test(value)) return parseInt(value, 10);
  return value;
}

async function api(method, path, body) {
  const opts = { method, headers: { 'x-forge-api-key': apiKey } };
  if (body) { opts.headers['Content-Type'] = 'application/json'; opts.body = JSON.stringify(body); }
  const res = await fetch(`${baseUrl}${path}`, opts);
  const text = await res.text();
  let parsed;
  try { parsed = JSON.parse(text); } catch { parsed = text; }
  if (!res.ok) {
    console.error(`HTTP ${res.status} ${res.statusText}`);
    console.error(typeof parsed === 'string' ? parsed : JSON.stringify(parsed, null, 2));
    process.exit(1);
  }
  return parsed;
}

function output(data) {
  console.log(typeof data === 'string' ? data : JSON.stringify(data, null, 2));
}

function pickFields(obj, fieldsStr) {
  if (!fieldsStr) return obj;
  const fields = fieldsStr.split(',').map(f => f.trim());
  const result = {};
  for (const f of fields) { result[f] = obj[f] !== undefined ? obj[f] : null; }
  return result;
}

function slimIssue(issue) {
  const { documentId, title, status, category, priority, complexity,
    description, acceptanceCriteria, aiAcceptanceCriteria, suggestedSolution,
    plan, previewUrl, previewApiUrl, changeHistory } = issue;
  return { documentId, title, status, category, priority, complexity,
    description, acceptanceCriteria, aiAcceptanceCriteria, suggestedSolution,
    plan, previewUrl, previewApiUrl, changeHistory };
}

async function getIssue(docId, flags) {
  const result = await api('GET', `/issues?filters[documentId][$eq]=${encodeURIComponent(docId)}&populate=*`);
  const issue = result?.data?.[0];
  if (!issue) { console.error(`Issue ${docId} not found`); process.exit(1); }
  if (flags.fields) { output(pickFields(issue, flags.fields)); }
  else if (flags.raw) { output(issue); }
  else { output(slimIssue(issue)); }
}

async function updateIssue(docId, flags) {
  let data = {};
  if (flags['data-file']) { data = JSON.parse(readFileSync(flags['data-file'], 'utf-8')); }
  else {
    for (const f of ['status','category','priority','complexity','plan','title','description','acceptanceCriteria','suggestedSolution','relations','sessionContext']) {
      if (flags[f] !== undefined) data[f] = resolveValue(flags[f]);
    }
  }
  if (!Object.keys(data).length) { console.error('No fields to update.'); process.exit(1); }
  output(await api('PUT', `/issues/${docId}`, { data }));
}

async function searchIssues(terms, flags) {
  const exclude = flags.exclude || '', limit = flags.limit || 10;
  const keywords = terms.split(/\s+/).filter(Boolean);
  const filters = [];
  for (const kw of keywords) {
    filters.push(`filters[$or][${filters.length}][title][$containsi]=${encodeURIComponent(kw)}`);
    filters.push(`filters[$or][${filters.length}][description][$containsi]=${encodeURIComponent(kw)}`);
  }
  let qs = filters.join('&');
  if (exclude) qs += `&filters[documentId][$ne]=${encodeURIComponent(exclude)}`;
  qs += `&pagination[pageSize]=${limit}`;
  output((await api('GET', `/issues?${qs}`))?.data || []);
}

async function listComments(issueDocId, flags) {
  const limit = flags.limit || 10;
  const data = (await api('GET', `/comments?filters[issue][documentId][$eq]=${encodeURIComponent(issueDocId)}&sort=createdAt:desc&pagination[pageSize]=${limit}`))?.data || [];
  if (flags.raw) { output(data); return; }
  output(data.map(c => ({ author: c.author, body: c.body, createdAt: c.createdAt })));
}

async function createComment(issueDocId, flags) {
  let body, author, attachments;
  if (flags['data-file']) {
    const content = JSON.parse(readFileSync(flags['data-file'], 'utf-8'));
    body = content.body; author = content.author || flags.author; attachments = content.attachments;
  } else {
    if (!flags.body) { console.error('--body or --data-file required'); process.exit(1); }
    body = resolveValue(flags.body); author = flags.author;
  }
  if (flags.attachments && !attachments) {
    attachments = String(flags.attachments).split(',').map(Number).filter(n => !isNaN(n));
  }
  if (!author) { console.error('--author is required'); process.exit(1); }
  const data = { body, issue: issueDocId, author };
  if (attachments?.length) data.attachments = attachments;
  output(await api('POST', '/comments', { data }));
}

async function uploadFile(filePath) {
  if (!existsSync(filePath)) { console.error(`File not found: ${filePath}`); process.exit(1); }
  const fileBuffer = readFileSync(filePath);
  const fileName = basename(filePath);
  const ext = extname(filePath).toLowerCase();
  const mimeTypes = {
    '.png':'image/png','.jpg':'image/jpeg','.jpeg':'image/jpeg',
    '.gif':'image/gif','.webp':'image/webp','.svg':'image/svg+xml',
    '.pdf':'application/pdf','.json':'application/json',
    '.txt':'text/plain','.md':'text/markdown',
  };
  const form = new FormData();
  form.append('file', new Blob([fileBuffer], { type: mimeTypes[ext] || 'application/octet-stream' }), fileName);
  const res = await fetch(`${baseUrl}/comments/upload`, { method: 'POST', headers: { 'x-forge-api-key': apiKey }, body: form });
  const text = await res.text();
  let parsed; try { parsed = JSON.parse(text); } catch { parsed = text; }
  if (!res.ok) { console.error(`HTTP ${res.status} ${res.statusText}`); console.error(typeof parsed === 'string' ? parsed : JSON.stringify(parsed, null, 2)); process.exit(1); }
  output(parsed);
}

const { flags, positional } = parseFlags(args.slice(1));

switch (command) {
  case 'get-issue':
    if (!positional[0]) { console.error('Usage: get-issue <documentId> [--fields=f1,f2]'); process.exit(1); }
    await getIssue(positional[0], flags); break;
  case 'update-issue':
    if (!positional[0]) { console.error('Usage: update-issue <documentId> --field=value'); process.exit(1); }
    await updateIssue(positional[0], flags); break;
  case 'search-issues':
    if (!positional[0]) { console.error('Usage: search-issues "keywords" --exclude=<id>'); process.exit(1); }
    await searchIssues(positional[0], flags); break;
  case 'list-comments':
    if (!positional[0]) { console.error('Usage: list-comments <issueDocId> --limit=5'); process.exit(1); }
    await listComments(positional[0], flags); break;
  case 'create-comment':
    if (!positional[0]) { console.error('Usage: create-comment <issueDocId> --body=@file.md --author=Name'); process.exit(1); }
    await createComment(positional[0], flags); break;
  case 'upload':
    if (!positional[0]) { console.error('Usage: upload <filepath>'); process.exit(1); }
    await uploadFile(positional[0]); break;
  default:
    console.error(`Unknown command: ${command}`);
    console.error('Commands: get-issue, update-issue, search-issues, list-comments, create-comment, upload');
    process.exit(1);
}

import { execSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, "..");
const ENV_CANDIDATES = [
  resolve(ROOT, ".env.local"),
  resolve(ROOT, ".env"),
];

function loadEnv() {
  const ENV_PATH = ENV_CANDIDATES.find((p) => existsSync(p));
  if (!ENV_PATH) {
    console.error("No .env found in web/ or project root");
    process.exit(1);
  }
  for (const line of readFileSync(ENV_PATH, "utf-8").split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const eq = trimmed.indexOf("=");
    if (eq === -1) continue;
    const key = trimmed.slice(0, eq).trim();
    const val = trimmed.slice(eq + 1).trim();
    if (!process.env[key]) process.env[key] = val;
  }
}

loadEnv();

const ACCESS_TOKEN = process.env.SUPABASE_ACCESS_TOKEN;
const TYPES_PATH = resolve(ROOT, process.env.SUPABASE_TYPES_PATH || "src/types/database.types.ts");

function getRef() {
  return process.env.SUPABASE_PROJECT_REF || extractRef(process.env.SUPABASE_URL);
}

function extractRef(url) {
  if (!url) return undefined;
  const m = url.match(/\/\/([a-z]+)\.supabase\.co/);
  return m ? m[1] : undefined;
}

function requireRef() {
  const ref = getRef();
  if (!ref) {
    console.error("Set SUPABASE_PROJECT_REF or SUPABASE_URL in .env");
    process.exit(1);
  }
  return ref;
}

function run(cmd, opts = {}) {
  const env = { ...process.env };
  if (ACCESS_TOKEN) env.SUPABASE_ACCESS_TOKEN = ACCESS_TOKEN;
  const execOpts = { cwd: ROOT, env };
  if (opts.capture) {
    return execSync(cmd, { encoding: "utf-8", ...execOpts });
  }
  execSync(cmd, { stdio: "inherit", ...execOpts });
}

function cmdTypes() {
  const ref = requireRef();
  console.log(`Generating types for project ${ref}...`);
  const output = run(`npx supabase gen types typescript --project-id ${ref} --schema public`, { capture: true });
  mkdirSync(dirname(TYPES_PATH), { recursive: true });
  writeFileSync(TYPES_PATH, output);
  console.log(`Types written to ${TYPES_PATH}`);
}

function cmdDeploy(args) {
  const ref = requireRef();
  const fnDir = resolve(ROOT, "supabase", "functions");

  if (args[0] === "--all" || args.length === 0) {
    if (!existsSync(fnDir)) { console.error("No supabase/functions/ directory found"); process.exit(1); }
    const fns = readdirSync(fnDir).filter((d) => existsSync(resolve(fnDir, d, "index.ts")));
    if (fns.length === 0) { console.log("No edge functions found"); return; }
    for (const name of fns) {
      console.log(`Deploying ${name}...`);
      run(`npx supabase functions deploy ${name} --project-ref ${ref}`);
    }
  } else {
    for (const name of args) {
      console.log(`Deploying ${name}...`);
      run(`npx supabase functions deploy ${name} --project-ref ${ref}`);
    }
  }
}

function cmdSecrets(args) {
  const ref = requireRef();

  if (args.length === 0 || args[0] === "set") {
    const kvPairs = args.slice(args[0] === "set" ? 1 : 0);
    if (kvPairs.length === 0) {
      const secret = process.env.SUPABASE_SECRET_KEY;
      if (!secret) { console.error("Set SUPABASE_SECRET_KEY in .env"); process.exit(1); }
      console.log("Setting SB_SECRET_KEY...");
      run(`npx supabase secrets set SB_SECRET_KEY=${secret} --project-ref ${ref}`);
    } else {
      run(`npx supabase secrets set ${kvPairs.join(" ")} --project-ref ${ref}`);
    }
  } else if (args[0] === "list") {
    run(`npx supabase secrets list --project-ref ${ref}`);
  } else if (args[0] === "unset") {
    if (args.length < 2) { console.error("Usage: sb secrets unset <KEY> [...]"); process.exit(1); }
    run(`npx supabase secrets unset ${args.slice(1).join(" ")} --project-ref ${ref}`);
  } else {
    console.error(`Unknown secrets sub-command: ${args[0]}`);
    process.exit(1);
  }
}

function cmdPush() { run(`npx supabase db push --project-ref ${requireRef()}`); }
function cmdPull() { run(`npx supabase db pull --project-ref ${requireRef()}`); }
function cmdDiff() { run(`npx supabase db diff --project-ref ${requireRef()}`); }
function cmdLink() { run(`npx supabase link --project-ref ${requireRef()}`); }

function cmdStatus() {
  console.log(`Project ref:   ${getRef() || "(not set)"}`);
  console.log(`URL:           ${process.env.SUPABASE_URL || "(not set)"}`);
  console.log(`Access token:  ${ACCESS_TOKEN ? "configured" : "MISSING"}`);
  console.log(`Secret key:    ${process.env.SUPABASE_SECRET_KEY ? "configured" : "(not set)"}`);
  console.log(`Types path:    ${TYPES_PATH}`);
  console.log();
  run("npx supabase projects list");
}

function cmdHelp() {
  console.log(`
sb.mjs — Supabase CLI wrapper for bbgo-saas

Usage: node scripts/sb.mjs <command> [args]

Commands:
  types                Generate TypeScript types
  deploy [name|--all]  Deploy edge functions (default: --all)
  secrets [set|list|unset]  Manage secrets (no args = set SB_SECRET_KEY)
  push                 Push migrations to remote
  pull                 Pull remote schema to local
  diff                 Diff local migrations against remote
  link                 Link local project to remote
  status               Show project config
  help                 Show this help

Environment (.env):
  SUPABASE_PROJECT_REF   Project ref (or auto-detected from URL)
  SUPABASE_ACCESS_TOKEN  CLI auth token (required)
  SUPABASE_SECRET_KEY    Auto-set as SB_SECRET_KEY
  SUPABASE_TYPES_PATH    Output path (default: src/types/database.types.ts)
  SUPABASE_URL           Auto-detect project ref from this

Examples:
  pnpm sb types
  pnpm sb deploy --all
  pnpm sb secrets list
  pnpm sb push
`);
}

const [,, command, ...args] = process.argv;
const handlers = {
  types: cmdTypes,
  deploy: () => cmdDeploy(args),
  secrets: () => cmdSecrets(args),
  push: cmdPush,
  pull: cmdPull,
  diff: cmdDiff,
  link: cmdLink,
  status: cmdStatus,
  help: cmdHelp,
};

const handler = handlers[command];
if (!handler) {
  console.error(`Unknown command: ${command || "(none)"}`);
  cmdHelp();
  process.exit(1);
}
handler();

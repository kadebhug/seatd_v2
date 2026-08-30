import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const specPath = resolve(scriptDir, "../openapi/seatd.v1.json");
const spec = JSON.parse(readFileSync(specPath, "utf8"));

if (spec.openapi !== "3.1.0") {
  throw new Error("OpenAPI contract must use version 3.1.0");
}
if (!spec.paths || Object.keys(spec.paths).length === 0) {
  throw new Error("OpenAPI contract must define at least one path");
}
if (!spec.components?.schemas?.ServiceStatus) {
  throw new Error("OpenAPI contract must define ServiceStatus");
}

console.log(`validated ${specPath}`);

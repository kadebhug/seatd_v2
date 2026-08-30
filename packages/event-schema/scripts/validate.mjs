import { readdirSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const schemaPath = resolve(scriptDir, "../schemas/v1/operational-event.schema.json");
const schema = JSON.parse(readFileSync(schemaPath, "utf8"));

if (schema.$schema !== "https://json-schema.org/draft/2020-12/schema") {
  throw new Error("event schemas must use JSON Schema draft 2020-12");
}
if (!schema.$id || !schema.$id.includes("/v1/")) {
  throw new Error("event schema $id must include its immutable version");
}
if (schema.additionalProperties !== false) {
  throw new Error("event schema must reject undeclared fields");
}
if (schema.properties.schemaVersion.const !== 1) {
  throw new Error("event schema v1 must pin schemaVersion to 1");
}
const allowedTypes = new Set(schema.properties.type.enum);
const required = new Set(schema.required);
const fixtureDir = resolve(scriptDir, "../fixtures/v1");
for (const name of readdirSync(fixtureDir)) {
  if (!name.endsWith(".json")) {
    continue;
  }
  const fixturePath = resolve(fixtureDir, name);
  const fixture = JSON.parse(readFileSync(fixturePath, "utf8"));
  for (const key of required) {
    if (!(key in fixture)) {
      throw new Error(`${name} is missing required field ${key}`);
    }
  }
  if (!allowedTypes.has(fixture.type)) {
    throw new Error(`${name} uses unsupported event type ${fixture.type}`);
  }
  if (fixture.schemaVersion !== 1) {
    throw new Error(`${name} must use schemaVersion 1`);
  }
  if (typeof fixture.data !== "object" || fixture.data === null || Array.isArray(fixture.data)) {
    throw new Error(`${name} data must be an object`);
  }
}

console.log(`validated ${schemaPath}`);

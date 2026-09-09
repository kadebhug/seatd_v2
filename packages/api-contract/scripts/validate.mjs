import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const specPath = resolve(scriptDir, "../openapi/seatd.v1.json");
const eventSchemaPath = resolve(scriptDir, "../../event-schema/schemas/v1/operational-event.schema.json");
const spec = JSON.parse(readFileSync(specPath, "utf8"));
const eventSchema = JSON.parse(readFileSync(eventSchemaPath, "utf8"));

if (spec.openapi !== "3.1.0") {
  throw new Error("OpenAPI contract must use version 3.1.0");
}
if (!spec.paths || Object.keys(spec.paths).length === 0) {
  throw new Error("OpenAPI contract must define at least one path");
}
if (!spec.components?.schemas?.ServiceStatus) {
  throw new Error("OpenAPI contract must define ServiceStatus");
}
for (const path of [
  "/v1/realtime",
  "/v1/realtime/metrics",
  "/v1/sync/location-snapshot",
  "/v1/sync/events",
]) {
  if (!spec.paths[path]) {
    throw new Error(`OpenAPI contract must define ${path}`);
  }
}
for (const name of [
  "RealtimeMessage",
  "RealtimeMetricsResponse",
  "SyncEventsResponse",
]) {
  if (!spec.components?.schemas?.[name]) {
    throw new Error(`OpenAPI contract must define ${name}`);
  }
}

const eventTypes = eventSchema.properties?.type?.enum;
const realtimeTypes = spec.components.schemas.RealtimeMessage.properties?.type?.enum;
if (!Array.isArray(eventTypes) || eventTypes.length === 0) {
  throw new Error("event schema must define event type enum");
}
if (JSON.stringify(realtimeTypes) !== JSON.stringify(eventTypes)) {
  throw new Error("RealtimeMessage.type enum must match event schema type enum");
}

console.log(`validated ${specPath}`);

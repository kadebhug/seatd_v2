import type { LayoutEditorSnapshotResponse } from "@seatd/typescript-seatd-client";
import { seatdFetch } from "../../../lib/seatd-api";
import { FloorEditor } from "./floor-editor";

export const dynamic = "force-dynamic";

export default async function LayoutPage() {
  const snapshot = await seatdFetch<LayoutEditorSnapshotResponse>(
    "/v1/layout/editor-snapshot",
  );

  return (
    <main className="page editor-page">
      <section className="page-heading">
        <p>Floor editor</p>
        <h1>Layout and live table preview</h1>
      </section>
      <FloorEditor initialSnapshot={snapshot} />
    </main>
  );
}

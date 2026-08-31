"use client";

import type {
  Floor,
  FloorRequest,
  LayoutEditorSnapshotResponse,
  Table,
  TableRequest,
  TableState,
  Zone,
  ZoneRequest,
} from "@seatd/typescript-seatd-client";
import { useMemo, useState } from "react";
import { occupancyLabel } from "../../../lib/occupancy";

type Geometry = {
  x: number;
  y: number;
  width?: number;
  height?: number;
  radius?: number;
  rotation?: number;
};

const defaultCanvas = { width: 1200, height: 800, unit: "px" };

export function FloorEditor({
  initialSnapshot,
}: Readonly<{ initialSnapshot: LayoutEditorSnapshotResponse }>) {
  const [snapshot, setSnapshot] = useState(initialSnapshot);
  const [selectedFloorId, setSelectedFloorId] = useState(
    initialSnapshot.floors[0]?.id ?? "",
  );
  const [selectedTableId, setSelectedTableId] = useState(
    initialSnapshot.tables[0]?.table.id ?? "",
  );
  const [status, setStatus] = useState("");
  const [statusIsError, setStatusIsError] = useState(false);

  const selectedFloor =
    snapshot.floors.find((floor) => floor.id === selectedFloorId) ??
    snapshot.floors[0];
  const zones = useMemo(
    () => snapshot.zones.filter((zone) => zone.floorId === selectedFloor?.id),
    [snapshot.zones, selectedFloor?.id],
  );
  const tableStates = useMemo(
    () =>
      snapshot.tables.filter(
        (state) => state.table.floorId === selectedFloor?.id,
      ),
    [snapshot.tables, selectedFloor?.id],
  );
  const selectedState =
    snapshot.tables.find((state) => state.table.id === selectedTableId) ??
    tableStates[0];
  const canvas = canvasSize(selectedFloor);

  async function refresh() {
    const response = await fetch("/api/owner/layout-snapshot");
    if (response.ok) {
      setSnapshot((await response.json()) as LayoutEditorSnapshotResponse);
    }
  }

  async function request<T>(
    path: string,
    method: string,
    body: unknown,
  ): Promise<T | null> {
    setStatus("");
    setStatusIsError(false);
    const response = await fetch(path, {
      method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      setStatus(await readErrorMessage(response));
      setStatusIsError(true);
      return null;
    }
    setStatus("Saved.");
    return response.json() as Promise<T>;
  }

  function replaceFloor(floor: Floor) {
    setSnapshot((current) => ({
      ...current,
      floors: current.floors.map((item) =>
        item.id === floor.id ? floor : item,
      ),
    }));
  }

  function replaceTable(table: Table) {
    setSnapshot((current) => ({
      ...current,
      tables: current.tables.map((item) =>
        item.table.id === table.id ? { ...item, table } : item,
      ),
    }));
  }

  function moveTable(table: Table, x: number, y: number) {
    replaceTable({ ...table, geometry: { ...geometry(table), x, y } });
  }

  async function saveTable(table: Table) {
    const saved = await request<Table>(
      `/api/owner/tables/${table.id}`,
      "PUT",
      tableRequest(table),
    );
    if (saved) {
      replaceTable(saved);
    }
  }

  async function toggleFloorArchive(floor: Floor) {
    if (floor.isActive) {
      if (
        !window.confirm(
          `Archive floor "${floor.name}"? Tables on this floor stay in history but leave active service.`,
        )
      ) {
        return;
      }
    }
    const saved = await request<Floor>(
      `/api/owner/floors/${floor.id}/${floor.isActive ? "archive" : "restore"}`,
      "POST",
      { expectedVersion: floor.version },
    );
    if (saved) {
      replaceFloor(saved);
    }
  }

  return (
    <div className="editor-grid">
      <aside className="panel stack">
        <h2>Floors</h2>
        <div className="list-buttons">
          {snapshot.floors.map((floor) => (
            <button
              aria-pressed={floor.id === selectedFloor?.id}
              className={floor.id === selectedFloor?.id ? "selected" : ""}
              key={floor.id}
              onClick={() => setSelectedFloorId(floor.id)}
              type="button"
            >
              {floor.name}
            </button>
          ))}
        </div>
        <form
          className="form"
          onSubmit={async (event) => {
            event.preventDefault();
            const data = new FormData(event.currentTarget);
            const body: FloorRequest = {
              slug: String(data.get("slug") || "new-floor"),
              name: String(data.get("name") || "New floor"),
              sortOrder: Number(data.get("sortOrder") || 0),
              canvas: defaultCanvas,
              backgroundAssetRef: String(data.get("backgroundAssetRef") || ""),
            };
            const floor = await request<Floor>(
              "/api/owner/floors",
              "POST",
              body,
            );
            if (floor) {
              setSnapshot((current) => ({
                ...current,
                floors: [...current.floors, floor],
              }));
              setSelectedFloorId(floor.id);
            }
          }}
        >
          <h3>Add Floor</h3>
          <label>
            Name
            <input name="name" placeholder="Main dining…" required />
          </label>
          <label>
            Slug
            <input
              autoComplete="off"
              name="slug"
              placeholder="main-dining…"
              required
              spellCheck={false}
            />
          </label>
          <label>
            Sort order
            <input defaultValue={0} name="sortOrder" type="number" />
          </label>
          <label>
            Background asset ref
            <input
              autoComplete="off"
              name="backgroundAssetRef"
              placeholder="Optional asset ref…"
              spellCheck={false}
            />
          </label>
          <button type="submit">Add Floor</button>
        </form>
        {selectedFloor ? (
          <button
            className={selectedFloor.isActive ? "danger" : "secondary"}
            onClick={() => void toggleFloorArchive(selectedFloor)}
            type="button"
          >
            {selectedFloor.isActive ? "Archive Floor" : "Restore Floor"}
          </button>
        ) : null}
      </aside>

      <section aria-label="Floor canvas" className="canvas-panel">
        <div
          className="floor-canvas"
          style={{ aspectRatio: `${canvas.width} / ${canvas.height}` }}
        >
          {tableStates.map((state) => (
            <TableTile
              canvas={canvas}
              key={state.table.id}
              onMove={moveTable}
              onSelect={() => setSelectedTableId(state.table.id)}
              selected={state.table.id === selectedState?.table.id}
              state={state}
            />
          ))}
        </div>
      </section>

      <aside className="panel stack">
        <h2>Objects</h2>
        {selectedState ? (
          <TableForm
            onArchive={async (table) => {
              if (table.isActive) {
                if (
                  !window.confirm(
                    `Archive table "${table.label}"? It will leave the active floor plan.`,
                  )
                ) {
                  return;
                }
              }
              const saved = await request<Table>(
                `/api/owner/tables/${table.id}/${table.isActive ? "archive" : "restore"}`,
                "POST",
                { expectedVersion: table.version },
              );
              if (saved) {
                replaceTable(saved);
              }
            }}
            onSave={saveTable}
            table={selectedState.table}
            zones={zones}
          />
        ) : (
          <p className="empty-state">Select a table to edit its properties.</p>
        )}
        {selectedFloor ? (
          <>
            <ZoneForm
              floor={selectedFloor}
              onCreate={async (body) => {
                const zone = await request<Zone>(
                  "/api/owner/zones",
                  "POST",
                  body,
                );
                if (zone) {
                  setSnapshot((current) => ({
                    ...current,
                    zones: [...current.zones, zone],
                  }));
                }
              }}
            />
            <AddTableForm
              floor={selectedFloor}
              onCreate={async (body) => {
                const table = await request<Table>(
                  "/api/owner/tables",
                  "POST",
                  body,
                );
                if (table) {
                  await refresh();
                  setSelectedTableId(table.id);
                }
              }}
              zones={zones}
            />
          </>
        ) : null}
        {status ? (
          <p
            aria-live="polite"
            className="toast"
            role={statusIsError ? "alert" : "status"}
          >
            {status}
          </p>
        ) : null}
      </aside>
    </div>
  );
}

function TableTile({
  state,
  selected,
  canvas,
  onMove,
  onSelect,
}: Readonly<{
  state: TableState;
  selected: boolean;
  canvas: { width: number; height: number };
  onMove: (table: Table, x: number, y: number) => void;
  onSelect: () => void;
}>) {
  const table = state.table;
  const geom = geometry(table);
  const width = geom.width ?? (geom.radius ? geom.radius * 2 : 90);
  const height = geom.height ?? (geom.radius ? geom.radius * 2 : 70);
  const label = occupancyLabel(state.occupancy.status);

  return (
    <button
      aria-label={`${table.label}, ${label}`}
      aria-pressed={selected}
      className={`table-tile ${table.shape} ${selected ? "selected" : ""} ${state.occupancy.status}`}
      onClick={onSelect}
      onPointerDown={(event) => {
        event.currentTarget.setPointerCapture(event.pointerId);
      }}
      onPointerMove={(event) => {
        if (event.buttons !== 1) return;
        const box = event.currentTarget.parentElement?.getBoundingClientRect();
        if (!box) return;
        onMove(
          table,
          Math.max(
            0,
            Math.round(
              ((event.clientX - box.left) / box.width) * canvas.width -
                width / 2,
            ),
          ),
          Math.max(
            0,
            Math.round(
              ((event.clientY - box.top) / box.height) * canvas.height -
                height / 2,
            ),
          ),
        );
      }}
      style={{
        left: `${(geom.x / canvas.width) * 100}%`,
        top: `${(geom.y / canvas.height) * 100}%`,
        width: `${(width / canvas.width) * 100}%`,
        height: `${(height / canvas.height) * 100}%`,
        transform: `rotate(${geom.rotation ?? 0}deg)`,
      }}
      type="button"
    >
      <strong>{table.label}</strong>
      <span>{label}</span>
    </button>
  );
}

function TableForm({
  table,
  zones,
  onSave,
  onArchive,
}: Readonly<{
  table: Table;
  zones: Zone[];
  onSave: (table: Table) => Promise<void>;
  onArchive: (table: Table) => Promise<void>;
}>) {
  const geom = geometry(table);
  return (
    <form
      className="form"
      onSubmit={(event) => {
        event.preventDefault();
        const data = new FormData(event.currentTarget);
        void onSave({
          ...table,
          zoneId: String(data.get("zoneId")),
          label: String(data.get("label")),
          capacityLabel: String(data.get("capacityLabel")),
          shape: String(data.get("shape")) as Table["shape"],
          geometry: {
            x: Number(data.get("x")),
            y: Number(data.get("y")),
            width: Number(data.get("width")),
            height: Number(data.get("height")),
            rotation: Number(data.get("rotation")),
          },
        });
      }}
    >
      <h3>Edit Table</h3>
      <label>
        Label
        <input defaultValue={table.label} name="label" required />
      </label>
      <label>
        Capacity label
        <input
          defaultValue={table.capacityLabel}
          name="capacityLabel"
          required
        />
      </label>
      <label>
        Shape
        <select defaultValue={table.shape} name="shape">
          <option value="rectangle">Rectangle</option>
          <option value="circle">Circle</option>
          <option value="square">Square</option>
          <option value="custom">Custom</option>
        </select>
      </label>
      <label>
        Zone
        <select defaultValue={table.zoneId} name="zoneId">
          {zones.map((zone) => (
            <option key={zone.id} value={zone.id}>
              {zone.name}
            </option>
          ))}
        </select>
      </label>
      <fieldset className="number-grid">
        <legend>Position and size</legend>
        <label>
          X
          <input defaultValue={geom.x} name="x" type="number" />
        </label>
        <label>
          Y
          <input defaultValue={geom.y} name="y" type="number" />
        </label>
        <label>
          Width
          <input defaultValue={geom.width ?? 100} name="width" type="number" />
        </label>
        <label>
          Height
          <input defaultValue={geom.height ?? 70} name="height" type="number" />
        </label>
        <label>
          Rotation
          <input
            defaultValue={geom.rotation ?? 0}
            name="rotation"
            type="number"
          />
        </label>
      </fieldset>
      <button type="submit">Save Table</button>
      <button
        className={table.isActive ? "danger" : "secondary"}
        onClick={() => void onArchive(table)}
        type="button"
      >
        {table.isActive ? "Archive Table" : "Restore Table"}
      </button>
    </form>
  );
}

function ZoneForm({
  floor,
  onCreate,
}: Readonly<{ floor: Floor; onCreate: (body: ZoneRequest) => Promise<void> }>) {
  return (
    <form
      className="form"
      onSubmit={(event) => {
        event.preventDefault();
        const data = new FormData(event.currentTarget);
        void onCreate({
          floorId: floor.id,
          name: String(data.get("name") || "New zone"),
          sortOrder: Number(data.get("sortOrder") || 0),
        });
      }}
    >
      <h3>Add Zone</h3>
      <label>
        Name
        <input name="name" placeholder="Patio…" required />
      </label>
      <label>
        Sort order
        <input defaultValue={0} name="sortOrder" type="number" />
      </label>
      <button type="submit">Add Zone</button>
    </form>
  );
}

function AddTableForm({
  floor,
  zones,
  onCreate,
}: Readonly<{
  floor: Floor;
  zones: Zone[];
  onCreate: (body: TableRequest) => Promise<void>;
}>) {
  return (
    <form
      className="form"
      onSubmit={(event) => {
        event.preventDefault();
        const data = new FormData(event.currentTarget);
        void onCreate({
          floorId: floor.id,
          zoneId: String(data.get("zoneId")),
          label: String(data.get("label") || "T"),
          capacityLabel: String(data.get("capacityLabel") || "2"),
          shape: "rectangle",
          geometry: { x: 80, y: 80, width: 110, height: 80, rotation: 0 },
        });
      }}
    >
      <h3>Add Table</h3>
      <label>
        Label
        <input name="label" placeholder="T12…" required />
      </label>
      <label>
        Capacity label
        <input name="capacityLabel" placeholder="4…" required />
      </label>
      <label>
        Zone
        <select defaultValue={zones[0]?.id} name="zoneId" required>
          {zones.map((zone) => (
            <option key={zone.id} value={zone.id}>
              {zone.name}
            </option>
          ))}
        </select>
      </label>
      <button disabled={zones.length === 0} type="submit">
        Add Table
      </button>
    </form>
  );
}

async function readErrorMessage(response: Response): Promise<string> {
  const body = await response.text();
  try {
    const parsed = JSON.parse(body) as { error?: { message?: string } };
    if (parsed.error?.message) {
      return parsed.error.message;
    }
  } catch {
    // non-JSON error bodies still get shown
  }
  return body.trim() || "Request failed.";
}

function tableRequest(table: Table): TableRequest {
  return {
    floorId: table.floorId,
    zoneId: table.zoneId,
    label: table.label,
    capacityLabel: table.capacityLabel,
    shape: table.shape,
    geometry: table.geometry,
    expectedVersion: table.version,
  };
}

function geometry(table: Table): Geometry {
  return table.geometry as Geometry;
}

function canvasSize(floor?: Floor): { width: number; height: number } {
  const canvas = floor?.canvas ?? defaultCanvas;
  return {
    width: Number(canvas.width ?? defaultCanvas.width),
    height: Number(canvas.height ?? defaultCanvas.height),
  };
}

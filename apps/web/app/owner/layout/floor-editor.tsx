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
    const response = await fetch(path, {
      method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      setStatus(await response.text());
      return null;
    }
    setStatus("Saved");
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

  function replaceZone(zone: Zone) {
    setSnapshot((current) => ({
      ...current,
      zones: current.zones.map((item) => (item.id === zone.id ? zone : item)),
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

  return (
    <div className="editor-grid">
      <aside className="panel stack">
        <h2>Floors</h2>
        <div className="list-buttons">
          {snapshot.floors.map((floor) => (
            <button
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
          <h3>Add floor</h3>
          <input name="name" placeholder="Name" />
          <input name="slug" placeholder="slug" />
          <input name="sortOrder" placeholder="0" type="number" />
          <input name="backgroundAssetRef" placeholder="background asset ref" />
          <button type="submit">Add floor</button>
        </form>
        {selectedFloor ? (
          <button
            type="button"
            onClick={async () => {
              const floor = await request<Floor>(
                `/api/owner/floors/${selectedFloor.id}/${selectedFloor.isActive ? "archive" : "restore"}`,
                "POST",
                { expectedVersion: selectedFloor.version },
              );
              if (floor) replaceFloor(floor);
            }}
          >
            {selectedFloor.isActive ? "Archive floor" : "Restore floor"}
          </button>
        ) : null}
      </aside>

      <section className="canvas-panel">
        <div
          className="floor-canvas"
          style={{ aspectRatio: `${canvas.width} / ${canvas.height}` }}
        >
          {tableStates.map((state) => (
            <TableTile
              key={state.table.id}
              selected={state.table.id === selectedState?.table.id}
              state={state}
              canvas={canvas}
              onMove={moveTable}
              onSelect={() => setSelectedTableId(state.table.id)}
            />
          ))}
        </div>
      </section>

      <aside className="panel stack">
        <h2>Objects</h2>
        {selectedState ? (
          <TableForm
            table={selectedState.table}
            zones={zones}
            onSave={saveTable}
            onArchive={async (table) => {
              const saved = await request<Table>(
                `/api/owner/tables/${table.id}/${table.isActive ? "archive" : "restore"}`,
                "POST",
                { expectedVersion: table.version },
              );
              if (saved) replaceTable(saved);
            }}
          />
        ) : null}
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
                if (zone)
                  setSnapshot((current) => ({
                    ...current,
                    zones: [...current.zones, zone],
                  }));
              }}
            />
            <AddTableForm
              floor={selectedFloor}
              zones={zones}
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
            />
          </>
        ) : null}
        {status ? <p className="toast">{status}</p> : null}
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

  return (
    <button
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
      <span>{state.occupancy.status}</span>
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
      <h3>Edit table</h3>
      <input name="label" defaultValue={table.label} />
      <input name="capacityLabel" defaultValue={table.capacityLabel} />
      <select name="shape" defaultValue={table.shape}>
        <option value="rectangle">Rectangle</option>
        <option value="circle">Circle</option>
        <option value="square">Square</option>
        <option value="custom">Custom</option>
      </select>
      <select name="zoneId" defaultValue={table.zoneId}>
        {zones.map((zone) => (
          <option key={zone.id} value={zone.id}>
            {zone.name}
          </option>
        ))}
      </select>
      <div className="number-grid">
        <input name="x" defaultValue={geom.x} type="number" />
        <input name="y" defaultValue={geom.y} type="number" />
        <input name="width" defaultValue={geom.width ?? 100} type="number" />
        <input name="height" defaultValue={geom.height ?? 70} type="number" />
        <input
          name="rotation"
          defaultValue={geom.rotation ?? 0}
          type="number"
        />
      </div>
      <button type="submit">Save table</button>
      <button type="button" onClick={() => void onArchive(table)}>
        {table.isActive ? "Archive table" : "Restore table"}
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
      <h3>Add zone</h3>
      <input name="name" placeholder="Zone name" />
      <input name="sortOrder" placeholder="0" type="number" />
      <button type="submit">Add zone</button>
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
      <h3>Add table</h3>
      <input name="label" placeholder="Label" />
      <input name="capacityLabel" placeholder="Capacity" />
      <select name="zoneId" defaultValue={zones[0]?.id}>
        {zones.map((zone) => (
          <option key={zone.id} value={zone.id}>
            {zone.name}
          </option>
        ))}
      </select>
      <button disabled={zones.length === 0} type="submit">
        Add table
      </button>
    </form>
  );
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

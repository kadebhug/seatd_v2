export type Occupancy = "available" | "occupied" | "attention";
export type TableShape = "rectangle" | "circle" | "square";
export type ZoneId = "dining" | "patio" | "bar" | "lounge";
export type FixtureKind = "host" | "aisle" | "counter" | "garden";

export type DemoTable = {
  id: string;
  label: string;
  capacity: string;
  zone: ZoneId;
  shape: TableShape;
  x: number;
  y: number;
  w: number;
  h: number;
  status: Occupancy;
};

export type FloorFixture = {
  id: string;
  kind: FixtureKind;
  label: string;
  x: number;
  y: number;
  w: number;
  h: number;
};

export type FloorLayout = {
  tables: DemoTable[];
  fixtures: FloorFixture[];
};

export type DemoZone = {
  id: ZoneId;
  name: string;
  hint: string;
};

export const DEMO_VENUE = "Marlowe's";

export const ZONES: DemoZone[] = [
  { id: "dining", name: "Main floor", hint: "Window and centre dining" },
  { id: "patio", name: "Patio", hint: "Covered outdoor seating" },
  { id: "bar", name: "Bar", hint: "High-tops and stools" },
  { id: "lounge", name: "Lounge", hint: "Upstairs seating" },
];

export const GUEST_ACTIONS = [
  { key: "call_waiter", label: "Call waiter" },
  { key: "request_bill", label: "Request bill" },
  { key: "request_water", label: "Request water" },
  { key: "request_service", label: "Request service" },
] as const;

export const MAIN_FIXTURES: FloorFixture[] = [
  { id: "host", kind: "host", label: "Host", x: 22, y: 5, w: 24, h: 12 },
  { id: "aisle", kind: "aisle", label: "Service aisle", x: 43, y: 20, w: 14, h: 52 },
];

export const MAIN_FLOOR: DemoTable[] = [
  t("T18", "2", "dining", "circle", 52, 5, 14, 16, "available"),
  t("T02", "2", "dining", "circle", 10, 24, 16, 18, "occupied"),
  t("T04", "4", "dining", "rectangle", 64, 24, 20, 18, "available"),
  t("T06", "4", "dining", "rectangle", 8, 50, 22, 20, "occupied"),
  t("T08", "2", "dining", "circle", 66, 50, 16, 18, "occupied"),
  t("T12", "4", "dining", "rectangle", 6, 76, 22, 18, "available"),
  t("T20", "4", "dining", "rectangle", 38, 76, 24, 20, "attention"),
  t("T14", "4", "dining", "rectangle", 66, 76, 20, 18, "occupied"),
];

export const PATIO_FIXTURES: FloorFixture[] = [
  { id: "garden", kind: "garden", label: "Garden", x: 76, y: 8, w: 18, h: 84 },
];

export const PATIO_FLOOR: DemoTable[] = [
  t("P01", "4", "patio", "rectangle", 12, 16, 22, 26, "occupied"),
  t("P02", "4", "patio", "rectangle", 44, 16, 22, 26, "available"),
  t("P03", "4", "patio", "rectangle", 12, 56, 22, 26, "occupied"),
  t("P04", "2", "patio", "rectangle", 46, 60, 18, 22, "occupied"),
];

export const BAR_FIXTURES: FloorFixture[] = [
  { id: "counter", kind: "counter", label: "Bar", x: 8, y: 10, w: 84, h: 12 },
];

export const BAR_FLOOR: DemoTable[] = [
  t("B01", "2", "bar", "circle", 12, 30, 13, 20, "occupied"),
  t("B02", "2", "bar", "circle", 33, 30, 13, 20, "available"),
  t("B03", "2", "bar", "circle", 54, 30, 13, 20, "occupied"),
  t("B04", "2", "bar", "circle", 75, 30, 13, 20, "attention"),
  t("B06", "4", "bar", "rectangle", 36, 64, 28, 22, "occupied"),
];

export const LOUNGE_FIXTURES: FloorFixture[] = [];

export const LOUNGE_FLOOR: DemoTable[] = [
  t("L01", "4", "lounge", "rectangle", 8, 14, 24, 28, "occupied"),
  t("L02", "2", "lounge", "square", 40, 22, 20, 24, "available"),
  t("L03", "4", "lounge", "rectangle", 68, 14, 24, 28, "occupied"),
  t("L04", "2", "lounge", "square", 10, 58, 22, 26, "available"),
  t("L05", "4", "lounge", "rectangle", 66, 58, 24, 28, "occupied"),
];

export const FLOOR_LAYOUTS: Record<ZoneId, FloorLayout> = {
  dining: { tables: MAIN_FLOOR, fixtures: MAIN_FIXTURES },
  patio: { tables: PATIO_FLOOR, fixtures: PATIO_FIXTURES },
  bar: { tables: BAR_FLOOR, fixtures: BAR_FIXTURES },
  lounge: { tables: LOUNGE_FLOOR, fixtures: LOUNGE_FIXTURES },
};

export const FLOORS: Record<ZoneId, DemoTable[]> = {
  dining: MAIN_FLOOR,
  patio: PATIO_FLOOR,
  bar: BAR_FLOOR,
  lounge: LOUNGE_FLOOR,
};

export const occupancyLabel: Record<Occupancy, string> = {
  available: "Available",
  occupied: "Occupied",
  attention: "Attention",
};

export function countByStatus(tables: DemoTable[]) {
  return tables.reduce(
    (acc, table) => {
      acc[table.status] += 1;
      return acc;
    },
    { available: 0, occupied: 0, attention: 0 },
  );
}

function t(
  label: string,
  capacity: string,
  zone: ZoneId,
  shape: TableShape,
  x: number,
  y: number,
  w: number,
  h: number,
  status: Occupancy,
): DemoTable {
  return { id: label, label, capacity, zone, shape, x, y, w, h, status };
}

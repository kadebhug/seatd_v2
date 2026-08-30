import type {
  DevicesResponse,
  OwnerSnapshotResponse,
} from "@seatd/typescript-seatd-client";
import { seatdFetch } from "../../../lib/seatd-api";
import { DeviceManager } from "./device-manager";

export const dynamic = "force-dynamic";

export default async function DevicesPage() {
  const [devices, snapshot] = await Promise.all([
    seatdFetch<DevicesResponse>("/v1/devices"),
    seatdFetch<OwnerSnapshotResponse>("/v1/owner/snapshot"),
  ]);

  return (
    <main className="page">
      <section className="page-heading">
        <p>Devices</p>
        <h1>Display and operational fleet</h1>
      </section>
      <DeviceManager
        initialDevices={devices.devices}
        locations={snapshot.locations}
      />
    </main>
  );
}

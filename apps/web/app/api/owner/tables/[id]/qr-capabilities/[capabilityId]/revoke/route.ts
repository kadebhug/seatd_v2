import { seatdProxy } from "../../../../../../../../lib/seatd-api";

export async function POST(
  request: Request,
  context: { params: Promise<{ id: string; capabilityId: string }> },
) {
  const { id, capabilityId } = await context.params;
  return seatdProxy(
    `/v1/tables/${id}/qr-capabilities/${capabilityId}/revoke`,
    request,
  );
}

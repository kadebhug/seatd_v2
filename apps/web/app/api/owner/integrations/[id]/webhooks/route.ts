import { seatdProxy } from "../../../../../../lib/seatd-api";

export async function GET(
  request: Request,
  context: { params: Promise<{ id: string }> },
) {
  const { id } = await context.params;
  return seatdProxy(`/v1/integrations/${id}/webhooks`, request);
}

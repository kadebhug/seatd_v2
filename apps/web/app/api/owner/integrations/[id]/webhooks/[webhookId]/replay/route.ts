import { seatdProxy } from "../../../../../../../../lib/seatd-api";

export async function POST(
  request: Request,
  context: { params: Promise<{ id: string; webhookId: string }> },
) {
  const { id, webhookId } = await context.params;
  return seatdProxy(
    `/v1/integrations/${id}/webhooks/${webhookId}/replay`,
    request,
  );
}

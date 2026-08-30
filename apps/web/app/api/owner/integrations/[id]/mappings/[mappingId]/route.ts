import { seatdProxy } from "../../../../../../../lib/seatd-api";

export async function PUT(
  request: Request,
  context: { params: Promise<{ id: string; mappingId: string }> },
) {
  const { id, mappingId } = await context.params;
  return seatdProxy(`/v1/integrations/${id}/mappings/${mappingId}`, request);
}

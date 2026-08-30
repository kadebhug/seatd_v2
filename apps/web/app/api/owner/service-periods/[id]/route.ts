import { seatdProxy } from "../../../../../lib/seatd-api";

export async function PUT(
  request: Request,
  context: { params: Promise<{ id: string }> },
) {
  const { id } = await context.params;
  return seatdProxy(`/v1/service-periods/${id}`, request);
}

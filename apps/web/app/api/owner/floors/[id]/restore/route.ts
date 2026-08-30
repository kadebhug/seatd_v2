import { seatdProxy } from "../../../../../../lib/seatd-api";

export async function POST(
  request: Request,
  context: { params: Promise<{ id: string }> },
) {
  const { id } = await context.params;
  return seatdProxy(`/v1/floors/${id}/restore`, request);
}

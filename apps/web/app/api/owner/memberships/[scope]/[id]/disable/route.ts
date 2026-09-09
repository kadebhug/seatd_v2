import { seatdProxy } from "../../../../../../../lib/seatd-api";

export async function POST(
  request: Request,
  context: { params: Promise<{ scope: string; id: string }> },
) {
  const { scope, id } = await context.params;
  return seatdProxy(`/v1/memberships/${scope}/${id}/disable`, request);
}

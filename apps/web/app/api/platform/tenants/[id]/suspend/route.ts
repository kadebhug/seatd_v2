import { seatdProxy } from "../../../../../../lib/seatd-api";

type Params = {
  id: string;
};

export async function POST(
  request: Request,
  context: { params: Promise<Params> },
) {
  const { id } = await context.params;
  return seatdProxy(`/v1/platform/tenants/${id}/suspend`, request);
}

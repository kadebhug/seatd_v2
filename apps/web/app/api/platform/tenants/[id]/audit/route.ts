import { seatdProxy } from "../../../../../../lib/seatd-api";

type Params = {
  id: string;
};

export async function GET(
  request: Request,
  context: { params: Promise<Params> },
) {
  const { id } = await context.params;
  const url = new URL(request.url);
  return seatdProxy(`/v1/platform/tenants/${id}/audit${url.search}`, request);
}

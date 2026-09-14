import { seatdProxy } from "../../../../../../../../lib/seatd-api";

type Params = {
  id: string;
  membershipId: string;
};

export async function POST(
  request: Request,
  context: { params: Promise<Params> },
) {
  const { id, membershipId } = await context.params;
  return seatdProxy(
    `/v1/platform/tenants/${id}/owners/${membershipId}/reactivate`,
    request,
  );
}

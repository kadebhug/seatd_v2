import { seatdProxy } from "../../../../../lib/seatd-api";

export function POST(request: Request) {
  return seatdProxy("/v1/platform/admins/invitations", request);
}

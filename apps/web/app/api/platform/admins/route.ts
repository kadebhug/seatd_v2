import { seatdProxy } from "../../../../lib/seatd-api";

export function GET(request: Request) {
  return seatdProxy("/v1/platform/admins", request);
}

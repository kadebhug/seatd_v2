import { seatdProxy } from "../../../../lib/seatd-api";

export function GET(request: Request) {
  return seatdProxy("/v1/service-periods", request);
}

export function POST(request: Request) {
  return seatdProxy("/v1/service-periods", request);
}

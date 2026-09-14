import { seatdProxy } from "../../../../lib/seatd-api";

export function GET(request: Request) {
  const url = new URL(request.url);
  return seatdProxy(`/v1/platform/tenants${url.search}`, request);
}

export function POST(request: Request) {
  return seatdProxy("/v1/platform/tenants", request);
}

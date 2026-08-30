import { seatdProxy } from "../../../../../lib/seatd-api";

export function GET(request: Request) {
  const url = new URL(request.url);
  return seatdProxy(`/v1/analytics/data-quality?${url.searchParams}`, request);
}

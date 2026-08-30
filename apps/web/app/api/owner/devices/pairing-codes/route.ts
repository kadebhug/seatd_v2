import { seatdProxy } from "../../../../../lib/seatd-api";

export function POST(request: Request) {
  return seatdProxy("/v1/devices/pairing-codes", request);
}

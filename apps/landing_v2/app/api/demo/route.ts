export async function POST(request: Request) {
  let body: Record<string, unknown>;
  try {
    body = (await request.json()) as Record<string, unknown>;
  } catch {
    return Response.json(
      { message: "Send the form as JSON and try again." },
      { status: 400 },
    );
  }

  const name = String(body.name ?? "").trim();
  const restaurant = String(body.restaurant ?? "").trim();
  const email = String(body.email ?? "").trim();
  const phone = String(body.phone ?? "").trim();
  const tables = String(body.tables ?? "").trim();
  const fields: Record<string, string> = {};

  if (!name) fields.name = "Enter your name.";
  if (!restaurant) fields.restaurant = "Enter the restaurant or group name.";
  if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    fields.email = "Enter a valid email address.";
  }
  if (!tables) fields.tables = "Choose how large the floor is.";

  if (Object.keys(fields).length > 0) {
    return Response.json(
      {
        message: "Check the highlighted fields and try again.",
        fields,
      },
      { status: 400 },
    );
  }

  console.info("seatd.demo.request", {
    name,
    restaurant,
    email,
    phone: phone || undefined,
    tables,
  });

  return Response.json({ ok: true });
}

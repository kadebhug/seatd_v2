import { expect, test, type Page } from "@playwright/test";

const token = "guest-token-with-prefix";
const requestedAt = "2026-09-09T12:00:00Z";

type AssistStatus = "pending" | "acknowledged" | "resolved" | "cancelled";

type Assist = {
  id: string;
  tableId: string;
  tableSessionId: string;
  status: AssistStatus;
  requestedAt: string;
  version: number;
  actionKey: string;
};

function context(activeRequest?: Assist) {
  return {
    locationName: "Main Street",
    tableLabel: "12",
    occupancy: {
      status: "occupied",
      currentSessionId: "session-1",
      version: 2,
      updatedAt: "2026-09-09T11:55:00Z",
    },
    actions: [
      { key: "call_waiter", label: "Call waiter" },
      { key: "request_bill", label: "Request bill" },
      { key: "request_water", label: "Request water" },
    ],
    ...(activeRequest ? { activeRequest } : {}),
  };
}

function assist(status: AssistStatus, version = 1): Assist {
  return {
    id: "assist-1",
    tableId: "table-1",
    tableSessionId: "session-1",
    status,
    requestedAt,
    version,
    actionKey: "call_waiter",
  };
}

async function routeGuestContext(
  page: Page,
  response: unknown,
  status = 200,
) {
  await page.route("**/v1/guest/qr/*", async (route) => {
    if (route.request().method() !== "GET") {
      await route.fallback();
      return;
    }
    await route.fulfill({
      contentType: "application/json",
      status,
      body: JSON.stringify(response),
    });
  });
}

test("valid token can create a guest request", async ({ page }) => {
  await routeGuestContext(page, context());
  await page.route("**/v1/guest/qr/*/requests", async (route) => {
    expect(route.request().method()).toBe("POST");
    const body = route.request().postDataJSON() as { actionKey: string };
    expect(body.actionKey).toBe("call_waiter");
    await route.fulfill({
      contentType: "application/json",
      status: 201,
      body: JSON.stringify(assist("pending")),
    });
  });
  await page.route("**/v1/guest/qr/*/requests/*", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(assist("pending")),
    });
  });

  await page.goto(`/qr/${token}`);
  await expect(page.getByRole("heading", { name: "Main Street" })).toBeVisible();
  await expect(page.getByText("Table")).toBeVisible();
  await expect(page.getByRole("button", { name: "Call waiter" })).toBeVisible();

  await page.getByRole("button", { name: "Call waiter" }).click();

  await expect(page.getByText("Call waiter sent.")).toBeVisible();
  await expect(page.getByText("Staff will be with you shortly.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Cancel request" })).toBeVisible();
});

test("missing token shows incomplete QR link state", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByText("This QR link is incomplete.")).toBeVisible();
  await expect(page.getByText("Ask staff for a new table code.")).toBeVisible();
});

test("revoked or expired token shows unavailable code state", async ({ page }) => {
  await routeGuestContext(
    page,
    { error: { code: "not_found", message: "resource not found" } },
    404,
  );

  await page.goto(`/qr/${token}`);

  await expect(
    page.getByText("This table code is not available right now."),
  ).toBeVisible();
  await expect(
    page.getByText("Ask staff for help, or try scanning again."),
  ).toBeVisible();
});

test("duplicate active request is rendered without creating another request", async ({
  page,
}) => {
  await routeGuestContext(page, context(assist("pending")));

  await page.goto(`/qr/${token}`);

  await expect(page.getByText("Call waiter sent.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Cancel request" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Call waiter" })).toHaveCount(0);
});

test("rate limited request explains the next step", async ({ page }) => {
  await routeGuestContext(page, context());
  await page.route("**/v1/guest/qr/*/requests", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      status: 429,
      body: JSON.stringify({
        error: { code: "rate_limited", message: "too many guest requests" },
      }),
    });
  });

  await page.goto(`/qr/${token}`);
  await page.getByRole("button", { name: "Call waiter" }).click();

  await expect(
    page.getByRole("alert").filter({
      hasText: "Too many guest requests. Try again in a moment.",
    }),
  ).toBeVisible();
});

test("guest can confirm and cancel a pending request", async ({ page }) => {
  await routeGuestContext(page, context(assist("pending")));
  await page.route("**/v1/guest/qr/*/requests/*/cancel", async (route) => {
    expect(route.request().method()).toBe("POST");
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(assist("cancelled", 2)),
    });
  });
  await page.route("**/v1/guest/qr/*/requests/*", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(assist("pending")),
    });
  });

  await page.goto(`/qr/${token}`);
  await page.getByRole("button", { name: "Cancel request" }).click();
  await expect(page.getByRole("button", { name: "Confirm cancel" })).toBeVisible();
  await page.getByRole("button", { name: "Confirm cancel" }).click();

  await expect(page.getByText("Request cancelled.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Call waiter" })).toBeVisible();
});

test("staff resolution is reflected through polling", async ({ page }) => {
  await page.clock.install();
  await routeGuestContext(page, context(assist("pending")));
  let pollCount = 0;
  await page.route("**/v1/guest/qr/*/requests/*", async (route) => {
    pollCount += 1;
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(pollCount === 1 ? assist("pending") : assist("resolved", 2)),
    });
  });

  await page.goto(`/qr/${token}`);
  await expect(page.getByText("Call waiter sent.")).toBeVisible();

  await page.clock.fastForward(3000);
  await expect(page.getByText("Call waiter sent.")).toBeVisible();
  await page.clock.fastForward(3000);

  await expect(page.getByText("This request is complete.")).toBeVisible();
  await expect(
    page.getByText("You can send another if you need anything else."),
  ).toBeVisible();
});

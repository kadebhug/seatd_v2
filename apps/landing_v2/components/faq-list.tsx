const faqs = [
  {
    q: "Does Seatd replace my POS?",
    a: "No. Seatd focuses on live restaurant-floor operations and is designed to work alongside the systems you already use. Your POS still runs orders, bills, and payments.",
  },
  {
    q: "Do guests need to download an app?",
    a: "No. Guests scan the table QR code and request help in the browser. No download, account, email, or password.",
  },
  {
    q: "What happens if the internet connection drops?",
    a: "Staff, owner, and display views need a live connection to stay in sync. If a guest phone goes offline, they can retry once they are back online. Display access uses revocable device tokens.",
  },
  {
    q: "Can we use our existing floor layout?",
    a: "Yes. Owners map floors, zones, and tables to match the physical venue. Table labels, capacities, and shapes can follow how you already talk about the room.",
  },
  {
    q: "Can I have multiple floors or seating areas?",
    a: "Yes. Seatd supports multiple floors and zones, so patio, bar, lounge, and upstairs seating can share one operational picture.",
  },
  {
    q: "Can staff see guest requests?",
    a: "Yes. When a guest sends a request, staff see the table, the action, and that it still needs attention.",
  },
  {
    q: "Can I customise the guest request buttons?",
    a: "Yes. Guest actions are configured per location, so the table QR can offer the requests your service style actually uses.",
  },
  {
    q: "Does this handle reservations?",
    a: "No. Seatd is built for live floor operations rather than booking future tables.",
  },
  {
    q: "Does it process payments?",
    a: "No. Payments stay with your POS.",
  },
];

export function FaqList() {
  return (
    <div className="faq">
      {faqs.map((item) => (
        <details key={item.q}>
          <summary className="focus-ring rounded-[8px]">{item.q}</summary>
          <p>{item.a}</p>
        </details>
      ))}
    </div>
  );
}

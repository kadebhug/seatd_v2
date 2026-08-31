export function BrandLockup({
  className = "h-8",
}: Readonly<{ className?: string }>) {
  return (
    <span className={`relative inline-grid ${className}`}>
      <img
        alt="Seatd"
        className="lockup-light col-start-1 row-start-1"
        height={32}
        src="/brand/seatd-lockup-color.svg"
        style={{ height: 32, width: "auto" }}
        width={104}
      />
      <img
        alt=""
        aria-hidden="true"
        className="lockup-dark col-start-1 row-start-1"
        height={32}
        src="/brand/seatd-lockup-on-ink.svg"
        style={{ height: 32, width: "auto" }}
        width={104}
      />
    </span>
  );
}

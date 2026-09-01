export function EntranceBoard() {
  return (
    <div className="border border-[color-mix(in_srgb,#f4efe6_16%,transparent)] bg-[#1c1410] p-6 text-[#f4efe6] md:p-8">
      <h3 className="font-sans text-2xl font-semibold tracking-tight">
        Seating available now
      </h3>
      <ul className="mt-6 grid gap-3">
        <BoardRow area="Main floor" value="8 tables" />
        <BoardRow area="Patio" value="3 tables" />
        <BoardRow area="Bar" value="2 high-tops" />
        <BoardRow area="Lounge" value="Wait 10 min" muted />
      </ul>
    </div>
  );
}

function BoardRow({
  area,
  value,
  muted = false,
}: Readonly<{ area: string; value: string; muted?: boolean }>) {
  return (
    <li className="flex items-baseline justify-between gap-4 border-b border-[color-mix(in_srgb,#f4efe6_14%,transparent)] py-3 last:border-b-0">
      <span>{area}</span>
      <strong
        className={`font-mono text-lg tabular-nums ${muted ? "text-[#d99a2b]" : "text-[#2f7d65]"}`}
      >
        {value}
      </strong>
    </li>
  );
}

/**
 * Metal clipboard clip for the signature host-chart composition.
 *
 * Rendered as vector rather than raster (see DESIGN.md "The Material Truth
 * Rule"): the shipped `seatd-clipboard-clip.webp` is an opaque 200x80 crop with
 * label text and mock background baked in, so it cannot layer over the board.
 * Board and sheet still use their rasters.
 *
 * The chrome ramp is deliberately non-monotonic — light, dark, light again,
 * dark. A plain light-to-dark gradient is what makes CSS clips read as plastic;
 * the mid-ramp bounce at 34% is the reflected-highlight band that sells metal.
 */
export function ClipboardClip() {
  return (
    <svg
      aria-hidden="true"
      className="clipboard-clip-svg"
      preserveAspectRatio="xMidYMin meet"
      viewBox="0 0 240 96"
    >
      <defs>
        <linearGradient id="seatd-clip-plate" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stopColor="#f4f2ef" />
          <stop offset="10%" stopColor="#d6d2cc" />
          <stop offset="22%" stopColor="#96918a" />
          <stop offset="34%" stopColor="#ebe8e3" />
          <stop offset="48%" stopColor="#b9b4ac" />
          <stop offset="66%" stopColor="#74706a" />
          <stop offset="82%" stopColor="#a8a29a" />
          <stop offset="100%" stopColor="#56524d" />
        </linearGradient>

        <linearGradient id="seatd-clip-dome" x1="0.18" x2="0.82" y1="0" y2="1">
          <stop offset="0%" stopColor="#fbfaf8" />
          <stop offset="28%" stopColor="#d2cec8" />
          <stop offset="58%" stopColor="#8d8880" />
          <stop offset="100%" stopColor="#5f5b55" />
        </linearGradient>

        <linearGradient id="seatd-clip-sheen" x1="0" x2="1" y1="0" y2="0">
          <stop offset="0%" stopColor="#ffffff" stopOpacity="0" />
          <stop offset="22%" stopColor="#ffffff" stopOpacity="0.5" />
          <stop offset="46%" stopColor="#ffffff" stopOpacity="0.72" />
          <stop offset="74%" stopColor="#ffffff" stopOpacity="0.28" />
          <stop offset="100%" stopColor="#ffffff" stopOpacity="0" />
        </linearGradient>

        <radialGradient
          cx="0.36"
          cy="0.3"
          id="seatd-clip-rivet"
          r="0.78"
        >
          <stop offset="0%" stopColor="#ffffff" />
          <stop offset="34%" stopColor="#cbc6bf" />
          <stop offset="72%" stopColor="#7d7871" />
          <stop offset="100%" stopColor="#48443f" />
        </radialGradient>

        <linearGradient id="seatd-clip-lip" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stopColor="#3d3a36" />
          <stop offset="100%" stopColor="#22201d" />
        </linearGradient>
      </defs>

      {/* Spring housing rising behind the plate */}
      <path
        d="M84 34 L84 22 Q84 8 100 8 L140 8 Q156 8 156 22 L156 34 Z"
        fill="url(#seatd-clip-dome)"
        stroke="rgb(0 0 0 / 0.4)"
        strokeWidth="1"
      />
      <path
        d="M91 32 L91 23 Q91 14 101 14 L114 14"
        fill="none"
        stroke="rgb(255 255 255 / 0.55)"
        strokeLinecap="round"
        strokeWidth="1.6"
      />

      {/* Grip lip tucked under the plate, biting the sheet */}
      <rect
        fill="url(#seatd-clip-lip)"
        height="14"
        rx="3"
        width="176"
        x="32"
        y="62"
      />

      {/* Hardware plate */}
      <rect
        fill="url(#seatd-clip-plate)"
        height="44"
        rx="7"
        stroke="rgb(0 0 0 / 0.42)"
        strokeWidth="1"
        width="200"
        x="20"
        y="28"
      />

      {/* Reflected-highlight band across the upper plate */}
      <rect
        fill="url(#seatd-clip-sheen)"
        height="5"
        rx="2.5"
        width="184"
        x="28"
        y="38"
      />

      {/* Top edge catch-light and bottom shade, keyed upper-left like the field */}
      <path
        d="M27 30.5 H213"
        stroke="rgb(255 255 255 / 0.62)"
        strokeLinecap="round"
        strokeWidth="1.4"
      />
      <path
        d="M30 69.5 H210"
        stroke="rgb(0 0 0 / 0.34)"
        strokeLinecap="round"
        strokeWidth="1.6"
      />

      {/* Pivot pins */}
      <circle cx="38" cy="50" fill="url(#seatd-clip-rivet)" r="5.5" />
      <circle
        cx="38"
        cy="50"
        fill="none"
        r="5.5"
        stroke="rgb(0 0 0 / 0.34)"
        strokeWidth="0.9"
      />
      <circle cx="202" cy="50" fill="url(#seatd-clip-rivet)" r="5.5" />
      <circle
        cx="202"
        cy="50"
        fill="none"
        r="5.5"
        stroke="rgb(0 0 0 / 0.34)"
        strokeWidth="0.9"
      />
    </svg>
  );
}

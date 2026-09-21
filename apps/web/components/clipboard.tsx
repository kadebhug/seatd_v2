import type { ReactNode } from "react";
import { ClipboardClip } from "./clipboard-clip";

type ClipboardProps = Readonly<{
  children: ReactNode;
  className?: string;
  sheetClassName?: string;
  stamp?: string;
}>;

export function Clipboard({
  children,
  className = "",
  sheetClassName = "",
  stamp,
}: ClipboardProps) {
  return (
    <div className={`clipboard ${className}`.trim()}>
      <div aria-hidden="true" className="clipboard-clip">
        <ClipboardClip />
      </div>
      {stamp ? <p className="clipboard-stamp">{stamp}</p> : null}
      <div className={`chart-sheet ${sheetClassName}`.trim()}>{children}</div>
    </div>
  );
}

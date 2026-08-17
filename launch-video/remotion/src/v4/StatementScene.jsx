import React from "react";
import {AbsoluteFill, useCurrentFrame} from "remotion";
import {colors, fade} from "./theme.js";

export function StatementScene({children, index}) {
  const frame = useCurrentFrame();

  return (
    <AbsoluteFill
      style={{
        display: "grid",
        placeItems: "center",
        background: colors.white,
        color: colors.ink,
      }}
    >
      <div
        style={{
          fontSize: 112,
          fontWeight: 700,
          letterSpacing: -7,
          opacity: fade(frame, 1, 7),
          translate: `0 ${fade(frame, 1, 9, 18, 0)}px`,
        }}
      >
        {children}<span style={{color: "#77a81a"}}>.</span>
      </div>
      <div style={{position: "absolute", right: 64, bottom: 48, color: colors.muted, fontSize: 13, fontWeight: 700}}>
        {index}
      </div>
    </AbsoluteFill>
  );
}


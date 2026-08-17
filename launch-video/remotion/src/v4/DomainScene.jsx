import React from "react";
import {AbsoluteFill, useCurrentFrame} from "remotion";
import {colors, fade} from "./theme.js";

export function DomainScene() {
  const frame = useCurrentFrame();

  const reveal = (start) => ({
    opacity: fade(frame, start, start + 6),
    translate: `0 ${fade(frame, start, start + 8, 16, 0)}px`,
  });

  return (
    <AbsoluteFill style={{display: "grid", placeItems: "center", background: colors.white, color: colors.ink}}>
      <div style={{display: "flex", flexDirection: "column", alignItems: "center", textAlign: "center"}}>
        <div
          style={{
            ...reveal(1),
            fontSize: 16,
            fontWeight: 700,
            letterSpacing: 5.5,
            color: colors.muted,
          }}
        >
          YOUR LEARNING HOME
        </div>
        <div style={{...reveal(6), marginTop: 32, fontSize: 96, fontWeight: 700, letterSpacing: -4, lineHeight: 1}}>
          <span style={{color: "#77a81a"}}>maya</span>
          <span>.learnloom.blog</span>
        </div>
        <div
          style={{
            ...reveal(12),
            marginTop: 34,
            fontSize: 22,
            fontWeight: 600,
            letterSpacing: -0.3,
            color: colors.muted,
          }}
        >
          Private by default. Published when you decide.
        </div>
      </div>
    </AbsoluteFill>
  );
}

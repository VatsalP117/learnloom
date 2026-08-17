import React from "react";
import {AbsoluteFill, Img, staticFile, useCurrentFrame} from "remotion";
import {colors, fade} from "./theme.js";

export function CloseScene() {
  const frame = useCurrentFrame();

  return (
    <AbsoluteFill style={{display: "grid", placeItems: "center", background: colors.white, color: colors.ink}}>
      <div style={{display: "flex", flexDirection: "column", alignItems: "center", opacity: fade(frame, 3, 11)}}>
        <div style={{display: "flex", alignItems: "center", gap: 12}}>
          <Img src={staticFile("favicon.svg")} style={{width: 38, height: 38, borderRadius: 11}} />
          <span style={{fontSize: 31, fontWeight: 700, letterSpacing: -1.3}}>Learnloom</span>
        </div>
        <div style={{marginTop: 34, textAlign: "center", fontSize: 94, fontWeight: 700, lineHeight: 0.98, letterSpacing: -6}}>
          STAY CURRENT.<br />BUILD UNDERSTANDING.
        </div>
        <div style={{marginTop: 34, color: colors.muted, fontSize: 18, fontWeight: 600}}>learnloom.blog</div>
      </div>
    </AbsoluteFill>
  );
}


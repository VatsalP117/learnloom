import React from "react";
import {AbsoluteFill, Img, staticFile, useCurrentFrame} from "remotion";
import {colors, enter} from "./theme.js";

export function CloseScene() {
  const frame = useCurrentFrame();
  const brand = enter(frame, 45, 12);

  return (
    <AbsoluteFill style={{overflow: "hidden", background: "#ffffff", color: colors.ink}}>
      <div
        style={{
          position: "absolute",
          zIndex: 30,
          inset: 0,
          display: "grid",
          placeItems: "center",
          opacity: brand,
        }}
      >
        <div style={{display: "flex", flexDirection: "column", alignItems: "center", textAlign: "center"}}>
          <div style={{display: "flex", alignItems: "center", gap: 14}}>
            <Img src={staticFile("favicon.svg")} style={{width: 46, height: 46, borderRadius: 13}} />
            <span style={{fontSize: 36, fontWeight: 700, letterSpacing: -1.5}}>Learnloom</span>
          </div>
          <div style={{maxWidth: 1450, marginTop: 42, fontSize: 104, fontWeight: 700, lineHeight: 0.94, letterSpacing: -6.5}}>
            STAY CURRENT.<br />BUILD UNDERSTANDING.
          </div>
          <div style={{marginTop: 44, padding: "17px 24px", borderRadius: 999, color: colors.ink, background: colors.lime, fontSize: 17, fontWeight: 800}}>
            Start a learning stream · learnloom.blog →
          </div>
        </div>
      </div>
    </AbsoluteFill>
  );
}

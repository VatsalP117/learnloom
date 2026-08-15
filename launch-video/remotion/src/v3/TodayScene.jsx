import React from "react";
import {AbsoluteFill, useCurrentFrame} from "remotion";
import {ProductFrame} from "./ProductFrame.jsx";
import {colors, enter, eyebrow, grain, hero, sceneBase} from "./theme.js";

export function TodayScene() {
  const frame = useCurrentFrame();
  const label = enter(frame, 25, 12);

  return (
    <AbsoluteFill style={{...sceneBase, background: "linear-gradient(135deg,#f6f3ff 0%,#fbfbfa 58%,#eff8f2 100%)"}}>
      <div style={grain} />
      <ProductFrame clip="today.webm" playbackRate={1.16} style={{...hero}} />
      <div
        style={{
          position: "absolute",
          zIndex: 30,
          left: 132,
          bottom: 64,
          minWidth: 600,
          padding: "25px 30px 27px",
          borderRadius: 20,
          color: colors.white,
          background: colors.ink,
          opacity: label,
        }}
      >
        <div style={{...eyebrow, color: colors.lime, fontSize: 12}}>Today, decided</div>
        <div style={{marginTop: 8, fontSize: 43, fontWeight: 650, letterSpacing: -2}}>One worthwhile step. Ready when you are.</div>
      </div>
    </AbsoluteFill>
  );
}

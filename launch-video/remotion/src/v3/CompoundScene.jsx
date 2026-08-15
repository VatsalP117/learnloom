import React from "react";
import {AbsoluteFill, useCurrentFrame} from "remotion";
import {ProductFrame} from "./ProductFrame.jsx";
import {colors, enter, eyebrow, grain, sceneBase, stage} from "./theme.js";

export function CompoundScene() {
  const frame = useCurrentFrame();
  const publishingIn = enter(frame, 72, 13);

  return (
    <AbsoluteFill style={{...sceneBase, background: "linear-gradient(145deg,#f7f3ec,#f7f7f5 52%,#eff5f1)"}}>
      <div style={grain} />
      <div style={{position: "absolute", zIndex: 40, left: 94, top: 70, width: 500}}>
        <div style={eyebrow}>A learning history that compounds</div>
        <div style={{marginTop: 15, fontSize: 52, fontWeight: 700, lineHeight: 1.02, letterSpacing: -2.5}}>
          KEEP THE LESSON.<br />FOLLOW THE THREAD.
        </div>
      </div>
      <ProductFrame
        clip="library.webm"
        playbackRate={1.15}
        start={8}
        style={{...stage}}
      />
      <ProductFrame
        clip="publishing.webm"
        playbackRate={1.1}
        style={{...stage, opacity: publishingIn}}
      />
      <div
        style={{
          position: "absolute",
          zIndex: 50,
          right: 112,
          bottom: 60,
          padding: "17px 21px",
          borderRadius: 14,
          color: colors.white,
          background: colors.ink,
          fontSize: 17,
          fontWeight: 700,
          opacity: enter(frame, 112, 10),
        }}
      >
        Private by default. Published when you decide. ↗
      </div>
    </AbsoluteFill>
  );
}

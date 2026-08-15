import React from "react";
import {AbsoluteFill, useCurrentFrame} from "remotion";
import {ProductFrame} from "./ProductFrame.jsx";
import {colors, enter, eyebrow, grain, sceneBase, stage} from "./theme.js";

export function LessonScene() {
  const frame = useCurrentFrame();
  const statement = enter(frame, 4, 11);
  const support = enter(frame, 36, 12);

  return (
    <AbsoluteFill style={{...sceneBase, background: colors.ink}}>
      <div style={{...grain, opacity: 0.12}} />
      <div
        style={{
          position: "absolute",
          zIndex: 30,
          top: 84,
          left: 94,
          width: 470,
          color: colors.white,
          opacity: statement,
        }}
      >
        <div style={{...eyebrow, color: colors.lime}}>Built for understanding</div>
        <div style={{marginTop: 24, fontSize: 96, fontWeight: 700, lineHeight: 0.88, letterSpacing: -6}}>
          NOT A<br />SUMMARY.
        </div>
        <div style={{marginTop: 35, color: "#b8bbb5", fontSize: 25, fontWeight: 500, lineHeight: 1.35, opacity: support}}>
          Evidence, mechanism, a worked example—and the question that makes it stick.
        </div>
      </div>
      <ProductFrame
        clip="lesson.webm"
        playbackRate={1.24}
        start={12}
        style={{...stage}}
      />
      <div
        style={{
          position: "absolute",
          zIndex: 40,
          right: 116,
          bottom: 68,
          padding: "14px 18px",
          borderRadius: 999,
          color: colors.ink,
          background: colors.lime,
          fontSize: 13,
          fontWeight: 800,
          letterSpacing: 1.2,
          opacity: enter(frame, 92, 10),
        }}
      >
        SOURCE-GROUNDED · 12 MIN
      </div>
    </AbsoluteFill>
  );
}

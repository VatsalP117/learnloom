import React from "react";
import {AbsoluteFill, interpolate, useCurrentFrame} from "remotion";
import {colors, enter, eyebrow, grain, sceneBase} from "./theme.js";

export function HookScene() {
  const frame = useCurrentFrame();
  const lines = ["STAYING CURRENT", "SHOULDN’T MEAN", "STARTING OVER."];

  return (
    <AbsoluteFill style={{...sceneBase, padding: "90px 96px"}}>
      <div style={grain} />
      <div style={{...eyebrow, display: "flex", alignItems: "center", gap: 12}}>
        <span style={{width: 10, height: 10, borderRadius: "50%", background: colors.lime}} />
        Learnloom · A learning practice for fast-moving work
      </div>
      <div style={{position: "absolute", left: 96, right: 96, bottom: 98}}>
        {lines.map((line, index) => {
          const reveal = enter(frame, 5 + index * 11, 9);
          return (
            <div
              key={line}
              style={{
                overflow: "hidden",
                color: index === 2 ? colors.ink : "#b1b3ae",
                fontSize: 116,
                fontWeight: 700,
                lineHeight: 0.92,
                letterSpacing: -7,
              }}
            >
              <div
                style={{
                  opacity: reveal,
                  translate: `${interpolate(reveal, [0, 1], [-54, 0])}px 0`,
                }}
              >
                {line}
              </div>
            </div>
          );
        })}
      </div>
      <div style={{position: "absolute", right: 96, top: 88, color: colors.muted, fontSize: 13, fontWeight: 700}}>01 / 06</div>
    </AbsoluteFill>
  );
}

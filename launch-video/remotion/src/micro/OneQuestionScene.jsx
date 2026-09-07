import React from "react";
import {AbsoluteFill, interpolate, spring, useCurrentFrame} from "remotion";
import {bricolage, colors, ease, fade} from "./theme.js";

const QUESTION = "How do AI systems learn — and fail?";
const TYPE_START = 6;
const TYPE_END = 66;

export function OneQuestionScene() {
  const frame = useCurrentFrame();

  const pillIn = spring({frame, fps: 30, config: {damping: 15, stiffness: 170, mass: 0.8}});

  // One character roughly every two frames: snappy, but still a felt typing beat.
  const chars = Math.floor(
    interpolate(frame, [TYPE_START, TYPE_END], [0, QUESTION.length], {
      extrapolateLeft: "clamp",
      extrapolateRight: "clamp",
      easing: ease,
    }),
  );
  const typed = QUESTION.slice(0, chars);

  const blinking = frame % 8 < 4 ? 1 : 0.3;
  const caretOpacity =
    frame < TYPE_START
      ? 0
      : frame >= TYPE_END
        ? fade(frame, TYPE_END, TYPE_END + 6, 1, 0)
        : blinking;

  return (
    <AbsoluteFill style={{display: "grid", placeItems: "center"}}>
      <div
        style={{
          opacity: fade(frame, 0, 8) * fade(frame, 71, 78, 1, 0),
          transform: `scale(${pillIn})`,
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 18,
            padding: "26px 44px",
            borderRadius: 999,
            background: colors.white,
            border: `1px solid ${colors.line}`,
            boxShadow: "0 16px 44px rgba(16, 17, 15, 0.08)",
          }}
        >
          <span
            style={{
              width: 10,
              height: 10,
              borderRadius: 5,
              background: colors.lime,
              border: `2px solid ${colors.green}`,
              flexShrink: 0,
            }}
          />
          <span
            style={{
              fontFamily: bricolage,
              fontSize: 68,
              fontWeight: 600,
              letterSpacing: -2.8,
              color: colors.ink,
              whiteSpace: "nowrap",
            }}
          >
            {typed}
            <span style={{opacity: caretOpacity, color: colors.green, fontWeight: 800}}>|</span>
          </span>
        </div>
      </div>
    </AbsoluteFill>
  );
}

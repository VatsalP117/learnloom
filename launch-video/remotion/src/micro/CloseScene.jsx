import React from "react";
import {AbsoluteFill, Img, interpolateColors, spring, staticFile, useCurrentFrame} from "remotion";
import {bricolage, colors, fade, manrope, rise} from "./theme.js";

/*
 * Brand close: a clean white card on warm paper, the favicon lockup,
 * the promise line, and the domain.
 */

const reveal = (frame, start, distance = 14) => ({
  opacity: fade(frame, start, start + 7),
  translate: `0 ${rise(frame, start, start + 9, distance)}px`,
});

export function CloseScene() {
  const frame = useCurrentFrame();

  const paper = interpolateColors(frame, [0, 18], [colors.white, colors.paper]);
  const cardIn = spring({frame, fps: 30, config: {damping: 15, stiffness: 170, mass: 0.9}});

  return (
    <AbsoluteFill style={{background: paper, display: "grid", placeItems: "center"}}>
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          width: 820,
          padding: "58px 64px",
          borderRadius: 28,
          background: colors.white,
          border: `1px solid ${colors.hairline}`,
          boxShadow: "0 26px 70px rgba(16, 17, 15, 0.12)",
          opacity: fade(frame, 2, 10),
          transform: `scale(${cardIn})`,
        }}
      >
        <div style={{display: "flex", alignItems: "center", gap: 14, ...reveal(frame, 4)}}>
          <Img src={staticFile("favicon.svg")} style={{width: 46, height: 46, borderRadius: 14}} />
          <span style={{fontFamily: bricolage, fontSize: 40, fontWeight: 800, letterSpacing: -1.8, color: colors.ink}}>
            Learnloom
          </span>
        </div>

        <div style={{marginTop: 30, width: 120, height: 1, background: colors.line}} />

        <div
          style={{
            marginTop: 30,
            fontFamily: bricolage,
            fontSize: 27,
            fontWeight: 600,
            letterSpacing: -1,
            color: colors.ink,
            textAlign: "center",
            ...reveal(frame, 11),
          }}
        >
          Turn curiosity into a practice<span style={{color: colors.green}}>.</span>
        </div>

        <div
          style={{
            marginTop: 28,
            display: "flex",
            alignItems: "center",
            gap: 9,
            fontFamily: manrope,
            fontSize: 15.5,
            fontWeight: 700,
            letterSpacing: 0.3,
            color: colors.muted,
            ...reveal(frame, 17, 10),
          }}
        >
          <span style={{width: 7, height: 7, borderRadius: 4, background: colors.lime, border: `1.5px solid ${colors.green}`}} />
          learnloom.blog
        </div>
      </div>
    </AbsoluteFill>
  );
}
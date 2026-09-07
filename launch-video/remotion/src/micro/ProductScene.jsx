import React from "react";
import {Video} from "@remotion/media";
import {AbsoluteFill, interpolate, spring, staticFile, useCurrentFrame} from "remotion";
import {colors, ease, fade, manrope} from "./theme.js";

/*
 * The Knowledge Dossier surface (520×300 at the canvas center) morphs into a
 * rounded product window holding `lesson.webm`. Three short words float as
 * translucent chips near the window corners; a slow push keeps the camera
 * alive without hiding the core UI.
 */

// Matching the dossier geometry from SourcesScene.
const ARTIFACT = {width: 520, height: 300};
const WINDOW = {width: 1340, height: 754};
const CENTER = {left: 290, top: 153}; // (960, 530) centered on the canvas.

const CHIPS = [
  {word: "Sourced.", at: 14, left: 300, top: 168},
  {word: "Structured.", at: 19, left: 1560, top: 452},
  {word: "Remembered.", at: 24, left: 316, top: 854},
];

function Chip({word, at, left, top}) {
  const frame = useCurrentFrame();
  const pop = spring({frame: frame - at, fps: 30, config: {damping: 14, stiffness: 190, mass: 0.8}});
  return (
    <div
      style={{
        position: "absolute",
        left,
        top,
        display: "flex",
        alignItems: "center",
        gap: 9,
        padding: "11px 19px",
        borderRadius: 999,
        background: "rgba(247, 247, 244, 0.86)",
        backdropFilter: "blur(10px)",
        WebkitBackdropFilter: "blur(10px)",
        border: `1px solid ${colors.hairline}`,
        boxShadow: "0 10px 28px rgba(16, 17, 15, 0.08)",
        fontFamily: manrope,
        fontSize: 17,
        fontWeight: 600,
        letterSpacing: -0.2,
        color: colors.ink,
        opacity: fade(frame, at, at + 6),
        transform: `scale(${pop}) translateY(${fade(frame, at, at + 8, 10, 0)}px)`,
      }}
    >
      <span style={{width: 7, height: 7, borderRadius: 4, background: colors.lime, border: `1.5px solid ${colors.green}`, flexShrink: 0}} />
      {word}
    </div>
  );
}

export function ProductScene() {
  const frame = useCurrentFrame();

  // The dossier surface expands to window size on the shared ease curve.
  const morph = interpolate(frame, [0, 24], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: ease,
  });
  const left = interpolate(morph, [0, 1], [700, CENTER.left]);
  const top = interpolate(morph, [0, 1], [380, CENTER.top]);
  const width = interpolate(morph, [0, 1], [ARTIFACT.width, WINDOW.width]);
  const height = interpolate(morph, [0, 1], [ARTIFACT.height, WINDOW.height]);
  const radius = 22 + morph * 4;

  // Camera push after the morph settles.
  const push = interpolate(frame, [26, 112], [1, 1.045], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: ease,
  });

  return (
    <AbsoluteFill>
      <div
        style={{
          position: "absolute",
          width: 1920,
          height: 1080,
          transform: `scale(${push})`,
        }}
      >
        <div
          style={{
            position: "absolute",
            left,
            top,
            width,
            height,
            borderRadius: radius,
            overflow: "hidden",
            background: colors.white,
            border: `1px solid ${colors.line}`,
            boxShadow: `0 34px 90px rgba(16, 17, 15, ${0.16 * fade(frame, 8, 30)})`,
          }}
        >
          <Video
            src={staticFile("product-clips/lesson.webm")}
            trimBefore={12}
            playbackRate={0.4}
            muted
            pauseWhenBuffering
            objectFit="cover"
            style={{
              position: "absolute",
              inset: 0,
              width: "100%",
              height: "100%",
              opacity: fade(frame, 8, 18),
            }}
          />
        </div>

        {CHIPS.map((chip) => (
          <Chip key={chip.word} {...chip} />
        ))}
      </div>
    </AbsoluteFill>
  );
}

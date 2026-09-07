import React from "react";
import {Video} from "@remotion/media";
import {AbsoluteFill, interpolate, spring, staticFile, useCurrentFrame} from "remotion";
import {bricolage, colors, ease, fade, manrope} from "./theme.js";

/*
 * Continuity, not a second demo: the same window keeps pushing while the
 * lesson recording crossfades into the library recording. One chip survives,
 * and a single headline lands below the window.
 */

const WINDOW = {left: 290, top: 153, width: 1340, height: 754};

export function HomeScene() {
  const frame = useCurrentFrame();

  // The camera push from ProductScene continues, then the window recedes.
  const push = interpolate(frame, [0, 24], [1.045, 1.06], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: ease,
  });
  const exit = fade(frame, 31, 41, 1, 0);

  const remembered = spring({frame: frame - 2, fps: 30, config: {damping: 14, stiffness: 190, mass: 0.8}});

  const headlineOut = fade(frame, 6, 14) * fade(frame, 31, 41, 1, 0);

  return (
    <AbsoluteFill>
      <div style={{position: "absolute", width: 1920, height: 1080, transform: `scale(${push})`, opacity: exit}}>
        <div
          style={{
            position: "absolute",
            ...WINDOW,
            borderRadius: 26,
            overflow: "hidden",
            background: colors.white,
            border: `1px solid ${colors.line}`,
            boxShadow: "0 34px 90px rgba(16, 17, 15, 0.16)",
          }}
        >
          <Video
            src={staticFile("product-clips/lesson.webm")}
            trimBefore={55}
            playbackRate={0.4}
            muted
            pauseWhenBuffering
            objectFit="cover"
            style={{
              position: "absolute",
              inset: 0,
              width: "100%",
              height: "100%",
              opacity: fade(frame, 2, 10, 1, 0),
            }}
          />
          <Video
            src={staticFile("product-clips/library.webm")}
            trimBefore={8}
            muted
            pauseWhenBuffering
            objectFit="cover"
            style={{
              position: "absolute",
              inset: 0,
              width: "100%",
              height: "100%",
              opacity: fade(frame, 4, 12),
            }}
          />
        </div>

        {/* Only "Remembered." survives into the learning home. */}
        <div
          style={{
            position: "absolute",
            left: 316,
            top: 854,
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
            opacity: fade(frame, 2, 8),
            transform: `scale(${remembered})`,
          }}
        >
          <span style={{width: 7, height: 7, borderRadius: 4, background: colors.lime, border: `1.5px solid ${colors.green}`, flexShrink: 0}} />
          Remembered.
        </div>
      </div>

      <div
        style={{
          position: "absolute",
          left: 960,
          top: 948,
          transform: "translateX(-50%)",
          textAlign: "center",
          whiteSpace: "nowrap",
          fontFamily: bricolage,
          fontSize: 36,
          fontWeight: 700,
          letterSpacing: -1.4,
          color: colors.ink,
          opacity: headlineOut,
          translate: `0 ${fade(frame, 6, 14, 12, 0)}px`,
        }}
      >
        Every lesson has somewhere to live<span style={{color: colors.green}}>.</span>
      </div>
    </AbsoluteFill>
  );
}

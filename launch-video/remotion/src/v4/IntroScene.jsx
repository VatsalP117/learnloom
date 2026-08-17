import React from "react";
import {AbsoluteFill, Img, staticFile, useCurrentFrame} from "remotion";
import {colors, hold} from "./theme.js";

const centered = {
  position: "absolute",
  inset: 0,
  display: "grid",
  placeItems: "center",
  textAlign: "center",
};

export function IntroScene() {
  const frame = useCurrentFrame();

  return (
    <AbsoluteFill style={{background: colors.white, color: colors.ink}}>
      <div style={{...centered, opacity: hold(frame, 0, 8, 25, 33)}}>
        <Img src={staticFile("favicon.svg")} style={{width: 74, height: 74, borderRadius: 20}} />
      </div>
      <div style={{...centered, opacity: hold(frame, 30, 38, 58, 66), fontSize: 50, fontWeight: 650, letterSpacing: -2.3}}>
        Introducing
      </div>
      <div style={{...centered, opacity: hold(frame, 63, 71, 99, 107), fontSize: 50, fontWeight: 650, letterSpacing: -2.3}}>
        <span>
          Introducing Learnloom<span style={{color: "#77a81a"}}>.</span>
        </span>
      </div>
      <div style={{...centered, opacity: hold(frame, 104, 112, 142, 150)}}>
        <div style={{maxWidth: 1150, fontSize: 68, fontWeight: 650, lineHeight: 1.04, letterSpacing: -3.8}}>
          A learning practice for fast-moving work.
        </div>
      </div>
    </AbsoluteFill>
  );
}


import React from "react";
import {Audio} from "@remotion/media";
import {AbsoluteFill, Sequence, staticFile} from "remotion";
import {CloseScene} from "./CloseScene.jsx";
import {HomeScene} from "./HomeScene.jsx";
import {OneQuestionScene} from "./OneQuestionScene.jsx";
import {ProductScene} from "./ProductScene.jsx";
import {SourcesScene} from "./SourcesScene.jsx";
import {colors, manrope} from "./theme.js";

export const MICRO_DURATION = 360;

// Subtle blurred sage/lime light shared by every scene.
const light = {position: "absolute", borderRadius: "50%", filter: "blur(70px)"};

export function LearnloomMicroLaunch({sound = true}) {
  return (
    <AbsoluteFill style={{fontFamily: manrope, background: colors.white}}>
      <div style={{...light, left: 120, top: 80, width: 560, height: 560, background: "rgba(31, 69, 51, 0.13)"}} />
      <div style={{...light, left: 1330, top: 460, width: 620, height: 620, background: "rgba(217, 255, 114, 0.3)"}} />

      {sound ? (
        <>
          <Audio
            src={staticFile("launch-music-v1-1.m4a")}
            volume={(f) => (f < 318 ? 0.4 : Math.max(0, (0.4 * (360 - f)) / 42))}
          />
          {/* Restrained physical cues only: one key at typing, one at the
              window morph, one at the brand lockup. */}
          <Sequence from={6}>
            <Audio src={staticFile("sfx-source/key-soft-02.wav")} volume={0.2} />
          </Sequence>
          <Sequence from={158}>
            <Audio src={staticFile("sfx-source/key-soft-01.wav")} volume={0.16} />
          </Sequence>
          <Sequence from={316}>
            <Audio src={staticFile("sfx-source/key-soft-03.wav")} volume={0.14} />
          </Sequence>
        </>
      ) : null}

      <Sequence from={0} durationInFrames={78} name="One question">
        <OneQuestionScene />
      </Sequence>
      <Sequence from={78} durationInFrames={76} name="Sources assemble">
        <SourcesScene />
      </Sequence>
      <Sequence from={154} durationInFrames={116} name="Product payoff">
        <ProductScene />
      </Sequence>
      <Sequence from={270} durationInFrames={42} name="Learning home">
        <HomeScene />
      </Sequence>
      <Sequence from={312} durationInFrames={48} name="Brand close">
        <CloseScene />
      </Sequence>
    </AbsoluteFill>
  );
}
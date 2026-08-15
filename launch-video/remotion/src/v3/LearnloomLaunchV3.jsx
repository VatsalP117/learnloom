import React from "react";
import {Audio} from "@remotion/media";
import {loadFont as loadBricolage} from "@remotion/google-fonts/BricolageGrotesque";
import {loadFont as loadManrope} from "@remotion/google-fonts/Manrope";
import {TransitionSeries, linearTiming} from "@remotion/transitions";
import {fade} from "@remotion/transitions/fade";
import {AbsoluteFill, Sequence, staticFile} from "remotion";
import {CloseScene} from "./CloseScene.jsx";
import {CompoundScene} from "./CompoundScene.jsx";
import {HookScene} from "./HookScene.jsx";
import {LessonScene} from "./LessonScene.jsx";
import {RecallScene} from "./RecallScene.jsx";
import {TodayScene} from "./TodayScene.jsx";

const {fontFamily: bodyFont} = loadManrope("normal", {
  weights: ["400", "500", "600", "700"],
  subsets: ["latin"],
});

const {fontFamily: displayFont} = loadBricolage("normal", {
  weights: ["500", "600", "700"],
  subsets: ["latin"],
});

const sceneTransition = linearTiming({durationInFrames: 6});

export const V3_DURATION = 805;

export function LearnloomLaunchV3({sound = true}) {
  return (
    <AbsoluteFill style={{fontFamily: displayFont, background: "#111210"}}>
      {sound ? (
        <>
          <Audio
            src={staticFile("launch-music-v1-1.m4a")}
            volume={(frame) => (frame < 750 ? 0.55 : Math.max(0, 0.55 * (805 - frame) / 55))}
          />
          {[64, 214, 359, 494, 644].map((from, index) => (
            <Sequence key={from} from={from} layout="none">
              <Audio
                src={staticFile(index % 2 ? "sfx-source/key-soft-02.wav" : "sfx-source/human-soft-02.wav")}
                volume={index % 2 ? 0.3 : 0.42}
              />
            </Sequence>
          ))}
        </>
      ) : null}

      <div style={{fontFamily: bodyFont}}>
        <TransitionSeries>
          <TransitionSeries.Sequence durationInFrames={71} name="Hook">
            <HookScene />
          </TransitionSeries.Sequence>
          <TransitionSeries.Transition presentation={fade()} timing={sceneTransition} />
          <TransitionSeries.Sequence durationInFrames={156} name="Today">
            <TodayScene />
          </TransitionSeries.Sequence>
          <TransitionSeries.Transition presentation={fade()} timing={sceneTransition} />
          <TransitionSeries.Sequence durationInFrames={151} name="Lesson">
            <LessonScene />
          </TransitionSeries.Sequence>
          <TransitionSeries.Transition presentation={fade()} timing={sceneTransition} />
          <TransitionSeries.Sequence durationInFrames={141} name="Recall">
            <RecallScene />
          </TransitionSeries.Sequence>
          <TransitionSeries.Transition presentation={fade()} timing={sceneTransition} />
          <TransitionSeries.Sequence durationInFrames={156} name="Compound">
            <CompoundScene />
          </TransitionSeries.Sequence>
          <TransitionSeries.Transition presentation={fade()} timing={sceneTransition} />
          <TransitionSeries.Sequence durationInFrames={160} name="Close">
            <CloseScene />
          </TransitionSeries.Sequence>
        </TransitionSeries>
      </div>
    </AbsoluteFill>
  );
}

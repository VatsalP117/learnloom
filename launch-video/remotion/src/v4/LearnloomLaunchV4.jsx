import React from "react";
import {Audio} from "@remotion/media";
import {loadFont as loadManrope} from "@remotion/google-fonts/Manrope";
import {AbsoluteFill, Sequence, staticFile} from "remotion";
import {CloseScene} from "./CloseScene.jsx";
import {DomainScene} from "./DomainScene.jsx";
import {IntroScene} from "./IntroScene.jsx";
import {ProductActionScene} from "./ProductActionScene.jsx";
import {RetentionScene} from "./RetentionScene.jsx";
import {StatementScene} from "./StatementScene.jsx";
import {TodayJourneyScene} from "./TodayJourneyScene.jsx";

const {fontFamily} = loadManrope("normal", {
  weights: ["400", "500", "600", "700"],
  subsets: ["latin"],
});

export const V4_DURATION = 882;

export function LearnloomLaunchV4({sound = true}) {
  return (
    <AbsoluteFill style={{fontFamily, background: "#ffffff"}}>
      {sound ? (
        <Audio
          src={staticFile("launch-music-v1-1.m4a")}
          volume={(frame) => (frame < 816 ? 0.44 : Math.max(0, 0.44 * (882 - frame) / 66))}
        />
      ) : null}

      <Sequence from={0} durationInFrames={150} name="Introduction">
        <IntroScene />
      </Sequence>
      <Sequence from={150} durationInFrames={34} name="Choose statement">
        <StatementScene index="01 / 04">Choose one worthwhile lesson</StatementScene>
      </Sequence>
      <Sequence from={184} durationInFrames={110} name="Today">
        <TodayJourneyScene />
      </Sequence>
      <Sequence from={294} durationInFrames={34} name="Understand statement">
        <StatementScene index="02 / 04">Understand it</StatementScene>
      </Sequence>
      <Sequence from={328} durationInFrames={110} name="Build understanding">
        <ProductActionScene clip="lesson.webm" trimBefore={8} />
      </Sequence>
      <Sequence from={438} durationInFrames={34} name="Retrieve statement">
        <StatementScene index="03 / 04">Retrieve it</StatementScene>
      </Sequence>
      <Sequence from={472} durationInFrames={110} name="Spaced retrieval">
        <ProductActionScene clip="review.webm" trimBefore={10} />
      </Sequence>
      <Sequence from={582} durationInFrames={34} name="Keep statement">
        <StatementScene index="04 / 04">Keep it</StatementScene>
      </Sequence>
      <Sequence from={616} durationInFrames={110} name="Keep and publish">
        <RetentionScene />
      </Sequence>
      <Sequence from={726} durationInFrames={90} name="Domain">
        <DomainScene />
      </Sequence>
      <Sequence from={816} durationInFrames={66} name="Close">
        <CloseScene />
      </Sequence>
    </AbsoluteFill>
  );
}

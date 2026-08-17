import React from "react";
import {Video} from "@remotion/media";
import {AbsoluteFill, Sequence, staticFile, useCurrentFrame} from "remotion";
import {colors, fade} from "./theme.js";

const footage = {
  position: "absolute",
  left: 0,
  top: -60,
  width: 1920,
  height: 1200,
};

export function TodayJourneyScene() {
  const frame = useCurrentFrame();

  return (
    <AbsoluteFill style={{overflow: "hidden", background: colors.white}}>
      <Sequence from={0} durationInFrames={35} name="Choose lesson">
        <Video
          src={staticFile("product-clips/today.webm")}
          trimBefore={16}
          muted
          pauseWhenBuffering
          style={{...footage, opacity: fade(frame, 0, 6)}}
        />
      </Sequence>
      <Sequence from={35} durationInFrames={75} name="Lesson opens">
        <Video
          src={staticFile("product-clips/today.webm")}
          trimBefore={66}
          muted
          pauseWhenBuffering
          style={footage}
        />
      </Sequence>
    </AbsoluteFill>
  );
}


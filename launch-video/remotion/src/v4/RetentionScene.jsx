import React from "react";
import {Video} from "@remotion/media";
import {AbsoluteFill, staticFile, useCurrentFrame} from "remotion";
import {colors, fade} from "./theme.js";

const footage = {
  position: "absolute",
  left: 0,
  top: -60,
  width: 1920,
  height: 1200,
};

export function RetentionScene() {
  const frame = useCurrentFrame();
  const publishing = fade(frame, 60, 67);

  return (
    <AbsoluteFill style={{overflow: "hidden", background: colors.white}}>
      <Video
        src={staticFile("product-clips/library.webm")}
        trimBefore={10}
        muted
        pauseWhenBuffering
        style={{...footage, opacity: fade(frame, 0, 6)}}
      />
      <Video
        src={staticFile("product-clips/publishing.webm")}
        trimBefore={105}
        muted
        pauseWhenBuffering
        style={{...footage, opacity: publishing}}
      />
    </AbsoluteFill>
  );
}

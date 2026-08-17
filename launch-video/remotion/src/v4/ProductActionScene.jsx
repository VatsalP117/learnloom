import React from "react";
import {Video} from "@remotion/media";
import {AbsoluteFill, staticFile, useCurrentFrame} from "remotion";
import {colors, fade} from "./theme.js";

export function ProductActionScene({clip, trimBefore = 0, playbackRate = 1}) {
  const frame = useCurrentFrame();

  return (
    <AbsoluteFill style={{overflow: "hidden", background: colors.white}}>
      <Video
        src={staticFile(`product-clips/${clip}`)}
        trimBefore={trimBefore}
        playbackRate={playbackRate}
        muted
        pauseWhenBuffering
        style={{
          position: "absolute",
          left: 0,
          top: -60,
          width: 1920,
          height: 1200,
          opacity: fade(frame, 0, 6),
        }}
      />
    </AbsoluteFill>
  );
}

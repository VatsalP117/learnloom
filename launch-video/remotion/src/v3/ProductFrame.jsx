import React from "react";
import {Video} from "@remotion/media";
import {staticFile, useCurrentFrame} from "remotion";
import {colors, enter} from "./theme.js";

export function ProductFrame({clip, trimBefore = 0, playbackRate = 1, start = 0, style, videoStyle}) {
  const frame = useCurrentFrame();

  return (
    <div
      style={{
        position: "absolute",
        overflow: "hidden",
        border: "1px solid rgba(17,18,16,.14)",
        background: colors.white,
        opacity: enter(frame, start, 6),
        ...style,
      }}
    >
      <Video
        src={staticFile(`product-clips/${clip}`)}
        trimBefore={trimBefore}
        playbackRate={playbackRate}
        muted
        pauseWhenBuffering
        objectFit="cover"
        style={{width: "100%", height: "100%", ...videoStyle}}
      />
    </div>
  );
}

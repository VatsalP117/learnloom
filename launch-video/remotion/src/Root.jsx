import React from "react";
import { Composition } from "remotion";
import { LearnloomLaunch } from "./LearnloomLaunch.jsx";

export function RemotionRoot() {
  return (
    <Composition
      id="LearnloomLaunch"
      component={LearnloomLaunch}
      durationInFrames={36 * 30}
      fps={30}
      width={1920}
      height={1080}
      defaultProps={{ sound: true }}
    />
  );
}

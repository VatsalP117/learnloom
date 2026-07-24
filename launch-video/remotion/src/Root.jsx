import React from "react";
import { Composition } from "remotion";
import { LearnloomLaunch, LearnloomLaunchV2 } from "./LearnloomLaunch.jsx";

export function RemotionRoot() {
  return (
    <>
      <Composition
        id="LearnloomLaunch"
        component={LearnloomLaunch}
        durationInFrames={45 * 30}
        fps={30}
        width={1920}
        height={1080}
        defaultProps={{ sound: true }}
      />
      <Composition
        id="LearnloomLaunchV2"
        component={LearnloomLaunchV2}
        durationInFrames={27 * 30}
        fps={30}
        width={1920}
        height={1080}
        defaultProps={{ sound: true }}
      />
    </>
  );
}

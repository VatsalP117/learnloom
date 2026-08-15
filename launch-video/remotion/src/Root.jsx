import React from "react";
import { Composition } from "remotion";
import { LearnloomLaunch, LearnloomLaunchV2 } from "./LearnloomLaunch.jsx";
import {LearnloomLaunchV3, V3_DURATION} from "./v3/LearnloomLaunchV3.jsx";

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
        id="LearnloomLaunchV11"
        component={LearnloomLaunch}
        durationInFrames={45 * 30}
        fps={30}
        width={1920}
        height={1080}
        defaultProps={{ sound: true, audioVersion: "v1.1" }}
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
      <Composition
        id="LearnloomLaunchV3"
        component={LearnloomLaunchV3}
        durationInFrames={V3_DURATION}
        fps={30}
        width={1920}
        height={1080}
        defaultProps={{sound: true}}
      />
    </>
  );
}

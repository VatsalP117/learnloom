import React from "react";
import {AbsoluteFill, useCurrentFrame} from "remotion";
import {ProductFrame} from "./ProductFrame.jsx";
import {colors, enter, grain, sceneBase, stage} from "./theme.js";

export function RecallScene() {
  const frame = useCurrentFrame();
  const words = [
    {text: "READ.", at: 2, color: "#aaada7"},
    {text: "RETRIEVE.", at: 12, color: colors.white},
    {text: "REMEMBER.", at: 22, color: colors.lime},
  ];

  return (
    <AbsoluteFill style={{...sceneBase, background: colors.ink}}>
      <div style={{...grain, opacity: 0.1}} />
      <div style={{position: "absolute", zIndex: 25, left: 92, top: 90}}>
        {words.map((word) => {
          const reveal = enter(frame, word.at, 8);
          return (
            <div
              key={word.text}
              style={{
                color: word.color,
                fontSize: 90,
                fontWeight: 700,
                lineHeight: 0.94,
                letterSpacing: -5.5,
                opacity: reveal,
              }}
            >
              {word.text}
            </div>
          );
        })}
      </div>
      <div style={{position: "absolute", left: 98, bottom: 102, width: 430, color: "#b8bbb5", fontSize: 23, lineHeight: 1.45, opacity: enter(frame, 48, 12)}}>
        Short retrieval passes turn today’s reading into tomorrow’s judgment.
      </div>
      <ProductFrame
        clip="review.webm"
        playbackRate={1.08}
        start={7}
        style={{...stage}}
      />
    </AbsoluteFill>
  );
}

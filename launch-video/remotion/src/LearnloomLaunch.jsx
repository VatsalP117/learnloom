import React from "react";
import {
  AbsoluteFill,
  Audio,
  Easing,
  Img,
  Sequence,
  interpolate,
  spring,
  staticFile,
  useCurrentFrame,
} from "remotion";

const FPS = 30;
const W = 1920;
const H = 1080;

const palette = {
  ink: "#171917",
  forest: "#1f4533",
  green: "#39654d",
  sage: "#dce8dc",
  mist: "#edf2ec",
  line: "#dfe4dd",
  muted: "#6f756f",
  paper: "#f7f6f1",
  white: "#ffffff",
};

const fontSans = "Manrope, Inter, Helvetica Neue, Arial, sans-serif";
const fontDisplay = "Bricolage Grotesque, Manrope, Helvetica Neue, Arial, sans-serif";
const asset = (name) => staticFile(name);
const clamp = (value) => Math.max(0, Math.min(1, value));
const ease = (value) => Easing.inOut(Easing.cubic)(clamp(value));
const fadeIn = (frame, start = 0, duration = 16) =>
  interpolate(frame, [start, start + duration], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.out(Easing.cubic),
  });
const fadeOut = (frame, start, duration = 16) =>
  1 - interpolate(frame, [start, start + duration], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.in(Easing.cubic),
  });
const sceneFade = (frame, end, edge = 15) =>
  fadeIn(frame, 0, edge) * fadeOut(frame, end - edge, edge);
const typed = (value, frame, start, duration) =>
  value.slice(0, Math.floor(clamp((frame - start) / duration) * value.length));

export function LearnloomLaunch({sound = true}) {
  return (
    <AbsoluteFill style={{backgroundColor: palette.white, fontFamily: fontSans}}>
      {sound ? (
        <>
          <Audio src={asset("soundtrack.m4a")} volume={0.07} />
          <Audio src={asset("launch-beat.m4a")} volume={0.65} />
          <Audio src={asset("launch-sfx-v1.m4a")} volume={0.48} />
        </>
      ) : null}

      <Sequence from={0} durationInFrames={90}>
        <TypedHook />
      </Sequence>
      <Sequence from={75} durationInFrames={75}>
        <BrandBeat />
      </Sequence>
      <Sequence from={135} durationInFrames={80}>
        <PromiseBeat />
      </Sequence>
      <Sequence from={200} durationInFrames={820}>
        <ContinuousDemo />
      </Sequence>
      <Sequence from={995} durationInFrames={145}>
        <LearningHomeScene />
      </Sequence>
      <Sequence from={1134} durationInFrames={216}>
        <FinalScene />
      </Sequence>
    </AbsoluteFill>
  );
}

export function LearnloomLaunchV2({sound = true}) {
  return (
    <AbsoluteFill style={{backgroundColor: palette.white, fontFamily: fontSans}}>
      {sound ? (
        <>
          <Audio src={asset("soundtrack.m4a")} volume={0.05} />
          <Audio src={asset("launch-beat.m4a")} volume={0.72} />
        </>
      ) : null}

      <Sequence from={0} durationInFrames={66}>
        <V2TypeBeat text="Learning alone is hard." duration={66} />
      </Sequence>
      <Sequence from={55} durationInFrames={65}>
        <V2TypeBeat text="Learnloom makes it easier." duration={65} />
      </Sequence>
      <Sequence from={105} durationInFrames={65}>
        <V2TypeBeat text="One question becomes a practice." duration={65} />
      </Sequence>
      <Sequence from={150} durationInFrames={450}>
        <ContinuousDemo pace={1.78} showSchedule={false} />
      </Sequence>
      <Sequence from={580} durationInFrames={230}>
        <V2KineticClose />
      </Sequence>
    </AbsoluteFill>
  );
}

function V2TypeBeat({text, duration}) {
  const frame = useCurrentFrame();
  const value = typed(text, frame, 2, Math.min(38, duration - 18));
  return (
    <PaperScene opacity={sceneFade(frame, duration, 8)}>
      <div style={centerStage}>
        <div style={{...promiseLine, fontSize: 72}}>
          {value}
          <Cursor visible={frame < duration - 14} />
        </div>
      </div>
    </PaperScene>
  );
}

function V2KineticClose() {
  const frame = useCurrentFrame();
  const beats = [
    {text: "TURN", start: 0, end: 28},
    {text: "CURIOSITY", start: 28, end: 56},
    {text: "INTO", start: 56, end: 84},
    {text: "A PRACTICE", start: 84, end: 116},
  ];
  const brandIn = fadeIn(frame, 116, 13);
  return (
    <PaperScene opacity={1}>
      {beats.map((beat) => (
        <div
          key={beat.text}
          style={{
            ...v2KineticWord,
            opacity: fadeIn(frame, beat.start, 4) * fadeOut(frame, beat.end - 5, 5),
            transform: `translateY(${interpolate(fadeIn(frame, beat.start, 5), [0, 1], [22, 0])}px)`,
          }}
        >
          {beat.text}
        </div>
      ))}
      <div style={{...centerStage, opacity: brandIn}}>
        <div style={{display: "flex", alignItems: "center", gap: 18}}>
          <BrandMark size={54} />
          <div style={{...brandWord, fontSize: 48}}>Learnloom</div>
        </div>
        <div style={{marginTop: 28, color: palette.muted, fontSize: 18}}>
          Make curiosity a place you return to.
        </div>
        <div style={v2Url}>learnloom.blog</div>
      </div>
    </PaperScene>
  );
}

function TypedHook() {
  const frame = useCurrentFrame();
  const lineOne = typed("Endless things to read.", frame, 4, 25);
  const lineTwo = typed("One curiosity worth following.", frame, 37, 32);
  return (
    <PaperScene opacity={fadeOut(frame, 76, 14)}>
      <TopRule label="A quieter way to keep learning" />
      <div style={centerStage}>
        <div style={{...hookLine, opacity: fadeOut(frame, 32, 8)}}>
          {lineOne}
          <Cursor visible={frame < 32} />
        </div>
        <div style={{...hookLine, position: "absolute", opacity: fadeIn(frame, 34, 8)}}>
          {lineTwo}
          <Cursor visible={frame > 35 && frame < 72} />
        </div>
      </div>
      <FrameCounter current="01" />
    </PaperScene>
  );
}

function BrandBeat() {
  const frame = useCurrentFrame();
  const opacity = sceneFade(frame, 75, 10);
  const intro = typed("Introducing Learnloom", frame, 6, 42);
  return (
    <PaperScene opacity={opacity}>
      <div style={centerStage}>
        <div style={{...typedBrandLine, opacity: fadeIn(frame, 0, 8)}}>
          {intro}
          <Cursor visible={frame > 5 && frame < 50} />
        </div>
        <div style={{...brandUnderLine, opacity: fadeIn(frame, 50, 10)}}>
          <BrandMark size={34} />
          <span>A learning home that grows with you.</span>
        </div>
      </div>
    </PaperScene>
  );
}

function PromiseBeat() {
  const frame = useCurrentFrame();
  const opacity = sceneFade(frame, 80, 10);
  const promise = typed("Built to turn curiosity into a practice.", frame, 4, 54);
  return (
    <PaperScene opacity={opacity}>
      <div style={centerStage}>
        <div style={promiseLine}>
          {promise}
          <Cursor visible={frame > 4 && frame < 60} />
        </div>
      </div>
    </PaperScene>
  );
}

function ContinuousDemo({pace = 1, showSchedule = true}) {
  const frame = useCurrentFrame() * pace;
  const opacity = fadeIn(frame, 0, 14) * fadeOut(frame, 802, 18);
  const typedIntent = typed("I want to understand how AI systems learn and fail.", frame, 18, 66);
  const composerCollapse = ease((frame - (showSchedule ? 132 : 112)) / 50);
  const researchIn = fadeIn(frame, 138, 16);
  const researchOut = fadeOut(frame, 278, 18);
  const researchProgress = clamp((frame - 160) / 215);
  const dossierIn = fadeIn(frame, 286, 12);
  const blueprintExpand = ease((frame - 286) / 44);
  const lessonReveal = ease((frame - 428) / 38);
  const publishIn = fadeIn(frame, 552, 16);
  const archiveIn = ease((frame - 625) / 62);

  const composerLeft = interpolate(composerCollapse, [0, 1], [180, 84]);
  const composerTop = interpolate(composerCollapse, [0, 1], [238, 108]);
  const composerWidth = interpolate(composerCollapse, [0, 1], [1440, 700]);
  const composerHeight = interpolate(composerCollapse, [0, 1], [310, 112]);
  const composerFont = interpolate(composerCollapse, [0, 1], [29, 15]);

  const expandedLeft = interpolate(blueprintExpand, [0, 1], [660, 0]);
  const expandedTop = interpolate(blueprintExpand, [0, 1], [168, 75]);
  const expandedWidth = interpolate(blueprintExpand, [0, 1], [760, 1800]);
  const expandedHeight = interpolate(blueprintExpand, [0, 1], [660, 855]);
  const dossierLeft = interpolate(archiveIn, [0, 1], [expandedLeft, 40]);
  const dossierTop = interpolate(archiveIn, [0, 1], [expandedTop, 105]);
  const dossierWidth = interpolate(archiveIn, [0, 1], [expandedWidth, 1240]);
  const dossierHeight = interpolate(archiveIn, [0, 1], [expandedHeight, 760]);
  const dossierRadius = interpolate(
    archiveIn,
    [0, 1],
    [interpolate(blueprintExpand, [0, 1], [27, 0]), 24],
  );

  const sources = [
    {name: "Distill", meta: "Mechanisms", x: 1015, y: 250, delay: 0},
    {name: "MIT CSAIL", meta: "Research", x: 1330, y: 340, delay: 14},
    {name: "Model cards", meta: "Evidence", x: 1100, y: 635, delay: 28},
  ];
  const tasks = [
    {name: "Discover", at: 178},
    {name: "Read", at: 202},
    {name: "Challenge", at: 226},
    {name: "Teach", at: 250},
  ];

  return (
    <PaperScene opacity={opacity}>
      <div style={continuousBackdrop}>
        <div style={continuousGlowA} />
        <div style={continuousGlowB} />
      </div>

      <div style={continuousWorkspace}>
        <div style={workspaceHeader}>
          <div style={{display: "flex", alignItems: "center", gap: 12}}>
            <BrandMark size={32} />
            <span style={workspaceBrand}>Learnloom</span>
          </div>
          <div style={workspaceStatus}>
            <span style={workspaceStatusDot} />
            {frame < 96 || !showSchedule
              ? "New learning stream"
              : frame < 138
                ? "Daily lesson · 8:00 AM"
              : frame < 292
                ? "Researching"
                : frame < 428
                  ? "Blueprint ready"
                : frame < 625
                  ? "Lesson ready"
                  : "Learning home live"}
          </div>
        </div>

        <div
          style={{
            ...continuousComposer,
            left: composerLeft,
            top: composerTop,
            width: composerWidth,
            height: composerHeight,
            borderRadius: interpolate(composerCollapse, [0, 1], [25, 18]),
            opacity: fadeOut(frame, 260, 18),
          }}
        >
          <div
            style={{
              ...composerLabel,
              opacity: 1 - composerCollapse,
              height: interpolate(composerCollapse, [0, 1], [22, 0]),
              marginBottom: interpolate(composerCollapse, [0, 1], [18, 0]),
            }}
          >
            What should become clearer?
          </div>
          <div style={{...composerText, fontSize: composerFont}}>
            {typedIntent}
            <Cursor visible={frame > 17 && frame < 91} small />
          </div>
          <div
            style={{
              ...composerArrow,
              width: interpolate(composerCollapse, [0, 1], [52, 38]),
              height: interpolate(composerCollapse, [0, 1], [52, 38]),
              opacity: fadeIn(frame, 86, 8),
            }}
          >
            ↑
          </div>
          {showSchedule ? (
            <div
              style={{
                ...composerSchedule,
                opacity: (1 - composerCollapse) * fadeIn(frame, 84, 9),
                transform: `translateY(${interpolate(fadeIn(frame, 84, 9), [0, 1], [10, 0])}px)`,
              }}
            >
              <div style={scheduleIntro}>
                <span style={scheduleEyebrow}>DELIVER MY LESSON</span>
                <strong style={scheduleCopy}>A fresh lesson, every day.</strong>
              </div>
              <div style={scheduleChoices}>
                <span style={{...scheduleChoice, ...scheduleChoiceActive}}>
                  <i style={scheduleCheck}>✓</i> Daily
                </span>
                <span style={scheduleChoice}>Weekdays</span>
                <span style={scheduleChoice}>Weekly</span>
              </div>
              <div style={scheduleTime}>
                <span style={scheduleTimeLabel}>AT</span>
                <strong style={scheduleTimeValue}>8:00 AM</strong>
              </div>
            </div>
          ) : null}
        </div>

        <svg width="1800" height="930" style={{position: "absolute", inset: 0, opacity: researchIn * researchOut}}>
          {sources.map((source, index) => {
            const p = clamp((researchProgress - index * 0.08) * 1.55);
            const startX = source.x + 115;
            const startY = source.y + 48;
            const endX = 1150;
            const endY = 475;
            return (
              <React.Fragment key={source.name}>
                <path
                  d={`M ${startX} ${startY} C ${(startX + endX) / 2} ${startY}, ${(startX + endX) / 2} ${endY}, ${endX} ${endY}`}
                  fill="none"
                  stroke={palette.green}
                  strokeWidth="2"
                  strokeDasharray={`${p * 420} 420`}
                  opacity={0.16 + p * 0.48}
                />
                {p > 0.05 && p < 0.96 ? (
                  <circle
                    cx={startX + (endX - startX) * p}
                    cy={startY + (endY - startY) * p}
                    r="5"
                    fill={palette.forest}
                  />
                ) : null}
              </React.Fragment>
            );
          })}
        </svg>

        {sources.map((source) => {
          const reveal = fadeIn(frame, 145 + source.delay, 12);
          return (
            <div
              key={source.name}
              style={{
                ...continuousSource,
                left: source.x,
                top: source.y,
                opacity: reveal * researchOut,
                transform: `translateY(${interpolate(reveal, [0, 1], [18, 0])}px)`,
              }}
            >
              <span style={sourceGlyph}>↗</span>
              <span style={{display: "flex", flexDirection: "column", gap: 3}}>
                <strong>{source.name}</strong>
                <small style={{color: palette.muted, fontSize: 10}}>{source.meta}</small>
              </span>
            </div>
          );
        })}

        <div
          style={{
            ...continuousCore,
            opacity: researchIn * researchOut,
            transform: `translate(-50%, -50%) translateY(${interpolate(researchIn, [0, 1], [20, 0])}px)`,
          }}
        >
          <BrandMark size={45} />
          <div style={continuousCoreEyebrow}>
            {researchProgress > 0.97 ? "DOSSIER READY" : "FOLLOWING THE THREAD"}
          </div>
          <div style={continuousCoreTitle}>How AI systems<br />learn and fail</div>
          <div style={continuousProgress}>
            <span style={{width: `${researchProgress * 100}%`}} />
          </div>
        </div>

        <div style={{...taskRail, opacity: researchIn * researchOut}}>
          {tasks.map((task, index) => {
            const active = fadeIn(frame, task.at, 10);
            return (
              <div key={task.name} style={{...taskChip, opacity: 0.32 + active * 0.68}}>
                <span style={{...taskCheck, background: active > 0.8 ? palette.forest : "#d5dad5"}}>
                  {active > 0.8 ? "✓" : String(index + 1)}
                </span>
                {task.name}
              </div>
            );
          })}
        </div>

        <div
          style={{
            ...continuousDossier,
            left: dossierLeft,
            top: dossierTop,
            width: dossierWidth,
            height: dossierHeight,
            borderRadius: dossierRadius,
            opacity: dossierIn,
          }}
        >
          <div
            style={{
              ...blueprintBuildLabel,
              opacity: fadeIn(frame, 286, 8) * fadeOut(frame, 306, 8),
            }}
          >
            <span style={blueprintBuildDot} /> Structuring your learning path
          </div>
          <div
            style={{
              ...dossierBlueprint,
              opacity: fadeIn(frame, 306, 14) * fadeOut(frame, 410, 14),
            }}
          >
            <div style={blueprintHero}>
              <div style={dossierBlueprintEyebrow}>LEARNING BLUEPRINT</div>
              <div style={dossierBlueprintTitle}>How AI systems<br />learn and fail</div>
              <div style={blueprintMeta}>4 chapters · 18 min · source-grounded</div>
            </div>
            <div style={blueprintOutline}>
              {[
                ["01", "The mechanism", "How models turn examples into behavior"],
                ["02", "A worked example", "Follow one shortcut from data to failure"],
                ["03", "Skeptical review", "Where the explanation stops being enough"],
                ["04", "Retrieve", "Test the model you just built"],
              ].map(([num, title, note], index) => {
                const sectionIn = fadeIn(frame, 316 + index * 18, 12);
                return (
                  <div
                    key={num}
                    style={{
                      ...blueprintRow,
                      opacity: sectionIn,
                      transform: `translateY(${interpolate(sectionIn, [0, 1], [12, 0])}px)`,
                    }}
                  >
                    <span>{num}</span>
                    <div style={{display: "flex", flexDirection: "column", gap: 6}}>
                      <strong style={{color: palette.ink, fontSize: 17}}>{title}</strong>
                      <small style={{color: palette.muted, fontSize: 11, fontWeight: 500}}>{note}</small>
                    </div>
                    <b>✓</b>
                  </div>
                );
              })}
            </div>
          </div>

          <div
            style={{
              ...continuousLesson,
              opacity: fadeIn(frame, 428, 14),
              clipPath: `inset(0 0 ${(1 - lessonReveal) * 100}% 0)`,
              transform: `translateY(${interpolate(lessonReveal, [0, 1], [18, 0])}px)`,
            }}
          >
            <div style={lessonBrowserBar}>
              <span style={browserDots}>● ● ●</span>
              <span style={lessonUrl}>
                <i /> maya.learnloom.blog
              </span>
              <span style={{color: palette.muted, fontSize: 11}}>Today’s lesson</span>
            </div>
            <div style={lessonBody}>
              <div style={lessonStream}>INTELLIGENCE, EXPLAINED · 18 MIN</div>
              <div style={lessonImpactTitle}>When models learn<br />the wrong lesson.</div>
              <div style={lessonImpactSub}>
                A source-grounded dossier on shortcuts, failure modes, and what evidence can actually tell us.
              </div>
              <div style={lessonColumns}>
                <div style={lessonObjective}>
                  <span style={{display: "block", marginBottom: 12, color: palette.green, fontFamily: fontSans, fontSize: 9, fontWeight: 800, letterSpacing: 1.5}}>
                    LEARNING OBJECTIVE
                  </span>
                  Explain why a system can perform well while learning the wrong underlying pattern.
                </div>
                <div style={lessonMap}>
                  {["Mechanism", "Worked example", "Skeptical review", "Recall"].map((item, index) => (
                    <div key={item} style={{display: "flex", gap: 12, padding: "9px 0", borderBottom: `1px solid ${palette.line}`}}>
                      <b style={{color: palette.green}}>0{index + 1}</b>{item}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>

        <div
          style={{
            ...publishBubble,
            opacity: publishIn * fadeOut(frame, 610, 14),
            transform: `translateY(${interpolate(publishIn, [0, 1], [18, 0])}px)`,
          }}
        >
          <span style={publishBubbleCheck}>✓</span>
          <div style={{display: "flex", flexDirection: "column", gap: 3}}>
            <strong style={{fontSize: 13}}>Published automatically</strong>
            <small style={{color: palette.green, fontSize: 10}}>maya.learnloom.blog</small>
          </div>
        </div>

        {[
          {title: "How AI systems learn and fail", tint: "#dce8dc", x: 1340, y: 225, delay: 0},
          {title: "Why models learn shortcuts", tint: "#e7e1c8", x: 1340, y: 410, delay: 15},
          {title: "Reading evidence carefully", tint: "#d9e6eb", x: 1340, y: 595, delay: 30},
        ].map((card) => {
          const reveal = fadeIn(frame, 640 + card.delay, 14) * archiveIn;
          return (
            <div
              key={card.title}
              style={{
                ...archiveCard,
                left: interpolate(archiveIn, [0, 1], [1570, card.x]),
                top: card.y,
                opacity: reveal,
              }}
            >
              <div style={{...archiveCardArt, background: card.tint}} />
              <span style={{display: "block", color: palette.green, fontSize: 8, fontWeight: 800, letterSpacing: 1.2}}>LEARNING HISTORY</span>
              <strong style={{display: "block", width: 180, marginTop: 54, fontFamily: fontDisplay, fontSize: 15, lineHeight: 1.1}}>{card.title}</strong>
            </div>
          );
        })}

        <div style={{...continuousFooter, opacity: fadeOut(frame, 278, 16)}}>
          {[
            ["01", "Intent", frame >= 0],
            ["02", "Research", frame >= 138],
            ["03", "Dossier", frame >= 292],
            ["04", "Published", frame >= 552],
          ].map(([num, label, active]) => (
            <div key={num} style={{display: "flex", alignItems: "center", gap: 5, color: active ? palette.ink : "#a7aca7"}}>
              <span style={{display: "grid", placeItems: "center", width: 18, height: 18, borderRadius: 5, color: active ? palette.white : "#8f958f", background: active ? palette.forest : "#d7dbd7"}}>{num}</span>
              {label}
            </div>
          ))}
        </div>
      </div>
    </PaperScene>
  );
}

function IntentDemo() {
  const frame = useCurrentFrame();
  const opacity = sceneFade(frame, 165, 12);
  const typedProgress = ease((frame - 28) / 64);
  const sent = fadeIn(frame, 102, 9);

  return (
    <PaperScene opacity={opacity}>
      <TopRule label="01 · Name the thread" />
      <div style={{...intentHeadline, opacity: fadeIn(frame, 2, 10)}}>
        One question is enough.
      </div>
      <div style={composerStage}>
        <div style={composerCrop}>
          <Img src={asset("captures/02-intent-empty.png")} style={composerImage} />
          <div
            style={{
              position: "absolute",
              inset: 0,
              overflow: "hidden",
              clipPath: `inset(14% ${96 - typedProgress * 70}% 57% 4%)`,
            }}
          >
            <Img src={asset("captures/03-intent-filled.png")} style={composerImage} />
          </div>
          <Img
            src={asset("captures/03-intent-filled.png")}
            style={{...composerImage, position: "absolute", opacity: fadeIn(frame, 102, 7)}}
          />
        </div>
      </div>
      <div style={{...actionPill, opacity: sent, transform: `translateY(${interpolate(sent, [0, 1], [10, 0])}px)`}}>
        Intent understood <span style={{color: palette.green}}>✓</span>
      </div>
      <BeatWipe frame={frame} at={151} />
    </PaperScene>
  );
}

function SetupDemo() {
  const frame = useCurrentFrame();
  const opacity = sceneFade(frame, 165, 11);
  const split = frame < 74;
  const local = split ? frame : frame - 74;
  const screenshot = split ? "captures/04-sources-filled.png" : "captures/05-rhythm.png";
  const label = split ? "02 · Choose the signal" : "03 · Set a rhythm";
  const title = split ? "Sources you trust." : "A pace you can keep.";
  const slide = interpolate(fadeIn(local, 0, 9), [0, 1], [32, 0]);

  return (
    <PaperScene opacity={opacity}>
      <TopRule label={label} />
      <div style={{...setupTitle, opacity: fadeIn(local, 2, 9)}}>{title}</div>
      <div style={detailStage}>
        <div style={{...detailCrop, transform: `translateY(${slide}px)`}}>
          <Img
            src={asset(screenshot)}
            style={split ? sourceDetailImage : rhythmDetailImage}
          />
        </div>
        <div style={{...detailSignal, opacity: fadeIn(local, 18, 9)}}>
          <div style={detailSignalEyebrow}>{split ? "SIGNAL LOCKED" : "RHYTHM SET"}</div>
          <div style={detailSignalValue}>{split ? "Distill" : "Daily · 08:00"}</div>
          <div style={detailSignalMeta}>
            {split ? "Trusted source added" : "20-minute focused lesson"}
          </div>
        </div>
      </div>
      <div style={{...stepRail, opacity: fadeIn(frame, 8, 14)}}>
        {["Intent", "Sources", "Rhythm"].map((item, index) => (
          <div
            key={item}
            style={{
              ...stepItem,
              color: index <= (split ? 1 : 2) ? palette.ink : "#a6aaa6",
              background: index === (split ? 1 : 2) ? palette.white : "transparent",
              borderColor: index === (split ? 1 : 2) ? palette.line : "transparent",
            }}
          >
            <span style={{...stepDot, background: index <= (split ? 1 : 2) ? palette.forest : "#c9cdc8"}} />
            {item}
          </div>
        ))}
      </div>
      <BeatWipe frame={frame} at={68} />
    </PaperScene>
  );
}

function AgentOrchestration() {
  const frame = useCurrentFrame();
  const opacity = sceneFade(frame, 185, 12);
  const progress = clamp((frame - 12) / 130);
  const agents = [
    {label: "Discover", note: "Find the signal", x: 235, y: 330, delay: 0},
    {label: "Read", note: "Extract the mechanism", x: 235, y: 630, delay: 12},
    {label: "Challenge", note: "Test the claims", x: 1285, y: 330, delay: 24},
    {label: "Shape", note: "Build the lesson", x: 1285, y: 630, delay: 36},
  ];

  return (
    <PaperScene opacity={opacity}>
      <TopRule label="A small research team, working in concert" />
      <div style={{...agentHeadline, opacity: fadeIn(frame, 2, 18)}}>
        Learnloom follows the thread.
      </div>
      <div style={{...agentSubhead, opacity: fadeIn(frame, 12, 18)}}>
        Research, cross-check, structure, and teach—without turning learning into another project to manage.
      </div>
      <svg width={W} height={H} style={{position: "absolute", inset: 0}}>
        {agents.map((agent, index) => (
          <AgentPath key={agent.label} agent={agent} index={index} progress={progress} />
        ))}
        {[0, 1, 2].map((ring) => (
          <circle
            key={ring}
            cx="960"
            cy="604"
            r={100 + ring * 45 + Math.sin((frame + ring * 8) / 20) * 5}
            fill="none"
            stroke={palette.green}
            strokeWidth="1"
            opacity={0.12 - ring * 0.025}
          />
        ))}
      </svg>
      {agents.map((agent) => (
        <AgentCard key={agent.label} agent={agent} frame={frame} />
      ))}
      <div
        style={{
          ...agentCore,
          opacity: fadeIn(frame, 40, 12),
          transform: `translate(-50%, -50%) translateY(${interpolate(fadeIn(frame, 40, 12), [0, 1], [18, 0])}px)`,
        }}
      >
        <BrandMark size={48} />
        <div style={coreLabel}>{progress > 0.9 ? "DOSSIER READY" : "WEAVING THE PATH"}</div>
        <div style={coreTitle}>How AI systems<br />learn and fail</div>
        <div style={progressBar}>
          <span style={{display: "block", height: "100%", width: `${progress * 100}%`, background: palette.forest}} />
        </div>
      </div>
      <div style={{...statusLine, opacity: fadeIn(frame, 66, 12)}}>
        {progress > 0.9 ? "A lesson worth returning to." : "Connecting evidence to explanation…"}
      </div>
    </PaperScene>
  );
}

function AgentPath({agent, index, progress}) {
  const startX = agent.x < 960 ? agent.x + 310 : agent.x;
  const startY = agent.y + 68;
  const endX = 960;
  const endY = 604;
  const p = clamp((progress - index * 0.08) * 1.42);
  const dash = 430;
  return (
    <>
      <path
        d={`M ${startX} ${startY} C ${(startX + endX) / 2} ${startY}, ${(startX + endX) / 2} ${endY}, ${endX} ${endY}`}
        fill="none"
        stroke={palette.green}
        strokeWidth="2"
        strokeDasharray={`${dash * p} ${dash}`}
        opacity={0.22 + p * 0.45}
      />
      {p > 0.03 && p < 0.98 ? (
        <circle
          cx={startX + (endX - startX) * p}
          cy={startY + (endY - startY) * p}
          r="5"
          fill={palette.forest}
        />
      ) : null}
    </>
  );
}

function AgentCard({agent, frame}) {
  const reveal = fadeIn(frame, 14 + agent.delay, 10);
  return (
    <div
      style={{
        ...agentCard,
        left: agent.x,
        top: agent.y,
        opacity: reveal,
        transform: `translateY(${interpolate(reveal, [0, 1], [18, 0])}px)`,
      }}
    >
      <div style={agentIndex}>{String(Math.floor(agent.delay / 12) + 1).padStart(2, "0")}</div>
      <div>
        <div style={agentName}>{agent.label}</div>
        <div style={agentNote}>{agent.note}</div>
      </div>
      <div style={{...agentPulse, transform: `scale(${1 + Math.sin((frame + agent.delay) / 9) * 0.14})`}} />
    </div>
  );
}

function LessonDemo() {
  const frame = useCurrentFrame();
  const opacity = sceneFade(frame, 160, 11);
  const reveal = fadeIn(frame, 3, 10);
  const portal = ease(frame / 22);

  return (
    <PaperScene opacity={opacity}>
      <TopRule label="The path resolves into a lesson" />
      <div style={{...lessonTitle, opacity: fadeIn(frame, 3, 16)}}>Built for understanding.</div>
      <div
        style={{
          ...demoShell,
          top: 185,
          height: 790,
          clipPath: `inset(${(1 - portal) * 38}% ${(1 - portal) * 34}% round 28px)`,
        }}
      >
        <div
          style={{
            ...screenshotMotion,
            transform: `translateY(${interpolate(reveal, [0, 1], [32, 0])}px)`,
          }}
        >
          <Img src={asset("captures/06-lesson.png")} style={captureImage} />
        </div>
      </div>
      {[
        {label: "Mechanism", at: 34, left: 220},
        {label: "Evidence", at: 52, left: 380},
        {label: "Recall", at: 70, left: 520},
      ].map((item) => (
        <div key={item.label} style={{...lessonPill, left: item.left, opacity: fadeIn(frame, item.at, 13)}}>
          <span>✓</span> {item.label}
        </div>
      ))}
    </PaperScene>
  );
}

function PublishingDemo() {
  const frame = useCurrentFrame();
  const opacity = sceneFade(frame, 170, 11);
  const imageIn = spring({
    frame: frame - 10,
    fps: FPS,
    config: {damping: 20, stiffness: 86},
  });
  const deploy = clamp((frame - 52) / 52);
  const urlText = typed("maya.learnloom.blog", frame, 53, 28);

  return (
    <PaperScene opacity={opacity}>
      <TopRule label="Your learning has an address" />
      <div style={{...publishCopy, opacity: fadeIn(frame, 2, 16)}}>
        <div style={publishTitle}>From lesson<br />to learning home.</div>
        <div style={publishBody}>
          Every dossier is archived and ready on your personal Learnloom subdomain.
        </div>
        <div style={{...urlBar, opacity: fadeIn(frame, 48, 9)}}>
          <span style={liveDot} />
          {urlText}
          <Cursor visible={frame > 54 && frame < 84} small />
        </div>
        <div style={{...deployStatus, opacity: fadeIn(frame, 92, 10)}}>
          <span style={{...checkDisc, transform: `scale(${interpolate(deploy, [0, 1], [0.65, 1])})`}}>✓</span>
          Published automatically
        </div>
      </div>
      <div
        style={{
          ...publishWindow,
          opacity: imageIn,
          transform: `translateY(${interpolate(imageIn, [0, 1], [35, 0])}px)`,
        }}
      >
        <Img src={asset("captures/07-publishing.png")} style={captureImage} />
      </div>
      <div style={deployRail}>
        <span style={{width: `${deploy * 100}%`}} />
      </div>
    </PaperScene>
  );
}

function LibraryDemo() {
  const frame = useCurrentFrame();
  const opacity = sceneFade(frame, 150, 11);
  const settle = spring({
    frame: frame - 4,
    fps: FPS,
    config: {damping: 19, stiffness: 90},
  });
  return (
    <PaperScene opacity={opacity}>
      <div style={{...libraryWindow, opacity: settle, transform: `translate(-50%, -50%) translateY(${interpolate(settle, [0, 1], [30, 0])}px)`}}>
        <Img src={asset("captures/08-library.png")} style={captureImage} />
      </div>
      <div style={{...libraryVeil, opacity: fadeIn(frame, 42, 10)}} />
      <div style={{...libraryMessage, opacity: fadeIn(frame, 48, 10)}}>
        <div style={libraryEyebrow}>A durable learning history</div>
        <div style={libraryTitle}>Every lesson builds<br />on the last.</div>
      </div>
    </PaperScene>
  );
}

function LearningHomeScene() {
  const frame = useCurrentFrame();
  const opacity = sceneFade(frame, 145, 14);
  const windowIn = spring({
    frame: frame - 3,
    fps: FPS,
    config: {damping: 21, stiffness: 88},
  });
  const url = typed("maya.learnloom.blog", frame, 14, 30);
  const lessonCards = [
    {
      eyebrow: "INTELLIGENCE, EXPLAINED",
      title: "When models learn the wrong lesson",
      meta: "18 min · Today",
      tint: "#dce8dc",
    },
    {
      eyebrow: "SYSTEMS, EXPLAINED",
      title: "Why models reach for shortcuts",
      meta: "14 min · Yesterday",
      tint: "#e8e1c7",
    },
    {
      eyebrow: "EVIDENCE, EXPLAINED",
      title: "Reading a claim skeptically",
      meta: "16 min · July 21",
      tint: "#dce8ed",
    },
  ];

  return (
    <PaperScene opacity={opacity}>
      <div style={continuousBackdrop}>
        <div style={continuousGlowA} />
        <div style={continuousGlowB} />
      </div>
      <div
        style={{
          ...homeBrowser,
          opacity: windowIn,
          transform: `translateY(${interpolate(windowIn, [0, 1], [28, 0])}px)`,
        }}
      >
        <div style={homeBrowserBar}>
          <div style={{display: "flex", alignItems: "center", gap: 11}}>
            <BrandMark size={29} />
            <span style={workspaceBrand}>Learnloom</span>
          </div>
          <div style={{...homeUrl, opacity: fadeIn(frame, 10, 10)}}>
            <span style={homeLiveDot} />
            {url}
            <Cursor visible={frame > 14 && frame < 47} small />
          </div>
          <span style={homeBrowserLabel}>Maya’s learning home</span>
        </div>

        <div style={homeBody}>
          <div style={homeIdentity}>
            <div style={{...homeEyebrow, opacity: fadeIn(frame, 18, 10)}}>
              A HOME FOR EVERYTHING YOU LEARN
            </div>
            <div
              style={{
                ...homeTitle,
                opacity: fadeIn(frame, 22, 12),
                transform: `translateY(${interpolate(fadeIn(frame, 22, 12), [0, 1], [14, 0])}px)`,
              }}
            >
              Maya’s<br />Learning Garden
            </div>
            <div style={{...homeDescription, opacity: fadeIn(frame, 31, 12)}}>
              Questions followed carefully. Lessons kept together.
            </div>
            <div style={{...homeStats, opacity: fadeIn(frame, 41, 10)}}>
              <span><strong>12</strong> lessons</span>
              <i style={homeStatDivider} />
              <span><strong>4</strong> active threads</span>
            </div>
          </div>

          <div style={homeLibrary}>
            <div style={{...homeLibraryHead, opacity: fadeIn(frame, 31, 10)}}>
              <span>RECENT LESSONS</span>
              <span>View learning history ↗</span>
            </div>
            <div style={homeCardGrid}>
              {lessonCards.map((card, index) => {
                const reveal = fadeIn(frame, 38 + index * 9, 12);
                return (
                  <div
                    key={card.title}
                    style={{
                      ...homeLessonCard,
                      opacity: reveal,
                      transform: `translateX(${interpolate(reveal, [0, 1], [24, 0])}px)`,
                    }}
                  >
                    <div style={{...homeCardArt, background: card.tint}}>
                      <span style={homeCardIndex}>0{index + 1}</span>
                      <BrandMark size={26} />
                    </div>
                    <div style={homeCardCopy}>
                      <span style={homeCardEyebrow}>{card.eyebrow}</span>
                      <strong>{card.title}</strong>
                      <small>{card.meta}</small>
                    </div>
                    <span style={homeCardArrow}>↗</span>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>
      <div style={{...homePayoff, opacity: fadeIn(frame, 72, 12)}}>
        Your own subdomain. Your own place to keep learning.
      </div>
    </PaperScene>
  );
}

function FinalScene() {
  const frame = useCurrentFrame();
  const opacity = fadeIn(frame, 0, 18);
  const reveal = spring({
    frame: frame - 5,
    fps: FPS,
    config: {damping: 19, stiffness: 82},
  });
  const line = typed("Make curiosity a place you return to.", frame, 32, 62);
  return (
    <PaperScene opacity={opacity}>
      <div style={{...centerStage, transform: `translateY(${interpolate(reveal, [0, 1], [25, 0])}px)`}}>
        <div style={{display: "flex", alignItems: "center", gap: 16}}>
          <BrandMark size={48} />
          <div style={{...brandWord, fontSize: 34, letterSpacing: -1.5}}>Learnloom</div>
        </div>
        <div style={finalTitle}>
          {line}
          <Cursor visible={frame > 34 && frame < 98} />
        </div>
        <div style={{...finalSub, opacity: fadeIn(frame, 98, 16)}}>
          Your first learning stream is one question away.
        </div>
        <div style={{...finalButton, opacity: fadeIn(frame, 112, 16)}}>
          Start learning at learnloom.blog <span>→</span>
        </div>
      </div>
      <div style={{...finalFoot, opacity: fadeIn(frame, 125, 14)}}>
        Research that becomes understanding.
      </div>
    </PaperScene>
  );
}

function PaperScene({children, opacity = 1}) {
  return (
    <AbsoluteFill
      style={{
        overflow: "hidden",
        color: palette.ink,
        background: `radial-gradient(circle at 50% 10%, ${palette.white} 0%, ${palette.paper} 72%)`,
        opacity,
      }}
    >
      <div style={paperGrain} />
      {children}
    </AbsoluteFill>
  );
}

function BeatWipe({frame, at}) {
  const travel = ease((frame - at) / 9);
  const opacity = fadeIn(frame, at, 2) * fadeOut(frame, at + 7, 2);
  return (
    <div
      style={{
        position: "absolute",
        zIndex: 50,
        top: 0,
        bottom: 0,
        left: `${interpolate(travel, [0, 1], [-12, 112])}%`,
        width: 90,
        background: palette.white,
        boxShadow: "0 0 55px rgba(255,255,255,.9)",
        opacity,
        transform: "skewX(-6deg)",
      }}
    />
  );
}

function TopRule({label}) {
  return (
    <div style={topRule}>
      <BrandMark size={28} />
      <span>{label}</span>
      <span style={{marginLeft: "auto", color: "#a0a6a0"}}>LEARNLOOM</span>
    </div>
  );
}

function FrameCounter({current}) {
  return <div style={frameCounter}>{current} / 08</div>;
}

function Cursor({visible, small = false}) {
  return (
    <span
      style={{
        display: "inline-block",
        width: small ? 2 : 4,
        height: small ? "0.95em" : "0.9em",
        marginLeft: small ? 5 : 10,
        verticalAlign: "-0.08em",
        background: palette.forest,
        opacity: visible ? 1 : 0,
      }}
    />
  );
}

function BrandMark({size = 40}) {
  return (
    <div
      style={{
        width: size,
        height: size,
        display: "grid",
        placeItems: "center",
        flex: "0 0 auto",
        borderRadius: size * 0.27,
        color: palette.white,
        background: palette.ink,
        fontSize: size * 0.42,
        fontWeight: 760,
        boxShadow: "0 10px 25px rgba(23,25,23,.13)",
      }}
    >
      ✦
    </div>
  );
}

const paperGrain = {
  position: "absolute",
  inset: 0,
  pointerEvents: "none",
  opacity: 0.22,
  backgroundImage:
    "repeating-linear-gradient(0deg, rgba(40,55,43,.018) 0px, rgba(40,55,43,.018) 1px, transparent 1px, transparent 5px)",
};
const centerStage = {
  position: "absolute",
  inset: 0,
  display: "flex",
  flexDirection: "column",
  alignItems: "center",
  justifyContent: "center",
};
const topRule = {
  position: "absolute",
  zIndex: 20,
  top: 52,
  left: 68,
  right: 68,
  display: "flex",
  alignItems: "center",
  gap: 14,
  paddingBottom: 18,
  borderBottom: `1px solid ${palette.line}`,
  color: palette.green,
  fontSize: 12,
  fontWeight: 760,
  letterSpacing: 1.8,
  textTransform: "uppercase",
};
const frameCounter = {
  position: "absolute",
  right: 68,
  bottom: 54,
  color: "#9ba19b",
  fontSize: 12,
  fontWeight: 700,
  letterSpacing: 2,
};
const hookLine = {
  width: 1560,
  color: palette.ink,
  fontFamily: fontDisplay,
  fontSize: 76,
  fontWeight: 650,
  lineHeight: 1.05,
  letterSpacing: -4.2,
  textAlign: "center",
};
const typedBrandLine = {
  width: 1500,
  color: palette.ink,
  fontFamily: fontDisplay,
  fontSize: 78,
  fontWeight: 670,
  lineHeight: 1.04,
  letterSpacing: -4.1,
  textAlign: "center",
};
const brandUnderLine = {
  display: "flex",
  alignItems: "center",
  gap: 13,
  marginTop: 30,
  color: palette.muted,
  fontSize: 17,
  fontWeight: 620,
};
const promiseLine = {
  width: 1320,
  minHeight: 190,
  color: palette.ink,
  fontFamily: fontDisplay,
  fontSize: 76,
  fontWeight: 670,
  lineHeight: 1.03,
  letterSpacing: -4,
  textAlign: "center",
};
const v2KineticWord = {
  position: "absolute",
  inset: 0,
  display: "grid",
  placeItems: "center",
  color: palette.ink,
  fontFamily: fontDisplay,
  fontSize: 112,
  fontWeight: 780,
  letterSpacing: -5,
};
const v2Url = {
  marginTop: 36,
  padding: "13px 18px",
  border: `1px solid ${palette.line}`,
  borderRadius: 999,
  color: palette.green,
  background: palette.white,
  fontSize: 12,
  fontWeight: 800,
  letterSpacing: 1.5,
  textTransform: "uppercase",
};
const continuousBackdrop = {
  position: "absolute",
  inset: 0,
  overflow: "hidden",
  background: "linear-gradient(155deg, #eef3f8 0%, #f8f4ee 48%, #f1e4d6 100%)",
};
const continuousGlowA = {
  position: "absolute",
  width: 900,
  height: 900,
  left: -160,
  bottom: -480,
  borderRadius: "50%",
  background: "rgba(126,157,132,.28)",
  filter: "blur(70px)",
};
const continuousGlowB = {
  position: "absolute",
  width: 780,
  height: 780,
  right: -190,
  top: -400,
  borderRadius: "50%",
  background: "rgba(165,187,224,.34)",
  filter: "blur(80px)",
};
const continuousWorkspace = {
  position: "absolute",
  top: 55,
  left: 60,
  width: 1800,
  height: 930,
  overflow: "hidden",
  border: "1px solid rgba(104,119,109,.22)",
  borderRadius: 34,
  background: "rgba(250,250,247,.86)",
  boxShadow: "0 42px 120px rgba(35,47,39,.17)",
  backdropFilter: "blur(18px)",
};
const workspaceHeader = {
  position: "absolute",
  zIndex: 30,
  top: 0,
  left: 0,
  right: 0,
  height: 75,
  display: "flex",
  alignItems: "center",
  padding: "0 28px",
  boxSizing: "border-box",
  borderBottom: `1px solid ${palette.line}`,
  background: "rgba(255,255,255,.84)",
};
const workspaceBrand = {
  fontFamily: fontDisplay,
  fontSize: 17,
  fontWeight: 720,
  letterSpacing: -0.7,
};
const workspaceStatus = {
  display: "flex",
  alignItems: "center",
  gap: 9,
  marginLeft: "auto",
  color: palette.muted,
  fontSize: 12,
  fontWeight: 650,
};
const workspaceStatusDot = {
  width: 7,
  height: 7,
  borderRadius: "50%",
  background: palette.green,
  boxShadow: `0 0 0 5px ${palette.mist}`,
};
const continuousComposer = {
  position: "absolute",
  zIndex: 24,
  padding: "26px 30px",
  boxSizing: "border-box",
  border: `1px solid #cfd8cf`,
  background: "rgba(255,255,255,.96)",
  boxShadow: "0 28px 75px rgba(31,47,36,.14)",
};
const composerLabel = {
  overflow: "hidden",
  color: palette.green,
  fontSize: 11,
  fontWeight: 790,
  letterSpacing: 1.7,
  textTransform: "uppercase",
};
const composerText = {
  width: "calc(100% - 82px)",
  minHeight: 36,
  color: palette.ink,
  fontFamily: fontDisplay,
  fontWeight: 620,
  lineHeight: 1.25,
  letterSpacing: -0.7,
};
const composerArrow = {
  position: "absolute",
  right: 24,
  bottom: 24,
  display: "grid",
  placeItems: "center",
  borderRadius: "50%",
  color: palette.white,
  background: palette.ink,
  fontSize: 23,
};
const composerSchedule = {
  position: "absolute",
  left: 30,
  right: 94,
  bottom: 24,
  minHeight: 72,
  display: "flex",
  alignItems: "center",
  gap: 22,
  padding: "13px 15px",
  boxSizing: "border-box",
  border: `1px solid ${palette.line}`,
  borderRadius: 16,
  background: "#f7f8f4",
};
const scheduleIntro = {
  display: "flex",
  flexDirection: "column",
  gap: 5,
  minWidth: 220,
};
const scheduleEyebrow = {
  color: palette.green,
  fontSize: 8,
  fontWeight: 800,
  letterSpacing: 1.3,
};
const scheduleCopy = {
  color: palette.ink,
  fontFamily: fontDisplay,
  fontSize: 13,
  fontWeight: 650,
};
const scheduleChoices = {
  display: "flex",
  gap: 7,
};
const scheduleChoice = {
  display: "flex",
  alignItems: "center",
  gap: 6,
  padding: "9px 12px",
  border: `1px solid ${palette.line}`,
  borderRadius: 999,
  color: palette.muted,
  background: palette.white,
  fontSize: 11,
  fontWeight: 700,
};
const scheduleChoiceActive = {
  color: palette.white,
  borderColor: palette.forest,
  background: palette.forest,
};
const scheduleCheck = {
  fontSize: 9,
  fontStyle: "normal",
};
const scheduleTime = {
  display: "flex",
  flexDirection: "column",
  gap: 5,
  marginLeft: "auto",
  paddingLeft: 19,
  borderLeft: `1px solid ${palette.line}`,
  color: palette.green,
  fontSize: 9,
  fontWeight: 800,
  letterSpacing: 1,
};
const scheduleTimeLabel = {
  color: palette.muted,
  fontSize: 8,
};
const scheduleTimeValue = {
  color: palette.ink,
  fontFamily: fontDisplay,
  fontSize: 13,
  letterSpacing: 0,
};
const continuousSource = {
  position: "absolute",
  zIndex: 8,
  display: "flex",
  alignItems: "center",
  gap: 12,
  width: 230,
  padding: "14px 16px",
  boxSizing: "border-box",
  border: `1px solid ${palette.line}`,
  borderRadius: 17,
  background: "rgba(255,255,255,.9)",
  boxShadow: "0 16px 42px rgba(32,49,38,.09)",
  color: palette.ink,
  fontSize: 13,
};
const sourceGlyph = {
  display: "grid",
  placeItems: "center",
  width: 34,
  height: 34,
  flex: "0 0 auto",
  borderRadius: 10,
  color: palette.forest,
  background: palette.mist,
  fontSize: 15,
  fontWeight: 800,
};
const continuousCore = {
  position: "absolute",
  zIndex: 12,
  left: 1150,
  top: 475,
  width: 355,
  padding: "27px 29px",
  boxSizing: "border-box",
  border: `1px solid #ccd7cc`,
  borderRadius: 24,
  background: palette.white,
  boxShadow: "0 27px 72px rgba(27,45,33,.15)",
};
const continuousCoreEyebrow = {
  marginTop: 19,
  color: palette.green,
  fontSize: 9,
  fontWeight: 800,
  letterSpacing: 1.7,
};
const continuousCoreTitle = {
  marginTop: 10,
  fontFamily: fontDisplay,
  fontSize: 27,
  fontWeight: 680,
  lineHeight: 1.07,
  letterSpacing: -1.2,
};
const continuousProgress = {
  height: 5,
  marginTop: 22,
  overflow: "hidden",
  borderRadius: 99,
  background: "#e7ebe7",
};
const taskRail = {
  position: "absolute",
  zIndex: 9,
  left: 785,
  bottom: 135,
  display: "flex",
  gap: 9,
};
const taskChip = {
  display: "flex",
  alignItems: "center",
  gap: 7,
  padding: "9px 12px",
  border: `1px solid ${palette.line}`,
  borderRadius: 999,
  color: palette.ink,
  background: "rgba(255,255,255,.86)",
  fontSize: 11,
  fontWeight: 680,
};
const taskCheck = {
  display: "grid",
  placeItems: "center",
  width: 18,
  height: 18,
  borderRadius: "50%",
  color: palette.white,
  fontSize: 8,
  fontWeight: 800,
};
const continuousDossier = {
  position: "absolute",
  zIndex: 20,
  overflow: "hidden",
  border: `1px solid #cdd7cd`,
  borderRadius: 27,
  background: palette.white,
  boxShadow: "0 32px 90px rgba(28,46,34,.17)",
};
const blueprintBuildLabel = {
  position: "absolute",
  inset: 0,
  zIndex: 2,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  gap: 10,
  color: palette.green,
  fontSize: 11,
  fontWeight: 760,
  letterSpacing: 1.25,
  textTransform: "uppercase",
};
const blueprintBuildDot = {
  width: 8,
  height: 8,
  borderRadius: "50%",
  background: palette.green,
  boxShadow: `0 0 0 6px ${palette.mist}`,
};
const dossierBlueprint = {
  position: "absolute",
  inset: 0,
  display: "grid",
  gridTemplateColumns: ".78fr 1.22fr",
  alignItems: "center",
  gap: 96,
  padding: "66px 90px",
  boxSizing: "border-box",
};
const blueprintHero = {
  alignSelf: "center",
  paddingLeft: 24,
};
const blueprintOutline = {
  alignSelf: "center",
};
const dossierBlueprintEyebrow = {
  color: palette.green,
  fontSize: 10,
  fontWeight: 800,
  letterSpacing: 1.8,
};
const dossierBlueprintTitle = {
  marginTop: 20,
  fontFamily: fontDisplay,
  fontSize: 55,
  fontWeight: 690,
  lineHeight: 0.98,
  letterSpacing: -2.9,
};
const blueprintMeta = {
  display: "inline-flex",
  marginTop: 30,
  padding: "10px 13px",
  border: `1px solid ${palette.line}`,
  borderRadius: 999,
  color: palette.muted,
  background: "#f7f8f4",
  fontSize: 10,
  fontWeight: 700,
};
const blueprintRow = {
  display: "grid",
  gridTemplateColumns: "44px 1fr 28px",
  alignItems: "center",
  gap: 15,
  minHeight: 122,
  borderTop: `1px solid ${palette.line}`,
  color: palette.green,
  fontSize: 11,
  fontWeight: 760,
};
const continuousLesson = {
  position: "absolute",
  inset: 0,
  background: "#fdfcf8",
};
const lessonBrowserBar = {
  height: 61,
  display: "flex",
  alignItems: "center",
  gap: 22,
  padding: "0 23px",
  boxSizing: "border-box",
  borderBottom: `1px solid ${palette.line}`,
  background: "#f6f7f3",
};
const browserDots = {
  color: "#b4bab4",
  fontSize: 10,
  letterSpacing: 4,
};
const lessonUrl = {
  display: "flex",
  alignItems: "center",
  gap: 8,
  minWidth: 230,
  margin: "0 auto",
  padding: "8px 13px",
  border: `1px solid ${palette.line}`,
  borderRadius: 10,
  color: palette.muted,
  background: palette.white,
  fontSize: 11,
};
const lessonBody = {
  padding: "48px 80px",
  boxSizing: "border-box",
};
const lessonStream = {
  color: palette.green,
  fontSize: 10,
  fontWeight: 800,
  letterSpacing: 1.7,
};
const lessonImpactTitle = {
  marginTop: 16,
  fontFamily: fontDisplay,
  fontSize: 58,
  fontWeight: 680,
  lineHeight: 0.98,
  letterSpacing: -3,
};
const lessonImpactSub = {
  width: 690,
  marginTop: 20,
  color: palette.muted,
  fontSize: 15,
  lineHeight: 1.5,
};
const lessonColumns = {
  display: "grid",
  gridTemplateColumns: "1.35fr .65fr",
  gap: 38,
  marginTop: 48,
};
const lessonObjective = {
  padding: "25px 28px",
  border: `1px solid ${palette.line}`,
  borderRadius: 19,
  color: palette.ink,
  background: palette.white,
  fontFamily: fontDisplay,
  fontSize: 22,
  fontWeight: 610,
  lineHeight: 1.3,
};
const lessonMap = {
  display: "grid",
  gap: 8,
  color: palette.muted,
  fontSize: 12,
};
const publishBubble = {
  position: "absolute",
  zIndex: 28,
  right: 62,
  top: 93,
  display: "flex",
  alignItems: "center",
  gap: 12,
  minWidth: 285,
  padding: "14px 17px",
  boxSizing: "border-box",
  border: `1px solid #cdd8cd`,
  borderRadius: 17,
  background: "rgba(255,255,255,.96)",
  boxShadow: "0 16px 44px rgba(29,46,35,.14)",
};
const publishBubbleCheck = {
  display: "grid",
  placeItems: "center",
  width: 31,
  height: 31,
  borderRadius: "50%",
  color: palette.white,
  background: palette.forest,
  fontSize: 13,
  fontWeight: 800,
};
const archiveCard = {
  position: "absolute",
  zIndex: 23,
  width: 310,
  height: 160,
  padding: 16,
  boxSizing: "border-box",
  border: `1px solid ${palette.line}`,
  borderRadius: 18,
  background: palette.white,
  boxShadow: "0 18px 48px rgba(29,45,35,.12)",
};
const archiveCardArt = {
  position: "absolute",
  top: 14,
  right: 14,
  width: 85,
  height: 55,
  borderRadius: 12,
};
const continuityMessage = {
  position: "absolute",
  zIndex: 35,
  left: 0,
  right: 0,
  bottom: 74,
  color: palette.ink,
  fontFamily: fontDisplay,
  fontSize: 29,
  fontWeight: 680,
  letterSpacing: -1.2,
  textAlign: "center",
};
const homeBrowser = {
  position: "absolute",
  top: 72,
  left: 105,
  width: 1710,
  height: 825,
  overflow: "hidden",
  border: "1px solid rgba(104,119,109,.24)",
  borderRadius: 30,
  background: "#fbfaf6",
  boxShadow: "0 42px 120px rgba(35,47,39,.18)",
};
const homeBrowserBar = {
  height: 74,
  display: "grid",
  gridTemplateColumns: "1fr 1.15fr 1fr",
  alignItems: "center",
  padding: "0 27px",
  boxSizing: "border-box",
  borderBottom: `1px solid ${palette.line}`,
  background: "rgba(255,255,255,.88)",
};
const homeUrl = {
  justifySelf: "center",
  display: "flex",
  alignItems: "center",
  width: 340,
  minHeight: 38,
  padding: "0 15px",
  boxSizing: "border-box",
  border: `1px solid ${palette.line}`,
  borderRadius: 12,
  color: palette.green,
  background: palette.white,
  fontSize: 12,
  fontWeight: 700,
  letterSpacing: 0.25,
};
const homeLiveDot = {
  width: 7,
  height: 7,
  flex: "0 0 auto",
  marginRight: 10,
  borderRadius: "50%",
  background: palette.green,
  boxShadow: `0 0 0 4px ${palette.mist}`,
};
const homeBrowserLabel = {
  justifySelf: "end",
  color: palette.muted,
  fontSize: 11,
  fontWeight: 650,
};
const homeBody = {
  display: "grid",
  gridTemplateColumns: ".78fr 1.22fr",
  gap: 76,
  height: 751,
  padding: "72px 78px 64px",
  boxSizing: "border-box",
};
const homeIdentity = {
  display: "flex",
  flexDirection: "column",
  justifyContent: "center",
  paddingLeft: 10,
};
const homeEyebrow = {
  color: palette.green,
  fontSize: 10,
  fontWeight: 800,
  letterSpacing: 1.8,
};
const homeTitle = {
  marginTop: 20,
  color: palette.ink,
  fontFamily: fontDisplay,
  fontSize: 58,
  fontWeight: 690,
  lineHeight: 0.98,
  letterSpacing: -3.1,
};
const homeDescription = {
  width: 470,
  marginTop: 26,
  color: palette.muted,
  fontSize: 15,
  lineHeight: 1.55,
};
const homeStats = {
  display: "flex",
  alignItems: "center",
  gap: 16,
  marginTop: 34,
  color: palette.muted,
  fontSize: 12,
};
const homeStatDivider = {
  width: 1,
  height: 16,
  background: palette.line,
};
const homeLibrary = {
  display: "flex",
  flexDirection: "column",
  justifyContent: "center",
};
const homeLibraryHead = {
  display: "flex",
  justifyContent: "space-between",
  marginBottom: 14,
  color: palette.green,
  fontSize: 9,
  fontWeight: 800,
  letterSpacing: 1.2,
};
const homeCardGrid = {
  display: "grid",
  gap: 12,
};
const homeLessonCard = {
  position: "relative",
  display: "grid",
  gridTemplateColumns: "126px 1fr",
  alignItems: "center",
  minHeight: 142,
  padding: 13,
  boxSizing: "border-box",
  border: `1px solid ${palette.line}`,
  borderRadius: 19,
  background: palette.white,
  boxShadow: "0 14px 38px rgba(31,47,36,.08)",
};
const homeCardArt = {
  position: "relative",
  display: "flex",
  flexDirection: "column",
  alignItems: "flex-start",
  justifyContent: "space-between",
  height: 112,
  padding: 12,
  boxSizing: "border-box",
  overflow: "hidden",
  borderRadius: 13,
  color: palette.green,
};
const homeCardIndex = {
  fontFamily: fontDisplay,
  fontSize: 16,
  fontWeight: 650,
};
const homeCardCopy = {
  display: "flex",
  flexDirection: "column",
  gap: 8,
  padding: "0 44px 0 22px",
};
const homeCardEyebrow = {
  color: palette.green,
  fontSize: 8,
  fontWeight: 800,
  letterSpacing: 1.1,
};
const homeCardArrow = {
  position: "absolute",
  right: 19,
  top: "50%",
  color: palette.green,
  fontSize: 16,
  transform: "translateY(-50%)",
};
const homePayoff = {
  position: "absolute",
  left: 0,
  right: 0,
  bottom: 35,
  color: palette.ink,
  fontFamily: fontDisplay,
  fontSize: 23,
  fontWeight: 660,
  letterSpacing: -0.8,
  textAlign: "center",
};
const continuousFooter = {
  position: "absolute",
  zIndex: 34,
  left: 28,
  bottom: 22,
  display: "flex",
  gap: 24,
  color: palette.muted,
  fontSize: 10,
  fontWeight: 720,
  letterSpacing: 0.7,
  textTransform: "uppercase",
};
const brandWord = {
  color: palette.ink,
  fontFamily: fontDisplay,
  fontSize: 64,
  fontWeight: 720,
  letterSpacing: -3.2,
};
const intentHeadline = {
  position: "absolute",
  zIndex: 12,
  top: 136,
  left: 0,
  right: 0,
  fontFamily: fontDisplay,
  fontSize: 53,
  fontWeight: 680,
  letterSpacing: -2.5,
  textAlign: "center",
};
const composerStage = {
  position: "absolute",
  top: 245,
  left: 120,
  right: 120,
  height: 680,
  padding: 22,
  boxSizing: "border-box",
  border: `1px solid ${palette.line}`,
  borderRadius: 34,
  background: "rgba(255,255,255,.72)",
  boxShadow: "0 34px 95px rgba(34,49,39,.13)",
};
const composerCrop = {
  position: "relative",
  width: "100%",
  height: "100%",
  overflow: "hidden",
  borderRadius: 22,
  background: "#d7deef",
};
const composerImage = {
  position: "absolute",
  width: 2677,
  height: 1512,
  maxWidth: "none",
  left: -694,
  top: -357,
};
const demoShell = {
  position: "absolute",
  top: 155,
  left: 70,
  width: 1780,
  height: 835,
  overflow: "hidden",
  border: `1px solid ${palette.line}`,
  borderRadius: 28,
  background: palette.white,
  boxShadow: "0 34px 90px rgba(34,49,39,.14)",
};
const screenshotMotion = {
  position: "absolute",
  inset: 0,
  width: "100%",
  height: "100%",
  transformOrigin: "60% 40%",
  willChange: "transform",
};
const captureImage = {
  display: "block",
  width: "100%",
  height: "100%",
  objectFit: "cover",
  objectPosition: "center center",
};
const floatingCaption = {
  position: "absolute",
  zIndex: 10,
  left: 110,
  bottom: 58,
  padding: "13px 18px",
  border: `1px solid ${palette.line}`,
  borderRadius: 999,
  color: palette.ink,
  background: "rgba(255,255,255,.94)",
  fontSize: 15,
  fontWeight: 650,
  boxShadow: "0 12px 30px rgba(22,32,25,.1)",
};
const actionPill = {
  position: "absolute",
  zIndex: 10,
  right: 108,
  bottom: 58,
  display: "flex",
  alignItems: "center",
  gap: 12,
  padding: "13px 18px",
  borderRadius: 999,
  color: palette.ink,
  background: palette.white,
  border: `1px solid ${palette.line}`,
  fontSize: 14,
  fontWeight: 700,
  boxShadow: "0 12px 30px rgba(22,32,25,.1)",
};
const detailStage = {
  position: "absolute",
  top: 225,
  left: 110,
  right: 110,
  height: 730,
  overflow: "hidden",
  border: `1px solid ${palette.line}`,
  borderRadius: 30,
  background: palette.white,
  boxShadow: "0 30px 85px rgba(30,48,36,.12)",
};
const detailCrop = {
  position: "absolute",
  inset: 0,
  overflow: "hidden",
  background: "#d8dfef",
};
const sourceDetailImage = {
  position: "absolute",
  width: 2524,
  height: 1426,
  maxWidth: "none",
  left: -686,
  top: -300,
};
const rhythmDetailImage = {
  position: "absolute",
  width: 2390,
  height: 1350,
  maxWidth: "none",
  left: -650,
  top: -225,
};
const detailSignal = {
  position: "absolute",
  right: 34,
  bottom: 34,
  width: 330,
  padding: "20px 22px",
  boxSizing: "border-box",
  border: `1px solid ${palette.line}`,
  borderRadius: 19,
  background: "rgba(255,255,255,.95)",
  boxShadow: "0 18px 48px rgba(27,44,33,.14)",
};
const detailSignalEyebrow = {
  color: palette.green,
  fontSize: 9,
  fontWeight: 800,
  letterSpacing: 1.6,
};
const detailSignalValue = {
  marginTop: 10,
  color: palette.ink,
  fontFamily: fontDisplay,
  fontSize: 26,
  fontWeight: 680,
  letterSpacing: -1,
};
const detailSignalMeta = {
  marginTop: 5,
  color: palette.muted,
  fontSize: 12,
};
const setupTitle = {
  position: "absolute",
  zIndex: 12,
  left: 118,
  top: 128,
  fontFamily: fontDisplay,
  fontSize: 46,
  fontWeight: 680,
  letterSpacing: -2,
};
const stepRail = {
  position: "absolute",
  zIndex: 14,
  right: 110,
  top: 123,
  display: "flex",
  gap: 8,
  padding: 6,
  borderRadius: 999,
  background: "rgba(239,242,238,.92)",
};
const stepItem = {
  display: "flex",
  alignItems: "center",
  gap: 8,
  padding: "9px 13px",
  border: "1px solid transparent",
  borderRadius: 999,
  fontSize: 12,
  fontWeight: 680,
};
const stepDot = {
  width: 7,
  height: 7,
  borderRadius: "50%",
};
const agentHeadline = {
  position: "absolute",
  top: 132,
  left: 0,
  right: 0,
  fontFamily: fontDisplay,
  fontSize: 53,
  fontWeight: 680,
  letterSpacing: -2.6,
  textAlign: "center",
};
const agentSubhead = {
  position: "absolute",
  top: 202,
  left: "50%",
  width: 820,
  transform: "translateX(-50%)",
  color: palette.muted,
  fontSize: 17,
  lineHeight: 1.5,
  textAlign: "center",
};
const agentCard = {
  position: "absolute",
  display: "flex",
  alignItems: "center",
  gap: 16,
  width: 310,
  minHeight: 106,
  padding: "18px 20px",
  boxSizing: "border-box",
  border: `1px solid ${palette.line}`,
  borderRadius: 18,
  background: "rgba(255,255,255,.9)",
  boxShadow: "0 18px 45px rgba(31,52,39,.08)",
};
const agentIndex = {
  display: "grid",
  placeItems: "center",
  width: 40,
  height: 40,
  borderRadius: 12,
  color: palette.forest,
  background: palette.mist,
  fontSize: 11,
  fontWeight: 800,
  letterSpacing: 1,
};
const agentName = {
  color: palette.ink,
  fontSize: 16,
  fontWeight: 750,
};
const agentNote = {
  marginTop: 5,
  color: palette.muted,
  fontSize: 12,
};
const agentPulse = {
  width: 8,
  height: 8,
  marginLeft: "auto",
  borderRadius: "50%",
  background: palette.green,
  boxShadow: `0 0 0 6px ${palette.mist}`,
};
const agentCore = {
  position: "absolute",
  left: 960,
  top: 604,
  width: 380,
  padding: "30px 32px",
  boxSizing: "border-box",
  border: `1px solid #ccd9cd`,
  borderRadius: 25,
  background: palette.white,
  boxShadow: "0 26px 70px rgba(25,48,33,.14)",
};
const coreLabel = {
  marginTop: 22,
  color: palette.green,
  fontSize: 10,
  fontWeight: 800,
  letterSpacing: 1.8,
};
const coreTitle = {
  marginTop: 10,
  fontFamily: fontDisplay,
  fontSize: 29,
  fontWeight: 680,
  lineHeight: 1.08,
  letterSpacing: -1.2,
};
const progressBar = {
  height: 5,
  marginTop: 25,
  overflow: "hidden",
  borderRadius: 999,
  background: "#e8ece7",
};
const statusLine = {
  position: "absolute",
  left: 0,
  right: 0,
  bottom: 78,
  color: palette.green,
  fontSize: 14,
  fontWeight: 650,
  letterSpacing: 0.2,
  textAlign: "center",
};
const lessonTitle = {
  position: "absolute",
  zIndex: 10,
  left: 112,
  top: 123,
  fontFamily: fontDisplay,
  fontSize: 48,
  fontWeight: 680,
  letterSpacing: -2,
};
const lessonPill = {
  position: "absolute",
  zIndex: 15,
  bottom: 46,
  display: "flex",
  alignItems: "center",
  gap: 8,
  padding: "11px 16px",
  border: `1px solid ${palette.line}`,
  borderRadius: 999,
  color: palette.ink,
  background: palette.white,
  fontSize: 13,
  fontWeight: 700,
  boxShadow: "0 10px 25px rgba(24,38,29,.09)",
};
const publishCopy = {
  position: "absolute",
  left: 118,
  top: 205,
  width: 560,
};
const publishTitle = {
  fontFamily: fontDisplay,
  fontSize: 62,
  fontWeight: 680,
  lineHeight: 1.02,
  letterSpacing: -3,
};
const publishBody = {
  width: 470,
  marginTop: 28,
  color: palette.muted,
  fontSize: 18,
  lineHeight: 1.55,
};
const urlBar = {
  display: "flex",
  alignItems: "center",
  width: "fit-content",
  minWidth: 288,
  minHeight: 48,
  marginTop: 42,
  padding: "0 18px",
  border: `1px solid #cfdacf`,
  borderRadius: 999,
  color: palette.forest,
  background: palette.white,
  fontSize: 15,
  fontWeight: 720,
  boxShadow: "0 12px 30px rgba(31,55,39,.08)",
};
const liveDot = {
  width: 8,
  height: 8,
  marginRight: 10,
  borderRadius: "50%",
  background: palette.green,
  boxShadow: `0 0 0 5px ${palette.mist}`,
};
const deployStatus = {
  display: "flex",
  alignItems: "center",
  gap: 11,
  marginTop: 20,
  color: palette.green,
  fontSize: 14,
  fontWeight: 680,
};
const checkDisc = {
  display: "grid",
  placeItems: "center",
  width: 24,
  height: 24,
  borderRadius: "50%",
  color: palette.white,
  background: palette.forest,
  fontSize: 12,
};
const publishWindow = {
  position: "absolute",
  top: 155,
  left: 760,
  width: 1160,
  height: 805,
  overflow: "hidden",
  border: `1px solid ${palette.line}`,
  borderRadius: 26,
  background: palette.white,
  boxShadow: "0 30px 85px rgba(28,46,34,.14)",
  transformOrigin: "center center",
};
const deployRail = {
  position: "absolute",
  left: 118,
  bottom: 108,
  width: 480,
  height: 3,
  overflow: "hidden",
  borderRadius: 99,
  background: "#e2e6e1",
};
const libraryWindow = {
  position: "absolute",
  left: 960,
  top: 540,
  width: 1780,
  height: 920,
  overflow: "hidden",
  border: `1px solid ${palette.line}`,
  borderRadius: 28,
  background: palette.white,
  boxShadow: "0 34px 90px rgba(34,49,39,.14)",
};
const libraryVeil = {
  position: "absolute",
  inset: 0,
  background: "rgba(247,246,241,.83)",
  backdropFilter: "blur(3px)",
};
const libraryMessage = {
  position: "absolute",
  inset: 0,
  display: "flex",
  flexDirection: "column",
  alignItems: "center",
  justifyContent: "center",
  textAlign: "center",
};
const libraryEyebrow = {
  color: palette.green,
  fontSize: 12,
  fontWeight: 800,
  letterSpacing: 2.2,
  textTransform: "uppercase",
};
const libraryTitle = {
  marginTop: 22,
  fontFamily: fontDisplay,
  fontSize: 70,
  fontWeight: 680,
  lineHeight: 1.02,
  letterSpacing: -3.6,
};
const finalTitle = {
  width: 1420,
  minHeight: 165,
  marginTop: 92,
  fontFamily: fontDisplay,
  fontSize: 74,
  fontWeight: 680,
  lineHeight: 1.04,
  letterSpacing: -4,
  textAlign: "center",
};
const finalSub = {
  marginTop: 8,
  color: palette.muted,
  fontSize: 19,
};
const finalButton = {
  display: "flex",
  alignItems: "center",
  gap: 20,
  marginTop: 34,
  padding: "18px 25px",
  borderRadius: 999,
  color: palette.white,
  background: palette.ink,
  fontSize: 15,
  fontWeight: 740,
  boxShadow: "0 16px 36px rgba(23,25,23,.18)",
};
const finalFoot = {
  position: "absolute",
  left: 0,
  right: 0,
  bottom: 54,
  color: palette.green,
  fontSize: 11,
  fontWeight: 760,
  letterSpacing: 2.1,
  textAlign: "center",
  textTransform: "uppercase",
};

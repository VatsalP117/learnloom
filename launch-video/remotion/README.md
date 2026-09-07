# Learnloom launch film — Remotion

> **V1/V2/V3 are legacy compositions.** They contain older UI, pacing, or publication
> behavior. `LearnloomLaunchV4` is the current interaction-led cut; review its final
> claims and pacing before publishing.

The legacy 45-second composition is built around the July 2026 Learnloom UI.
The current V4 cut uses the refined August product, a clean white visual system,
black editorial typography, restrained lime accents, and action-led product
recordings.

The main demo is a continuous animated Learnloom workspace rather than a
sequence of screenshot slides. The learning intent persists while trusted
sources feed the research core, the core assembles a dossier, the dossier opens
into the lesson, and then shows a legacy automatic-publication story. The
replacement cut must show private-by-default learning and optional publishing
only after value.

The workspace is built from Learnloom's real product language and current
information architecture. Product energy comes from persistent components,
progressive generation, and purposeful artifact transformations rather than
camera zooms or full-page screenshot cuts. Captures in `public/captures/` remain
available as visual references.

The soundtrack combines the original ambient bed with an original restrained
beat track in `public/launch-beat.m4a`.

The V1.1 experiment keeps the original composition and tactile SFX but swaps
the music for `public/launch-music-v1-1.m4a`: an original, faster 128 BPM
electronic score with crisp percussion, warm bass, and a restrained arpeggio.

From this directory:

```sh
npm install
npm run studio
npm run render
npm run music:v1.1
npm run render:v1.1
npm run render:v2
npm run capture
npm run render:v3
npm run render:v4
```

The 45-second composition is `LearnloomLaunch` and renders to
`../output/learnloom-launch-remotion.mp4`. The separate 27-second fast cut is
`LearnloomLaunchV2` and renders to `../output/learnloom-launch-v2.mp4`.

`LearnloomLaunchV3` is a 26.8-second current-product cut. Its six scenes live in
`src/v3/`, and its demo-only product recordings live in
`public/product-clips/`. Start the Vite demo at `http://127.0.0.1:4173`, run
`npm run capture`, then render with `npm run render:v3`. The output is
`../output/learnloom-launch-v3.mp4`.

`LearnloomLaunchV4` is the current 29.4-second launch cut. It replaces decorative
transitions with a direct choose → understand → retrieve → keep story, uses
full-bleed product actions, and carries only the continuous music bed—no typing
or transition sound effects. A dedicated domain reveal introduces each learner's
personal learning home. Render it with `npm run render:v4`; the output is
`../output/learnloom-launch-v4.mp4`.

## Micro launch cut

`LearnloomMicroLaunch` is an additive 12-second (360-frame) micro launch film
with a separate scene file for each beat under `src/micro/`. It opens on a
single typed question, assembles three visible sources into a Knowledge
Dossier, morphs the dossier into a rounded product window playing the existing
`public/product-clips/lesson.webm` and `library.webm`, and closes on a clean
brand card. It uses Bricolage Grotesque for display copy, Manrope for UI text,
and only the existing `launch-music-v1-1.m4a` plus restrained key cues from
`public/sfx-source/`. Scene boundaries: 0–78 question, 78–154 sources,
154–270 product payoff, 270–312 learning home, 312–360 brand close.

```sh
npm run render:micro
npm run still:micro
```

`render:micro` exports `../output/learnloom-micro-launch.mp4`;
`still:micro` writes `../output/learnloom-micro-launch-preview.png` at frame 220.

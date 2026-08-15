# Learnloom launch film — Remotion

> **V1/V2 are legacy compositions.** They contain older UI and publication
> behavior. `LearnloomLaunchV3` is the current footage-led cut; review its final
> claims and pacing before publishing.

The legacy 45-second composition is built around the July 2026 Learnloom UI.
The current V3 cut uses the refined August product, a white/warm-paper visual
system, black editorial typography, sharp lime accents, and fresh product
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
```

The 45-second composition is `LearnloomLaunch` and renders to
`../output/learnloom-launch-remotion.mp4`. The separate 27-second fast cut is
`LearnloomLaunchV2` and renders to `../output/learnloom-launch-v2.mp4`.

`LearnloomLaunchV3` is a 26.8-second current-product cut. Its six scenes live in
`src/v3/`, and its demo-only product recordings live in
`public/product-clips/`. Start the Vite demo at `http://127.0.0.1:4173`, run
`npm run capture`, then render with `npm run render:v3`. The output is
`../output/learnloom-launch-v3.mp4`.

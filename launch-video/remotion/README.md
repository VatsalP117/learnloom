# Learnloom launch film — Remotion

> **Legacy composition — do not publish.** The current composition and captures
> contain pre-wedge personas, examples, UI, and publication behavior. Rendering
> successfully proves only that the media project builds; it does not prove the
> product or marketing is current.

The current launch film is a 45-second Remotion composition built around the
July 2026 Learnloom UI. It uses one consistent white/warm-paper visual system,
black editorial typography, restrained forest accents, and the product's calm
landscape artwork.

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
```

The 45-second composition is `LearnloomLaunch` and renders to
`../output/learnloom-launch-remotion.mp4`. The separate 27-second fast cut is
`LearnloomLaunchV2` and renders to `../output/learnloom-launch-v2.mp4`.

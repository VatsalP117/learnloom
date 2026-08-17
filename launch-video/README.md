# Learnloom launch film

> **Review before publication.** The V1/V2 renders and legacy storyboard predate
> the current product UI. `LearnloomLaunchV4` is the current 29.4-second cut,
> recorded from the local demo fixtures with the refined Today, Streams, lesson,
> Review, Library, and Publishing experiences. Give the final render an editorial
> and claims review before external use.

A 1920×1080 Remotion launch-film project built around editorial typography,
real product motion, white-space-led composition, and dark kinetic interludes.

The film uses original Learnloom visuals and an original generated ambient
soundtrack. It does not reuse footage, branding, or audio from the reference.

## Story

1. Introduce Learnloom as a learning home.
2. Contrast endless reading with durable understanding.
3. Start with a technical question and inspect Learnloom's explainable source
   portfolio (or supply specific sources).
4. Weave those sources into a Learning Blueprint.
5. Reveal the finished Dossier.
6. Show continuity through Learning History.
7. Keep it private by default; optionally deliver by email or publish later.
8. Close with the product promise and launch CTA.

## Render

Requires Node.js, `sips` (included with macOS), and FFmpeg:

```sh
node launch-video/render.mjs
```

The renderer creates the legacy storyboard only. Its output is not launch
evidence until the scenes and captures are replaced and reviewed.

## Remotion cut

The editable launch-film composition now lives in `remotion/` and is the
preferred path for the revised cut:

```sh
cd launch-video/remotion
npm install
npm run studio
npm run render
npm run render:v2
npm run capture
npm run render:v3
npm run render:v4
```

It exports a 45-second `output/learnloom-launch-remotion.mp4`. The current cut
uses a continuous animated product workspace, with the captured product states
in `remotion/public/captures/` retained as visual references. Every scene stays
within the same white, warm-paper, black-type visual language.

The V1.1 audio experiment keeps the original film and visuals intact while
using a separate original electronic score with a faster 128 BPM pulse. It
exports to `output/learnloom-launch-remotion-v1.1.mp4`:

```sh
cd launch-video/remotion
npm run music:v1.1
npm run render:v1.1
```

The separate `LearnloomLaunchV2` composition is a 27-second cut with compressed
typing, a nearly double-speed continuous workspace, and a kinetic closing
statement. It exports `output/learnloom-launch-v2.mp4`.

## Current V3 cut

`LearnloomLaunchV3` is a 26.8-second, footage-led launch film. It combines fresh
1440×900 recordings of the local demo product with fast camera movement,
kinetic editorial typography, cursor beats, and six compact scenes. It does not
create or change production account data.

From `launch-video/remotion/`:

```sh
# Run `npm run demo -- --host 127.0.0.1 --port 4173` at the repository root first.
npm run capture
npm run studio -- --no-open
npm run render:v3
```

The capture script writes replaceable footage to `public/product-clips/`. The
render is exported to `launch-video/output/learnloom-launch-v3.mp4`.

## Current V4 cut

`LearnloomLaunchV4` is a 29.4-second interaction-led film organized around four
clear actions: choose, understand, retrieve, and keep. Product footage is
full-bleed and shows real navigation, retrieval, filtering, and publication
state changes without a persistent caption rail. A dedicated
`maya.learnloom.blog` reveal highlights the personal learning-home domain. It
uses the continuous music bed without typing or transition sound effects and
ends on a clean white brand card.

```sh
cd launch-video/remotion
npm run capture
npm run render:v4
```

The render is exported to `launch-video/output/learnloom-launch-v4.mp4`.

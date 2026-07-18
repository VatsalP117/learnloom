# Learnloom

Learnloom turns current source material and a learner's recent history into a
durable daily lesson, then delivers it through the learner's chosen channels.

## Language

**Daily Run**:
One idempotent attempt to generate and deliver a Dossier for a profile and local date.
_Avoid_: Job, execution, newsletter run

**Dossier**:
The canonical generated learning artifact containing a lesson, skeptical review,
retrieval practice, and cited sources.
_Avoid_: Newsletter, report, output

**Source Item**:
A normalized piece of reference material supplied to a Daily Run.
_Avoid_: Article, feed entry, document

**Learning History**:
The bounded record of previous Dossiers used to reduce repetition and build continuity.
_Avoid_: Memory, state

**Delivery Receipt**:
The durable outcome of attempting to send one Dossier through one configured destination.
_Avoid_: Send result, notification status

**Newsletter**:
A saved recurring learning stream containing a topic, Source Items, learner
preferences, a local schedule, a timezone, and active or paused state. Its
Issues produce Dossiers.
_Avoid_: Profile, feed, subscription

**Issue**:
The durable lifecycle record for one scheduled or manual Newsletter occurrence.
It can point to a Dossier and, after delivery is enabled, Delivery Receipts.
_Avoid_: Job, execution, newsletter run

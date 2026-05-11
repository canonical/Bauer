# Requirements and Sessifollowing the archtiectures and specs in the attached docs folder I want us to create a new spec 002_figma_integration.

we need to create a new architecture v2.1 doc as well and a spec in how to reach it, these need to build upon the result of the architecture v2
the spec and new architecture are mostly about how to integrate with Figma.

- the main idea: simply when a figma link is provided, we need to fetch the design somehow, and we need to supply it to our agents so they can consume it and check that the code actually aligns with the design.
- A challenge we face: how will we tie design bits to the textual data we have extracted from google docs? An assumption you can make and document is: we assume the core text data itself and headings will not really change.
- so should we use text matching to extract the figma screenshots and tie them to certain changed areas in the text? or how will the data coming from our current flows be able to be tied with and connected to the data extracted from figma? this is a really important topic.
- There are also going to be comments on figma that we need to tie to the data. not sure how they are extracted, but they need to be, and we need to tie them to the data same as the previous point.
- you need to performa a full and comprehensive research - and give me a summary of your findings of course - of how can we integrate figma into this? is it through API or MCP? how will we do local development with the CLI and how will it look like in prod? I am not opposed to different setups but you need to present a strong case.
- For whatever option you choose you will also need to provide docs links and articles that backup your plan and decisions and general explorations even, do not simply depend on your memory and try to fetch some of the figma blog/docs and see if there have been any recent developments in this regard.
- your target audience - me - is a developer and has no idea how figma actually works, is it layers? do you need the link to the full figma design or jus the area marked ready as dev? (if this even has its own link)... etc. so your spec needs to give a full succinct background about how figma works and how this will affect our interaction with it, tldr we need a figma cheat sheet for devs inside the spec.
- in the mode where we open an issue instead of implementing the changes by the agent, would it also be possible to add the figma screenshots there? (of course the textual data after it is connected and processed will be included in the body anyway - this should be part of your plan anyway - my question here is about how to also include the visual assets which you should analyze and consider)
- for the API - and to a lesser extent the CLI bcs it is not run centrally really - we do not have any real mechanism of maintaining a history of the data we extract and the outputs coming from it, we simply override the outputs and prompt files on every run. We need to address this. Perhaps a folder that has timestamped dirs that contain the generated files and extracted data from docs and figma and potentially other future sources? analyze this and come up with a solution and a proposal, also include it in the spec.|
- in the spec, and especially the planning and task breakdown part, you should include and consider that this new spec could start its implementation after phase 2 only of the 001 spec, i.e. it will start with the CLI only, then be added to the API later as it also gets ready. You should take this into consideration in your task break down, ordering... etc. and document it.
- other than the points I have just mentioned there could potentially be a lot of other aspects I am missing, so you should critically consider that and think of any missing aspects as well and include them.
- keep the language simple and succinct, prefer lists to paragraphs. Use the current architecture and spec files as guidelines , but do not be afraid to improve upon them. You may use mermaid diagram if needed.
- in the case that this current spec definitely and absolutely affects part of the old spec and plan, then mention it and clearly mark it in the new spec only, and present me with the argument of why we should alter the old spec - which is not implemented yet - and I will make the decision of whether we should alter the 001 spec before its implementation.
  the spec should have a table of contents and executive overview to aid with readabilityon Log

_Last updated: 2026-04-30_

This file records the requirements requested in this documentation sprint and tracks what has been accomplished.

---

## Requirements

### Initial request

```

```

Supporting asks:

- Add an info banner to `README.md` pointing readers to `docs/specs/` while v2/v2.1 work is in progress.
- Update `docs/specs/001_v2_reconciliation.md` with accepted changes: source abstraction layer (`internal/source`) and append-only artifact history (`internal/artifacts`) as prerequisite tasks (T0.2a, T0.2b, T0.2c); Problems 11 & 12; updated roadmap and task overview.
- Update `docs/ARCHITECTURE_v2.md` to reflect the changes introduced by 001 (add `internal/source` and `internal/artifacts` to the system diagram, packages table, and design decisions).

---

### Revision 1 — direction correction

> - I need you to compare the MCP vs AI better. Present an overview comparison table, then dive deeped into each, presenting a choice a the end. Your current comparison is not very useful. You could even add some code or pseudocode samples or diagrams to aid with this. You also mentioend that they can be used together so how would that also look like? The spec should mention all of this in details, the comparison overview is the only one included in the archtiecture.
> - in this comparison you should give me examples of code, and the related docs link for each individual API endpoint or MCP spec bit by bit together and not all code/discussion, then all links as an appendix to the doc.
> - you should also start by giving an executive overview of what funcitonalities the API and MCP respectively offer each on their own, its something I actuall do not know already.
> - a really important part that is very underspecced is the congregation and tying of data coming from gdocs and figma. I need you to go into much more details. Present sample outputs from the current gdocs pkg, what it looks like eventually (you should probably use the types for this) and present the figma API and/or MCP responses, and how are we going to present this data and tie it in order to say "for this block of copy updates in table id t1, use the screenshots 1, 2, & 3" this preprocessing part is realllllly important. Give an overview of it first and detailed explanation, then also include it in the tasks.

> - regarding the changes proposed to the 001 spec:
>   -- Proposed change 1: accepted. You need to add an abstraction layer that calls sources and combines their data into a proper output. This abstraction layer should then feed the data into the prompt pkg. But a consideration to this is that the prompt needs to know exactly which data it will receive, so in 002 you need to add a task to update the prompt pkg to handle figma data as well.
>   -- proposed change 2: accepted. add it to 001 and add the reasoning. Also do you think a DB would be needed for this? or is file system enough? so file system for all prompts, extracted data and images OR DB (contains extracted data and full prompts) + filesystem for images? For now spec the simplest solution, and add a section in the main spec part to discuss future imporvements to this section in particular (so under an artifact history title)
> - you may aftwerwards remove the "requested changes to 001" section from the 002 file since its no longer relevant

> - if you look at 001 the task breakdown is in 3 parts: roadmap that has phases, task overview: very high level description of each task, and task implementation details breakdown which includes detailed description, task break down, code samples and acceptance criteria for each task. You should follow this outline and breakdown details.
> - if you even want to include what the file structure needs to look like at certain points then do so

> - the spec should include a mini section about how to set up the dev setup to actually generate tokens, download any tools.. etc. so local dev prepration basically

> - in the #file:README.md add an info banner at the very top right after the main title, to direct people to the docs/specs folder, this should also mention the current readme might be out of date

> - you have access to the entire code base, if you feel that you need to read bits of code to get a better understanding of the system then do so.

> - you have the freedom of making the 002 spec as detailed as possible, use this wisely. Simplicity and conciseness does not mean to skimp on details.

> - at the end of 002 add a `unified implementation and task breakdown plan` that combines all the tasks of 001 and 002(that you are yet to generate fully) that follows the path of fixing CLI in 001 -> full CLI+figma integration from 002 -> API setup in 001 & 002. This is not a replacement to the standalone task breakdown section of 002 on its own.

---

### Revision 2 — comprehensive feedback on 002 and architecture docs

This revision was sent twice due to context summarisation. Verbatim content:

> - chunks should not be removed from the prompt, these are still a fact, in fact they are even more important with the figma integration so as not to overwhelm the agents with too much context. readd and document this decision.
> - it is fine if the prompt pkg is tied to both the special gdocs and figma pkgs. do not try to abstract it, or the way it consumes the sources. it only needs to be aware WHAT sources it is consuming. as a minimum it should always assume its consuming gdocs data, if it gets also figma data, then it should know so as to add special figma related prompts. documen this.
> - the changes added in 001 should be reflected also in the architecture v2.0 doc
>
> you have not added any of the requested changes to 002, address my comments and add them there and to the v2.1 arch docsuments, I will add them again:
>
> - I need you to compare the MCP vs AI better. Present an overview comparison table, then dive deeped into each, presenting a choice a the end. Your current comparison is not very useful. You could even add some code or pseudocode samples or diagrams to aid with this. You also mentioend that they can be used together so how would that also look like? The spec should mention all of this in details, the comparison overview is the only one included in the archtiecture.
> - in this comparison you should give me examples of code, and the related docs link for each individual API endpoint or MCP spec bit by bit together and not all code/discussion, then all links in the same sections.
> - a really important part that is very very very underspecced is the congregation and tying of data coming from gdocs and figma. I need you to go into much more details. Present sample outputs from the current gdocs pkg, what it looks like eventually (you should probably use the types for this) and present the figma API and/or MCP responses, and how are we going to present this data and tie it in order to say "for this block of copy updates in table id t1, use the screenshots 1, 2, & 3" this preprocessing part is realllllly important. Give an overview of it first and detailed explanation, then also include it in the tasks.
> - if you look at 001 the task breakdown is in 3 parts: roadmap that has phases, task overview: very high level description of each task, and task implementation details breakdown which includes detailed description, task break down, code samples and acceptance criteria for each task. You should follow this outline and breakdown details.
> - if you even want to include what the file structure needs to look like at certain points then do so
> - the spec should include a mini section about how to set up the dev setup to actually generate tokens, download any tools.. etc. so local dev prepration basically. Am i also correct in understanding the MCP needs a figma client? disucss this, and if this is the reason why the MCP cannot work effectively with the API
> - you have the freedom of making the 002 spec as detailed as possible, use this wisely. Simplicity and conciseness does not mean to skimp on details.
> - at the end of 002 add a `unified implementation and task breakdown plan` that combines all the tasks of 001 and 002(that you are yet to generate fully) that follows the path of fixing CLI in 001 -> full CLI+figma integration from 002 -> API setup in 001 & 002. This is not a replacement to the standalone task breakdown section of 002 on its own.

---

## Action Items: Accomplished

| #   | File                                  | What was done                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| --- | ------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `README.md`                           | Added info banner after `# Bauer` pointing readers to `docs/specs/` while v2/v2.1 work is in progress                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| 2   | `docs/specs/001_v2_reconciliation.md` | Added Problems 11 & 12 (no source abstraction, no append-only artifact history); added T0.2a, T0.2b, T0.2c tasks (source interfaces, prompt bundle refactor, artifact history foundation); updated roadmap and task overview table                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| 3   | `docs/ARCHITECTURE_v2.md`             | Added `internal/source` and `internal/artifacts` to the system diagram, packages table, and design decisions section                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| 4   | `docs/ARCHITECTURE_v2_1.md`           | Created from scratch: executive overview, REST vs MCP comparison table, full architecture diagram, source data model, mapping model, comments, screenshots, artifact history section                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| 5   | `docs/specs/002_figma_integration.md` | Created from scratch (full rewrite after delete), incorporating all revision 2 feedback: local dev setup, REST API deep dive with per-endpoint code + docs links inline, MCP deep dive with client requirement explanation and diagrams, REST vs MCP vs Hybrid comparison table, using REST and MCP together (sequence diagrams), chunking rationale preserved and documented, prompt package design (direct gdocs+figma coupling, no abstraction, `FigmaContextJSON` field), preprocessing overview and detailed congregation/tying model with sample JSON, mapping resolver algorithm, comment extraction and association, screenshot pipeline, artifact history and run directory layout, 001-style task structure (roadmap → task overview → per-task implementation with rationale/files/pseudocode/acceptance criteria), file structure examples at milestones, unified implementation plan combining 001+002 tasks in phased delivery order |
| 6   | `docs/ARCHITECTURE_v2_1.md`           | Updated architecture principles: expanded principles 3 and 4, added principle 7 (prompt package intentionally coupled to gdocs+figma, not source-agnostic) and principle 8 (chunking preserved and more important with Figma)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |

## Action Items: Pending

| #   | Description                                   |
| --- | --------------------------------------------- |
| —   | No outstanding items from the requests above. |

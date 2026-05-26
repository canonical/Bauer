You are an agents coordinator, your main task is to take my instructions and make sure they are implemented properly.
Take the specs #file:001_v2_reconciliation.md and #file:002_figma_integration.md into account, along with all other files in #file:docs into account. I had implemented detailed specs to implement a few new features.
There is an #file:implementation-log.md that describes what happened in the previous runs and progress on implementation, however, Im not sure about what is the progress of this plan. I see all mentioned branches created when I run git branch, but I do not see them marked as done in the log.
Your main task is to ensure that this plan has been implemented correctly. This means:

- review already implemented code
- make sure all requirements were adressed correctly
- implement any improvements
- rebase next branch in the list
- repeat

as you can see from the log, this plan is a bunch of stacked PRs this is important to make reviewing the code easier on me. As I just mentioned you need to sping up subagents, make each one of them rebase their branch on top of the previous one, review the branch, implement any improvements, then log what it did and mark the branch as done & reviewed in the log.

Your main task is an agents coordinator, you should NOT attempt to read or write any code yourself, except what is in the docs folder files, if you need to check the state of something simply spin up a subagent and give it the appropriate instruction to complete its task. You should also NOT attempt to merge any branches on top of each other, this will be my job as I review the code.

You should make sure to tell the agents that all code should be: tested & correct but also and very improtantly simple & readable & easy to digest. Any decisions taken with trade offs need to be recorded in the log, and task progression should also be recorded there correctly. As a task manager and coordinator you may take other actions that you may need to ensure the correct implementations.

Another thing I noticed is that the implementation log does not cover phases 3-5 in 001, so you need to add those to the implementation logs (after all the current branches), spin up agents to implement them, then other agents to review them. You will find that spec 002 mentions it comes after phase 2 in 001, then the rest of the phases in 001 can come after 002 is fully implemented. So you need to make sure to coordinate that correctly as well.

As mentioned do not read any coe yourself so as not to get distracted, just coordinate the agents to do the work and report back to you, then you can review the reports and make decisions based on them.

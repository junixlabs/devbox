# svc-workflow-engine

Complex services benefit from separation by responsibility.

**Structure:**
```
services/workflow-engine/
├── index.ts              # Factory composing all modules
├── types.ts              # All interfaces
├── constants.ts          # UIDs, default workflows
├── approver-resolver.ts  # Approver resolution logic
├── workflow-factory.ts   # Workflow retrieval/creation
├── approval-transitions.ts # State transitions
└── approval-queries.ts   # Read operations
```

**Factory pattern:**
```typescript
// index.ts
import { resolveApprovers } from './approver-resolver';
import { getWorkflow } from './workflow-factory';
import { submit, approve, reject } from './approval-transitions';

export function createWorkflowEngine(strapi: Strapi.Strapi) {
  return {
    resolveApprovers: resolveApprovers(strapi),
    getWorkflow: getWorkflow(strapi),
    submit: submit(strapi),
    approve: approve(strapi),
    reject: reject(strapi),
  };
}
```

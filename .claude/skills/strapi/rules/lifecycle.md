# Lifecycle Hooks

## life-register

Register hooks in `src/bootstrap.ts`.

```typescript
// src/bootstrap.ts
export default async ({ strapi }) => {
  strapi.db.lifecycles.subscribe({
    models: ['api::video.video'],

    async afterCreate(event) {
      const { result } = event;
      console.log('Video created:', result.documentId);
    }
  });
};
```

## life-events

Available lifecycle events.

```typescript
strapi.db.lifecycles.subscribe({
  models: ['api::video.video'],

  // Before operations
  async beforeCreate(event) {
    const { params } = event;
    // Modify params.data before create
    params.data.slug = slugify(params.data.title);
  },

  async beforeUpdate(event) {
    const { params } = event;
    // Access: params.data, params.where
  },

  async beforeDelete(event) {
    const { params } = event;
    // Cleanup before delete
  },

  // After operations
  async afterCreate(event) {
    const { result } = event;
    // result contains created entity
  },

  async afterUpdate(event) {
    const { result, params } = event;
    // Check what changed
    if (params.data?.status === 'published') {
      await notifySubscribers(result);
    }
  },

  async afterDelete(event) {
    const { result } = event;
    // Cleanup related resources
  },

  // Query hooks
  async beforeFindOne(event) {},
  async afterFindOne(event) {},
  async beforeFindMany(event) {},
  async afterFindMany(event) {}
});
```

## life-async

Handle async operations properly.

```typescript
async afterCreate(event) {
  const { result } = event;

  // Fire and forget (non-blocking)
  sendNotification(result).catch(console.error);

  // Or await if needed
  try {
    await createRelatedResource(result);
  } catch (error) {
    strapi.log.error('Failed to create related resource', error);
  }
}
```

## life-multiple

Subscribe to multiple models.

```typescript
strapi.db.lifecycles.subscribe({
  models: ['api::video.video', 'api::project.project'],

  async afterCreate(event) {
    const { model, result } = event;

    if (model.uid === 'api::video.video') {
      // Video-specific logic
      await updateProjectVideoCount(result.project);
    }

    if (model.uid === 'api::project.project') {
      // Project-specific logic
      await createDefaultWorkflow(result);
    }
  }
});
```

## life-event-object

Event object structure.

```typescript
{
  action: 'create' | 'update' | 'delete',
  model: {
    uid: 'api::video.video',
    singularName: 'video',
    tableName: 'videos',
    attributes: { /* schema attributes */ }
  },
  params: {
    data: { /* input data */ },
    where: { /* filter conditions */ },
    populate: { /* populate config */ }
  },
  result: { /* created/updated entity */ },
  state: { /* custom state object */ }
}
```

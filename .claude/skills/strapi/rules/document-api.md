# Document API

## doc-custom

Use `strapi.documents()` for custom content types (`api::*`).

```typescript
// UID format: api::{content-type}.{content-type}
const videos = await strapi.documents('api::video.video').findMany({
  filters: { status: 'published' },
  populate: ['project', 'tags']
});
```

## doc-plugin

Use `strapi.db.query()` for plugin content types (`plugin::*`).

```typescript
// For users-permissions plugin
const user = await strapi.db.query('plugin::users-permissions.user').findOne({
  where: { id: userId },
  populate: ['role', 'team']
});
```

## doc-crud

Standard CRUD operations.

```typescript
// Find many
const items = await strapi.documents('api::item.item').findMany({
  filters: { active: true },
  populate: '*',
  sort: { createdAt: 'desc' },
  limit: 10,
  offset: 0
});

// Find one
const item = await strapi.documents('api::item.item').findOne({
  documentId: id,
  populate: ['category', 'tags']
});

// Create
const created = await strapi.documents('api::item.item').create({
  data: {
    title: 'New Item',
    status: 'draft'
  }
});

// Update
const updated = await strapi.documents('api::item.item').update({
  documentId: id,
  data: { title: 'Updated Title' }
});

// Delete
await strapi.documents('api::item.item').delete({
  documentId: id
});
```

## doc-populate

Populate relations correctly.

```typescript
// Populate all
populate: '*'

// Populate specific
populate: ['category', 'author']

// Populate with fields
populate: {
  category: {
    fields: ['name', 'slug']
  },
  author: {
    fields: ['username', 'email'],
    populate: ['avatar']
  }
}

// Deep populate
populate: {
  project: {
    populate: {
      team: {
        fields: ['name']
      }
    }
  }
}
```

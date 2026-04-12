# Security

## Authentication

- [Auth method: JWT/Session]
- Token expiry: [duration]
- Password requirements: [requirements]

---

## Authorization

### Roles

| Role | Description |
|------|-------------|
| Admin | Full access |
| User | Standard access |
| Guest | Limited access |

### Permission Matrix

| Role | Read | Write | Delete | Admin |
|------|------|-------|--------|-------|
| Admin | ✓ | ✓ | ✓ | ✓ |
| User | ✓ | ✓ | ✗ | ✗ |
| Guest | ✓ | ✗ | ✗ | ✗ |

---

## Authorization Flow

```
Request → Auth Middleware → Permission Check → Controller
```

---

## Data Protection

- [ ] HTTPS enforced
- [ ] Password hashing
- [ ] Input validation
- [ ] SQL injection prevention
- [ ] XSS prevention
- [ ] CSRF protection
- [ ] Rate limiting

---

## Policies

```typescript
// Example policy
export default async (ctx, config, { strapi }) => {
  const { user } = ctx.state;
  if (!user) return false;
  return true;
};
```

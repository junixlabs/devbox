# Design System

**[TODO: Theme description]**

## 1. Color Palette

| Role | Tailwind | Hex | Usage |
|------|----------|-----|-------|
| **Primary** | `[TODO]` | `#[TODO]` | Primary buttons, links |
| **Accent** | `[TODO]` | `#[TODO]` | Secondary actions |
| **Surface** | `white` | `#ffffff` | Background, cards |
| **Text Primary** | `[TODO]` | `#[TODO]` | Headings, body |
| **Text Secondary** | `[TODO]` | `#[TODO]` | Descriptions |
| **Border** | `[TODO]` | `#[TODO]` | Dividers |
| **Success** | `[TODO]` | `#[TODO]` | Success states |
| **Error** | `[TODO]` | `#[TODO]` | Error states |
| **Warning** | `[TODO]` | `#[TODO]` | Warning states |

---

## 2. Typography

**Font Family:** `[TODO]`, sans-serif

| Style | Weight | Size | Tailwind |
|-------|--------|------|----------|
| H1 | Bold (700) | 30px | `text-3xl font-bold` |
| H2 | Bold (700) | 24px | `text-2xl font-bold` |
| H3 | SemiBold (600) | 20px | `text-xl font-semibold` |
| Body | Regular (400) | 16px | `text-base` |
| Small | Regular (400) | 14px | `text-sm` |

---

## 3. Components

### Buttons

| Variant | Background | Text | Usage |
|---------|------------|------|-------|
| Primary | `bg-[TODO]` | `text-white` | Main actions |
| Secondary | `bg-[TODO]` | `text-white` | Secondary actions |
| Outline | `border-[TODO]` | `text-[TODO]` | Tertiary actions |
| Ghost | transparent | `text-[TODO]` | Cancel, subtle |
| Danger | `bg-red-500` | `text-white` | Destructive |

### Cards

| Variant | Style |
|---------|-------|
| Elevated | `bg-white rounded-[TODO] shadow-[TODO]` |
| Outlined | `bg-white rounded-[TODO] border` |

### Inputs

| State | Style |
|-------|-------|
| Default | `bg-[TODO] border-[TODO] rounded-[TODO]` |
| Focus | `border-[TODO] ring-2` |
| Error | `border-red-500 ring-red-100` |

---

## 4. Layout & Spacing

| Element | Spacing |
|---------|---------|
| Page padding | `p-6` |
| Card padding | `p-6` |
| Section gap | `space-y-6` |
| Form field gap | `space-y-4` |

---

## 5. Border Radius

| Component | Radius |
|-----------|--------|
| Buttons | `rounded-[TODO]` |
| Cards | `rounded-[TODO]` |
| Inputs | `rounded-[TODO]` |

---

## 6. Component Import

```tsx
import {
  Button,
  Card,
  Input,
  Modal,
  Table,
  Badge,
  // [TODO: Add components]
} from '@/components/ui';
```

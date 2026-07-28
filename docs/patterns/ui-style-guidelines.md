# UI style guidelines (summary)

Canonical long-form rules live in [go-mumble-server ui-style-guidelines](https://github.com/dchote/go-mumble-server/blob/main/docs/patterns/ui-style-guidelines.md). talkiepi uses the same Material Blue / MD3 look via `frontend/src/styles/theme.scss`.

## Hard rules (Vuetify)

| Rule | Do | Don’t |
|------|----|-------|
| Button spacing | `class="mr-2"` on the left button | Adjacent buttons with no margin; **`gap` / `gap-*`** |
| Form fields | `variant="outlined"` `density="compact"` `hide-details="auto"` | Comfortable density (except rare a11y cases) |
| Field spacing | Wrapper `mb-4`; actions `mt-4` + `justify-end` | Crowded stacks |
| Autocomplete | `autocomplete="off"` on non-auth fields | Missing autocomplete |
| Feedback | `v-alert` density compact | `alert()` / `confirm()` |
| Cards | `StandardCard`; flat brand header | Random nested card chrome in heroes |

## Headers

- `d-flex align-center` on header rows
- Primary actions right-aligned after `v-spacer`

## Typography and color

- System font stack (no decorative webfonts)
- Card titles: `text-h5` / `text-h6`, weight 600
- Primary `#1976D2`; theme-driven surfaces

## Tables

- Primary lists: `density="comfortable"` (or compact where space is tight on status tables)
- Prefer empty `#bottom` on `v-data-table` when pagination is unused

## Mobile

- Page padding via `v-container`; responsive `pa-3 pa-sm-*` when needed
- Touch targets ≥ 44px; fullscreen dialogs on small screens when used

## Cursor rules

Repo `.cursor/rules/` encode the same constraints (`button-spacing`, `form-inputs`, `margins-and-spacing`, `no-alert-confirm`, `list-toolbar`). Follow them when editing Vue SFCs.

# Frontend guide

talkiepi’s management UI follows the **same Vue 3 + Vuetify patterns** as [go-mumble-server](https://github.com/dchote/go-mumble-server/blob/main/docs/patterns/frontend-guide.md). Prefer that document for the full rule set; this page notes talkiepi layout and API differences.

## Shared with go-mumble-server

- Vue 3, Vuetify 3, Vite, `unplugin-vue-router` (`importMode: 'sync'` for embed)
- `StandardCard` for page chrome (gradient header, top border)
- Form inputs: `variant="outlined"`, `density="compact"`, `hide-details="auto"`, `autocomplete="off"`
- **Never use `gap`** — use `mr-2` / `mb-4` / `mt-4`
- No `alert()` / `confirm()` — use `v-alert` and dialogs
- Layout slot: no padding around the slot; pages use `v-container`

See also [ui-style-guidelines.md](ui-style-guidelines.md).

## talkiepi directory layout

```
frontend/src/
├── layouts/DefaultLayout.vue
├── pages/          # index, server, audio, channels, chat
├── components/common/StandardCard.vue
├── utils/api.js    # REST client (no JWT)
├── utils/ingress.js
├── plugins/
├── router/
├── styles/theme.scss
└── App.vue
```

## Auth model

Unlike the server admin UI, talkiepi has **no login**. The API is unauthenticated; Home Assistant ingress or LAN trust is assumed. There is no Vuex auth module or Authorization header.

## API client

Centralize calls in `utils/api.js` against `/api/v1/*` (and `/health`). Under HA ingress, use the ingress path helpers in `utils/ingress.js` so fetches stay same-origin.

## Pages (jobs)

| Page | Job |
|------|-----|
| Channels (`index`) | Channel tree + chat; primary landing page |
| Status | Connection state, connect/disconnect, PTT / Communicator |
| Server | Server + voice mode settings |
| Audio | Devices, volume, processing |
| Updates | Version check / release notes |

Keep pages thin; put shared chrome in `StandardCard`.

## Build / embed

```bash
cd frontend && yarn install --frozen-lockfile && yarn build
rm -rf cmd/talkiepi/frontend-dist && cp -r frontend/dist cmd/talkiepi/frontend-dist
```

Go release builds use `-tags embed_frontend`. Dev: Vite + `make run` with `-frontend-embed=false`.

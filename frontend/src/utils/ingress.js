// Ingress-compatible base path: under HA ingress we are at /api/hassio_ingress/<token>/...
// Returns the path prefix (e.g. '/api/hassio_ingress/xyz') or '' when not under ingress.
export function getIngressBase() {
  const path = window.location.pathname
  const segments = path.split('/').filter(Boolean)
  if (segments[0] === 'api' && segments[1] === 'hassio_ingress' && segments[2]) {
    return '/' + segments.slice(0, 3).join('/')
  }
  return ''
}

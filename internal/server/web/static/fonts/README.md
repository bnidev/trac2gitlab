# Fonts (bundled, no CDN)

This dashboard uses system font stacks so it works offline:

- UI: `Inter, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif`
- Display: `Outfit, Inter, system-ui, sans-serif` (falls back cleanly)
- Mono: `Geist Mono, ui-monospace, SFMono-Regular, Menlo, monospace`

To use the exact display/mono faces, drop licensed `woff2` files here:

```text
web/static/fonts/Outfit-600.woff2
web/static/fonts/Inter-400.woff2
web/static/fonts/GeistMono-400.woff2
```

and add matching `@font-face` rules at the top of `app.css`.
No remote font links: everything must load from `/static/`.

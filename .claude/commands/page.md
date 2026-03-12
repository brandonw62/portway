Create a new frontend page in the Portway web app.

1. Parse $ARGUMENTS for the page name and route path
2. Create the page component in `web/src/pages/<PageName>.tsx`
3. Follow the patterns established in existing pages (e.g., `CatalogPage.tsx`, `ResourcesPage.tsx`):
   - Use the API client from `web/src/api.ts` for data fetching
   - TypeScript strict — no `any` types
4. Add the route to `web/src/App.tsx` inside the `<Route element={<Layout />}>` block
5. Add a navigation link in `web/src/Layout.tsx` sidebar if appropriate
6. Verify with `cd web && npx tsc --noEmit` that there are no type errors

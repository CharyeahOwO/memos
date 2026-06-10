# Muling custom recovery, 2026-02

This branch reconstructs the public clues from the Muling custom Memos build described in:

- Blog: https://mulingowo.cn/2026/02/memos-%e8%87%aa%e5%ae%9a%e4%b9%89%e7%89%88%e6%9c%ac%ef%bc%9aexplore-%e9%a1%b5%e9%9d%a2%e5%a2%9e%e5%bc%ba%e4%b8%8e%e6%97%a5%e5%8e%86%e4%bf%ae%e5%a4%8d/
- Upstream PR: https://github.com/usememos/memos/pull/5605

Recovery baseline:

- Base commit: `b623162d37f87f9f174d8f6cd8e54c7034cfc789`
- Recovery branch: `recover-muling-custom-202602`
- PR head used for the calendar navigation fix: `2366e2f156dae132b106f9cc3a055f0a0a7d915b`

Recovered behavior:

- Explore calendar date clicks stay on the current route instead of navigating to the root route.
- Explore memo listing enables pinned memos and sorts pinned memos first.
- Explore memo cards no longer render the creator avatar/name block.
- Explore sidebar statistics use `listAllUserStats` through `useAllUserStats`, so the activity calendar is not limited by the current memo list page.

Touched files:

- `web/src/hooks/useDateFilterNavigation.ts`
- `web/src/pages/Explore.tsx`
- `web/src/hooks/useMemoQueries.ts`
- `web/src/hooks/useUserQueries.ts`
- `web/src/hooks/useFilteredMemoStats.ts`
- `web/src/layouts/MainLayout.tsx`

Docker archaeology:

- `docker pull charyeahowo/memos:custom` and `docker pull charyeahowo/memos:latest` could not be run locally because the `docker` CLI is not installed in this environment.
- Docker Hub tag metadata is still public:
  - `charyeahowo/memos:custom`, last updated `2026-02-09T10:55:33.444374Z`
  - `charyeahowo/memos:latest`, last updated `2026-02-09T10:55:37.797483Z`

Deployment boundary:

- This recovery does not connect to, restart, recreate, or update any production Memos, 1Panel, or Docker service.
- To build an image later in a safe local or CI environment, use the project Dockerfile from this branch and push/deploy separately from production operations.

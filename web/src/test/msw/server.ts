import { setupServer } from "msw/node";

/**
 * MSW server for component/data-hook tests (FRONTEND_PLAN §8). Empty by default;
 * each test declares the handlers it needs via `server.use(...)`. Started/reset
 * in the global test setup.
 */
export const server = setupServer();

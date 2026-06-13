import { installFetchMock, resetFetchMock } from "./fetch-mock";

// expo-secure-store is a native module; back it with an in-memory map in tests.
jest.mock("expo-secure-store", () => {
  const store = new Map<string, string>();
  return {
    getItemAsync: jest.fn(async (k: string) => store.get(k) ?? null),
    setItemAsync: jest.fn(async (k: string, v: string) => {
      store.set(k, v);
    }),
    deleteItemAsync: jest.fn(async (k: string) => {
      store.delete(k);
    }),
  };
});

beforeAll(() => installFetchMock());
afterEach(() => resetFetchMock());

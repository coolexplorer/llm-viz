import '@testing-library/jest-dom';
import { vi } from 'vitest';

// @testing-library/dom's waitFor checks `typeof jest !== 'undefined'` to detect
// fake timers. Vitest doesn't provide `jest` as a global, so we alias it to `vi`
// so that waitFor can properly advance fake timers via jest.advanceTimersByTime().
(globalThis as unknown as Record<string, unknown>).jest = vi;

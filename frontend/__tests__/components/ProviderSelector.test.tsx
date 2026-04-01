import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ProviderSelector from '@/app/components/ProviderSelector';

// Mock localStorage factory
const createLocalStorageMock = () => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => { store[key] = value; },
    removeItem: (key: string) => { delete store[key]; },
    clear: () => { store = {}; },
  };
};

beforeEach(() => {
  const localStorageMock = createLocalStorageMock();
  vi.stubGlobal('localStorage', localStorageMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe('ProviderSelector', () => {
  it('renders Provider Settings heading', () => {
    render(<ProviderSelector onChange={vi.fn()} />);
    expect(screen.getByText('Provider Settings')).toBeInTheDocument();
  });

  it('renders provider, model, and API key fields', () => {
    render(<ProviderSelector onChange={vi.fn()} />);
    expect(screen.getByText('Provider')).toBeInTheDocument();
    expect(screen.getByText('Model')).toBeInTheDocument();
    // Multiple elements may match "API Key" - just check at least one exists
    expect(screen.getAllByText(/API Key/i).length).toBeGreaterThanOrEqual(1);
  });

  it('defaults to anthropic provider', () => {
    render(<ProviderSelector onChange={vi.fn()} />);
    const providerSelect = screen.getByDisplayValue('Anthropic');
    expect(providerSelect).toBeInTheDocument();
  });

  it('calls onChange on initial mount', async () => {
    const onChange = vi.fn();
    render(<ProviderSelector onChange={onChange} />);
    await waitFor(() => expect(onChange).toHaveBeenCalled());
  });

  it('changes provider to openai', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ProviderSelector onChange={onChange} />);
    // Provider is the first combobox
    const comboboxes = screen.getAllByRole('combobox');
    await user.selectOptions(comboboxes[0], 'openai');
    await waitFor(() => {
      const calls = onChange.mock.calls;
      const lastCall = calls[calls.length - 1][0];
      expect(lastCall.provider).toBe('openai');
    });
  });

  it('shows anthropic model options when anthropic is selected', () => {
    render(<ProviderSelector onChange={vi.fn()} />);
    expect(screen.getByDisplayValue('claude-sonnet-4-6')).toBeInTheDocument();
  });

  it('toggles API key visibility', async () => {
    const user = userEvent.setup();
    render(<ProviderSelector onChange={vi.fn()} />);
    const input = screen.getByPlaceholderText(/sk-ant-.../i);
    expect(input).toHaveAttribute('type', 'password');
    const toggleBtn = screen.getByRole('button', { name: /show key/i });
    await user.click(toggleBtn);
    expect(input).toHaveAttribute('type', 'text');
    await user.click(screen.getByRole('button', { name: /hide key/i }));
    expect(input).toHaveAttribute('type', 'password');
  });

  it('shows API key warning when key is empty', () => {
    render(<ProviderSelector onChange={vi.fn()} />);
    expect(screen.getByText(/Enter your API key/i)).toBeInTheDocument();
  });

  it('hides API key warning when key is entered', async () => {
    const user = userEvent.setup();
    render(<ProviderSelector onChange={vi.fn()} />);
    const input = screen.getByPlaceholderText(/sk-ant-.../i);
    await user.type(input, 'sk-ant-test-key');
    await waitFor(() => {
      expect(screen.queryByText(/Enter your API key/i)).not.toBeInTheDocument();
    });
  });

  it('includes apiKey in onChange call', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ProviderSelector onChange={onChange} />);
    const input = screen.getByPlaceholderText(/sk-ant-.../i);
    await user.type(input, 'my-api-key');
    await waitFor(() => {
      const calls = onChange.mock.calls;
      const lastCall = calls[calls.length - 1][0];
      expect(lastCall.apiKey).toBe('my-api-key');
    });
  });

  it('updates model in onChange when model is changed', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ProviderSelector onChange={onChange} />);
    // Model is the second combobox
    const comboboxes = screen.getAllByRole('combobox');
    // Anthropic default is claude-sonnet-4-6, switch to claude-haiku-4-5
    await user.selectOptions(comboboxes[1], 'claude-haiku-4-5');
    await waitFor(() => {
      const calls = onChange.mock.calls;
      const lastCall = calls[calls.length - 1][0];
      expect(lastCall.model).toBe('claude-haiku-4-5');
    });
  });
});

// ─── API Key Management (Backend-integrated) ─────────────────────────────────

const mockSavedKeys = [
  { id: 'key-1', provider: 'anthropic', maskedKey: 'sk-ant-***abc', label: 'sk-ant-***abc (Active)' },
  { id: 'key-2', provider: 'openai', maskedKey: 'sk-***xyz', label: 'sk-***xyz (Active)' },
];

function setupFetchMock(options: {
  getKeys?: object[];
  saveResponse?: object;
  saveError?: boolean;
  deleteResponse?: object;
  deleteError?: boolean;
} = {}) {
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET';

    if (method === 'GET' && String(url).includes('/api/keys')) {
      return {
        ok: true,
        json: async () => ({ keys: options.getKeys ?? [] }),
      };
    }

    if (method === 'POST' && String(url).includes('/api/keys')) {
      if (options.saveError) {
        return { ok: false, status: 400, json: async () => ({ error: 'Invalid API key format' }) };
      }
      return {
        ok: true,
        json: async () => options.saveResponse ?? { id: 'key-new', maskedKey: 'sk-ant-***new' },
      };
    }

    if (method === 'DELETE' && String(url).includes('/api/keys')) {
      if (options.deleteError) {
        return { ok: false, status: 500, json: async () => ({ error: 'Delete failed' }) };
      }
      return {
        ok: true,
        json: async () => options.deleteResponse ?? { success: true },
      };
    }

    throw new Error(`Unexpected fetch call: ${method} ${url}`);
  });

  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

describe('ProviderSelector - API Key Management', () => {
  // ── Section rendering ──────────────────────────────────────────────────────

  it('renders API Key Management section', async () => {
    setupFetchMock();
    render(<ProviderSelector onChange={vi.fn()} />);
    await waitFor(() => {
      expect(screen.getByText(/API Key Management/i)).toBeInTheDocument();
    });
  });

  it('renders a toggle button to expand/collapse the API key section', async () => {
    setupFetchMock();
    render(<ProviderSelector onChange={vi.fn()} />);
    await waitFor(() => {
      const toggleBtn = screen.getByRole('button', { name: /add api key|manage api keys|api key/i });
      expect(toggleBtn).toBeInTheDocument();
    });
  });

  it('API key input section is collapsed by default', async () => {
    setupFetchMock();
    render(<ProviderSelector onChange={vi.fn()} />);
    await waitFor(() => {
      // The save-key form input should NOT be visible initially
      expect(screen.queryByPlaceholderText(/paste your api key/i)).not.toBeInTheDocument();
    });
  });

  it('clicking the toggle button expands the API key input section', async () => {
    const user = userEvent.setup();
    setupFetchMock();
    render(<ProviderSelector onChange={vi.fn()} />);
    const toggleBtn = await screen.findByRole('button', { name: /add api key|manage api keys|api key/i });
    await user.click(toggleBtn);
    await waitFor(() => {
      expect(screen.getByPlaceholderText(/paste your api key/i)).toBeInTheDocument();
    });
  });

  it('clicking the toggle again collapses the API key input section', async () => {
    const user = userEvent.setup();
    setupFetchMock();
    render(<ProviderSelector onChange={vi.fn()} />);
    const toggleBtn = await screen.findByRole('button', { name: /add api key|manage api keys|api key/i });
    await user.click(toggleBtn); // open
    await user.click(toggleBtn); // close
    await waitFor(() => {
      expect(screen.queryByPlaceholderText(/paste your api key/i)).not.toBeInTheDocument();
    });
  });

  // ── Fetch existing keys on mount ───────────────────────────────────────────

  it('fetches saved keys from GET /api/keys on mount', async () => {
    const fetchMock = setupFetchMock({ getKeys: mockSavedKeys });
    render(<ProviderSelector onChange={vi.fn()} />);
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining('/api/keys'),
        expect.objectContaining({ method: 'GET' }),
      );
    });
  });

  it('displays saved keys as "sk-ant-***xyz (Active)" format', async () => {
    setupFetchMock({ getKeys: mockSavedKeys });
    render(<ProviderSelector onChange={vi.fn()} />);
    await waitFor(() => {
      expect(screen.getByText('sk-ant-***abc (Active)')).toBeInTheDocument();
    });
  });

  it('displays all saved keys for the current provider', async () => {
    setupFetchMock({ getKeys: mockSavedKeys });
    render(<ProviderSelector onChange={vi.fn()} />);
    await waitFor(() => {
      // anthropic key visible by default (anthropic is the default provider)
      expect(screen.getByText('sk-ant-***abc (Active)')).toBeInTheDocument();
    });
  });

  // ── Save key flow ──────────────────────────────────────────────────────────

  it('saves API key via POST /api/keys on "Save Key" button click', async () => {
    const user = userEvent.setup();
    const fetchMock = setupFetchMock({
      getKeys: [],
      saveResponse: { id: 'key-new', maskedKey: 'sk-ant-***new' },
    });
    render(<ProviderSelector onChange={vi.fn()} />);

    const toggleBtn = await screen.findByRole('button', { name: /add api key|manage api keys|api key/i });
    await user.click(toggleBtn);

    const input = screen.getByPlaceholderText(/paste your api key/i);
    await user.type(input, 'sk-ant-api123testkey');

    const saveBtn = screen.getByRole('button', { name: /save key|save/i });
    await user.click(saveBtn);

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining('/api/keys'),
        expect.objectContaining({
          method: 'POST',
          body: expect.stringContaining('sk-ant-api123testkey'),
        }),
      );
    });
  });

  it('displays newly saved masked key after successful save', async () => {
    const user = userEvent.setup();
    setupFetchMock({
      getKeys: [],
      saveResponse: { id: 'key-new', maskedKey: 'sk-ant-***new', label: 'sk-ant-***new (Active)' },
    });
    render(<ProviderSelector onChange={vi.fn()} />);

    const toggleBtn = await screen.findByRole('button', { name: /add api key|manage api keys|api key/i });
    await user.click(toggleBtn);

    const input = screen.getByPlaceholderText(/paste your api key/i);
    await user.type(input, 'sk-ant-api123testkey');

    await user.click(screen.getByRole('button', { name: /save key|save/i }));

    await waitFor(() => {
      expect(screen.getByText(/sk-ant-\*\*\*new/i)).toBeInTheDocument();
    });
  });

  it('clears the input field after successful save', async () => {
    const user = userEvent.setup();
    setupFetchMock({
      getKeys: [],
      saveResponse: { id: 'key-new', maskedKey: 'sk-ant-***new' },
    });
    render(<ProviderSelector onChange={vi.fn()} />);

    const toggleBtn = await screen.findByRole('button', { name: /add api key|manage api keys|api key/i });
    await user.click(toggleBtn);

    const input = screen.getByPlaceholderText(/paste your api key/i);
    await user.type(input, 'sk-ant-api123testkey');
    await user.click(screen.getByRole('button', { name: /save key|save/i }));

    await waitFor(() => {
      expect((input as HTMLInputElement).value).toBe('');
    });
  });

  it('disables Save Key button when input is empty', async () => {
    const user = userEvent.setup();
    setupFetchMock({ getKeys: [] });
    render(<ProviderSelector onChange={vi.fn()} />);

    const toggleBtn = await screen.findByRole('button', { name: /add api key|manage api keys|api key/i });
    await user.click(toggleBtn);

    const saveBtn = screen.getByRole('button', { name: /save key|save/i });
    expect(saveBtn).toBeDisabled();
  });

  // ── Delete key flow ────────────────────────────────────────────────────────

  it('renders a delete button for each saved key', async () => {
    setupFetchMock({ getKeys: mockSavedKeys });
    render(<ProviderSelector onChange={vi.fn()} />);
    await waitFor(() => {
      const deleteButtons = screen.getAllByRole('button', { name: /delete|remove/i });
      expect(deleteButtons.length).toBeGreaterThanOrEqual(1);
    });
  });

  it('sends DELETE /api/keys/:id when delete button is clicked', async () => {
    const user = userEvent.setup();
    const fetchMock = setupFetchMock({ getKeys: [mockSavedKeys[0]] });
    render(<ProviderSelector onChange={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('sk-ant-***abc (Active)')).toBeInTheDocument();
    });

    const deleteBtn = screen.getByRole('button', { name: /delete|remove/i });
    await user.click(deleteBtn);

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining('/api/keys/key-1'),
        expect.objectContaining({ method: 'DELETE' }),
      );
    });
  });

  it('removes deleted key from display after successful deletion', async () => {
    const user = userEvent.setup();
    setupFetchMock({ getKeys: [mockSavedKeys[0]], deleteResponse: { success: true } });
    render(<ProviderSelector onChange={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('sk-ant-***abc (Active)')).toBeInTheDocument();
    });

    const deleteBtn = screen.getByRole('button', { name: /delete|remove/i });
    await user.click(deleteBtn);

    await waitFor(() => {
      expect(screen.queryByText('sk-ant-***abc (Active)')).not.toBeInTheDocument();
    });
  });

  // ── Error handling ─────────────────────────────────────────────────────────

  it('shows error message when save request returns an error', async () => {
    const user = userEvent.setup();
    setupFetchMock({ getKeys: [], saveError: true });
    render(<ProviderSelector onChange={vi.fn()} />);

    const toggleBtn = await screen.findByRole('button', { name: /add api key|manage api keys|api key/i });
    await user.click(toggleBtn);

    const input = screen.getByPlaceholderText(/paste your api key/i);
    await user.type(input, 'bad-key');
    await user.click(screen.getByRole('button', { name: /save key|save/i }));

    await waitFor(() => {
      expect(screen.getByText(/invalid api key format|error/i)).toBeInTheDocument();
    });
  });

  it('shows error message when network fails during save', async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
      const method = init?.method ?? 'GET';
      if (method === 'GET') return { ok: true, json: async () => ({ keys: [] }) };
      throw new TypeError('Failed to fetch');
    });
    vi.stubGlobal('fetch', fetchMock);

    render(<ProviderSelector onChange={vi.fn()} />);

    const toggleBtn = await screen.findByRole('button', { name: /add api key|manage api keys|api key/i });
    await user.click(toggleBtn);

    const input = screen.getByPlaceholderText(/paste your api key/i);
    await user.type(input, 'sk-ant-api123testkey');
    await user.click(screen.getByRole('button', { name: /save key|save/i }));

    await waitFor(() => {
      expect(screen.getByText(/failed to save|network error|error/i)).toBeInTheDocument();
    });
  });

  it('shows error message when delete request fails', async () => {
    const user = userEvent.setup();
    setupFetchMock({ getKeys: [mockSavedKeys[0]], deleteError: true });
    render(<ProviderSelector onChange={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('sk-ant-***abc (Active)')).toBeInTheDocument();
    });

    const deleteBtn = screen.getByRole('button', { name: /delete|remove/i });
    await user.click(deleteBtn);

    await waitFor(() => {
      expect(screen.getByText(/delete failed|failed to delete|error/i)).toBeInTheDocument();
    });
  });

  it('does not remove key from UI when delete request fails', async () => {
    const user = userEvent.setup();
    setupFetchMock({ getKeys: [mockSavedKeys[0]], deleteError: true });
    render(<ProviderSelector onChange={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('sk-ant-***abc (Active)')).toBeInTheDocument();
    });

    const deleteBtn = screen.getByRole('button', { name: /delete|remove/i });
    await user.click(deleteBtn);

    await waitFor(() => {
      // Key should still be displayed after a failed delete
      expect(screen.getByText('sk-ant-***abc (Active)')).toBeInTheDocument();
    });
  });

  // ── Security ───────────────────────────────────────────────────────────────

  it('API key input is masked (type=password) by default', async () => {
    const user = userEvent.setup();
    setupFetchMock({ getKeys: [] });
    render(<ProviderSelector onChange={vi.fn()} />);

    const toggleBtn = await screen.findByRole('button', { name: /add api key|manage api keys|api key/i });
    await user.click(toggleBtn);

    const input = screen.getByPlaceholderText(/paste your api key/i);
    expect(input).toHaveAttribute('type', 'password');
  });

  it('full API key value is not exposed in the DOM after saving', async () => {
    const user = userEvent.setup();
    const rawKey = 'sk-ant-api123secretkey';
    setupFetchMock({
      getKeys: [],
      saveResponse: { id: 'key-new', maskedKey: 'sk-ant-***key', label: 'sk-ant-***key (Active)' },
    });
    render(<ProviderSelector onChange={vi.fn()} />);

    const toggleBtn = await screen.findByRole('button', { name: /add api key|manage api keys|api key/i });
    await user.click(toggleBtn);

    const input = screen.getByPlaceholderText(/paste your api key/i);
    await user.type(input, rawKey);
    await user.click(screen.getByRole('button', { name: /save key|save/i }));

    await waitFor(() => {
      // Full key should not be anywhere in the DOM as text content
      expect(screen.queryByText(rawKey)).not.toBeInTheDocument();
    });

    // Input field should be cleared (no longer contains the raw key value)
    await waitFor(() => {
      expect((input as HTMLInputElement).value).toBe('');
    });
  });

  it('only shows masked version of saved keys (not plaintext)', async () => {
    const fullKey = 'sk-ant-api123secretkey';
    setupFetchMock({
      getKeys: [{ id: 'key-1', provider: 'anthropic', maskedKey: 'sk-ant-***key', label: 'sk-ant-***key (Active)' }],
    });
    render(<ProviderSelector onChange={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('sk-ant-***key (Active)')).toBeInTheDocument();
    });

    // The full plaintext key should never appear
    expect(screen.queryByText(fullKey)).not.toBeInTheDocument();
  });

  // ── Request payload validation ─────────────────────────────────────────────

  it('includes provider in the POST /api/keys request body', async () => {
    const user = userEvent.setup();
    const fetchMock = setupFetchMock({
      getKeys: [],
      saveResponse: { id: 'key-new', maskedKey: 'sk-ant-***new' },
    });
    render(<ProviderSelector onChange={vi.fn()} />);

    const toggleBtn = await screen.findByRole('button', { name: /add api key|manage api keys|api key/i });
    await user.click(toggleBtn);

    const input = screen.getByPlaceholderText(/paste your api key/i);
    await user.type(input, 'sk-ant-api123testkey');
    await user.click(screen.getByRole('button', { name: /save key|save/i }));

    await waitFor(() => {
      const postCall = fetchMock.mock.calls.find(
        ([, init]) => (init as RequestInit)?.method === 'POST',
      );
      expect(postCall).toBeDefined();
      const body = JSON.parse(postCall![1]!.body as string);
      expect(body.provider).toBe('anthropic');
    });
  });

  it('shows saved keys specific to the currently selected provider', async () => {
    const user = userEvent.setup();
    setupFetchMock({ getKeys: mockSavedKeys });
    render(<ProviderSelector onChange={vi.fn()} />);

    // Default is anthropic — should show anthropic key only
    await waitFor(() => {
      expect(screen.getByText('sk-ant-***abc (Active)')).toBeInTheDocument();
      expect(screen.queryByText('sk-***xyz (Active)')).not.toBeInTheDocument();
    });

    // Switch provider to openai
    const comboboxes = screen.getAllByRole('combobox');
    await user.selectOptions(comboboxes[0], 'openai');

    await waitFor(() => {
      expect(screen.getByText('sk-***xyz (Active)')).toBeInTheDocument();
      expect(screen.queryByText('sk-ant-***abc (Active)')).not.toBeInTheDocument();
    });
  });
});

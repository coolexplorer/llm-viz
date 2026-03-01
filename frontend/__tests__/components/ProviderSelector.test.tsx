import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ProviderSelector from '@/app/components/ProviderSelector';

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => { store[key] = value; },
    removeItem: (key: string) => { delete store[key]; },
    clear: () => { store = {}; },
  };
})();

Object.defineProperty(window, 'localStorage', { value: localStorageMock });

beforeEach(() => {
  localStorageMock.clear();
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

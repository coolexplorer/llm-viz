import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ChatInput from '@/app/components/ChatInput';

describe('ChatInput', () => {
  it('renders textarea and send button', () => {
    render(<ChatInput onSubmit={vi.fn()} isLoading={false} disabled={false} />);
    expect(screen.getByRole('textbox')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /send/i })).toBeInTheDocument();
  });

  it('shows placeholder text', () => {
    render(<ChatInput onSubmit={vi.fn()} isLoading={false} disabled={false} />);
    expect(screen.getByPlaceholderText(/Send a message to track token usage/i)).toBeInTheDocument();
  });

  it('shows disabled message when disabled prop is true', () => {
    render(<ChatInput onSubmit={vi.fn()} isLoading={false} disabled={true} />);
    expect(screen.getByText(/Enter an API key/i)).toBeInTheDocument();
  });

  it('does not show disabled message when enabled', () => {
    render(<ChatInput onSubmit={vi.fn()} isLoading={false} disabled={false} />);
    expect(screen.queryByText(/Enter an API key/i)).not.toBeInTheDocument();
  });

  it('Send button is disabled when input is empty', () => {
    render(<ChatInput onSubmit={vi.fn()} isLoading={false} disabled={false} />);
    expect(screen.getByRole('button', { name: /send/i })).toBeDisabled();
  });

  it('Send button is enabled after typing', async () => {
    const user = userEvent.setup();
    render(<ChatInput onSubmit={vi.fn()} isLoading={false} disabled={false} />);
    await user.type(screen.getByRole('textbox'), 'Hello');
    expect(screen.getByRole('button', { name: /send/i })).not.toBeDisabled();
  });

  it('calls onSubmit when Send button is clicked', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<ChatInput onSubmit={onSubmit} isLoading={false} disabled={false} />);
    await user.type(screen.getByRole('textbox'), 'Hello');
    await user.click(screen.getByRole('button', { name: /send/i }));
    expect(onSubmit).toHaveBeenCalledWith('Hello');
  });

  it('clears input after submission', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<ChatInput onSubmit={onSubmit} isLoading={false} disabled={false} />);
    const textarea = screen.getByRole('textbox');
    await user.type(textarea, 'Hello');
    await user.click(screen.getByRole('button', { name: /send/i }));
    await waitFor(() => expect((textarea as HTMLTextAreaElement).value).toBe(''));
  });

  it('calls onSubmit when Enter is pressed (without Shift)', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<ChatInput onSubmit={onSubmit} isLoading={false} disabled={false} />);
    const textarea = screen.getByRole('textbox');
    await user.type(textarea, 'Hello');
    await user.keyboard('{Enter}');
    expect(onSubmit).toHaveBeenCalledWith('Hello');
  });

  it('does not call onSubmit when Shift+Enter is pressed', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<ChatInput onSubmit={onSubmit} isLoading={false} disabled={false} />);
    await user.type(screen.getByRole('textbox'), 'Hello');
    await user.keyboard('{Shift>}{Enter}{/Shift}');
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('textarea is disabled when disabled prop is true', () => {
    render(<ChatInput onSubmit={vi.fn()} isLoading={false} disabled={true} />);
    expect(screen.getByRole('textbox')).toBeDisabled();
  });

  it('textarea is disabled when isLoading is true', () => {
    render(<ChatInput onSubmit={vi.fn()} isLoading={true} disabled={false} />);
    expect(screen.getByRole('textbox')).toBeDisabled();
  });

  it('does not call onSubmit with whitespace-only input', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<ChatInput onSubmit={onSubmit} isLoading={false} disabled={false} />);
    const textarea = screen.getByRole('textbox');
    await user.type(textarea, '   ');
    fireEvent.click(screen.getByRole('button'));
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('shows spinner icon when isLoading is true', () => {
    render(<ChatInput onSubmit={vi.fn()} isLoading={true} disabled={false} />);
    // Send button should not show text "Send" when loading
    expect(screen.queryByRole('button', { name: /send/i })).not.toBeInTheDocument();
    // The button itself should still exist
    expect(screen.getByRole('button')).toBeInTheDocument();
  });
});

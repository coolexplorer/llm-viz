'use client';

import { useState, useRef } from 'react';

interface Props {
  onSubmit: (message: string) => Promise<void>;
  isLoading: boolean;
  disabled: boolean;
}

export default function ChatInput({ onSubmit, isLoading, disabled }: Props) {
  const [input, setInput] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleSubmit = async () => {
    const msg = input.trim();
    if (!msg || isLoading || disabled) return;
    setInput('');
    await onSubmit(msg);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      void handleSubmit();
    }
  };

  return (
    <div className="rounded-2xl bg-white/5 border border-white/10 p-4 backdrop-blur-sm">
      <div className="flex gap-3 items-end">
        <textarea
          ref={textareaRef}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Send a message to track token usage…"
          rows={2}
          disabled={disabled || isLoading}
          className="flex-1 resize-none rounded-xl bg-slate-800 border border-slate-700 text-white px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-teal-500 placeholder:text-slate-600 disabled:opacity-50"
        />
        <button
          onClick={() => void handleSubmit()}
          disabled={!input.trim() || isLoading || disabled}
          className="rounded-xl px-5 py-3 text-sm font-semibold bg-teal-600 hover:bg-teal-500 disabled:bg-slate-700 disabled:text-slate-500 text-white transition-colors"
        >
          {isLoading ? (
            <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          ) : (
            'Send'
          )}
        </button>
      </div>
      {disabled && (
        <p className="mt-2 text-xs text-amber-400">Enter an API key above to start sending messages.</p>
      )}
    </div>
  );
}
